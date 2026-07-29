package Modules

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// memoFile reads the text behind a memo field. A memo field in the DBF holds only
// a block number; the text itself lives in a sibling .FPT (FoxPro) or .DBT (dBase)
// file. Without it a memo column exports as a meaningless block number.
type memoFile struct {
	f         *os.File
	blockSize int
	foxPro    bool
}

const (
	defaultFptBlockSize = 512
	dbtBlockSize        = 512
	fptBlockHeaderSize  = 8
	fptTypeText         = 1
	dbtTerminator       = 0x1A
)

// openMemoFile looks for the memo file belonging to a DBF. A missing memo file is
// not an error: the caller falls back to exporting the raw field value.
func openMemoFile(dbfPath string) (*memoFile, error) {
	base := strings.TrimSuffix(dbfPath, filepath.Ext(dbfPath))

	for _, candidate := range []struct {
		ext    string
		foxPro bool
	}{
		{".FPT", true}, {".fpt", true}, {".DBT", false}, {".dbt", false},
	} {
		f, err := os.Open(base + candidate.ext)
		if err != nil {
			continue
		}

		memo := &memoFile{f: f, foxPro: candidate.foxPro, blockSize: dbtBlockSize}

		if candidate.foxPro {
			// FPT header: bytes 6-7 hold the block size, big endian
			var header [8]byte
			if _, err := io.ReadFull(f, header[:]); err != nil {
				f.Close()
				return nil, fmt.Errorf("read memo header: %w", err)
			}

			memo.blockSize = int(binary.BigEndian.Uint16(header[6:8]))
			if memo.blockSize <= 0 {
				memo.blockSize = defaultFptBlockSize
			}
		}

		return memo, nil
	}

	return nil, nil
}

func (m *memoFile) Close() error {
	if m == nil || m.f == nil {
		return nil
	}
	return m.f.Close()
}

// text resolves the memo reference stored in a record field. Field values are
// either a right-aligned ASCII block number (10 bytes) or a little endian uint32
// (4 bytes); block 0 means the record has no memo.
func (m *memoFile) text(raw []byte, enc Encoding) (string, error) {
	block, err := memoBlockNumber(raw)
	if err != nil || block <= 0 {
		return "", err
	}

	if m.foxPro {
		return m.foxProText(block, enc)
	}

	return m.dbaseText(block, enc)
}

func memoBlockNumber(raw []byte) (int64, error) {
	if len(raw) == 4 {
		return int64(binary.LittleEndian.Uint32(raw)), nil
	}

	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return 0, nil
	}

	block, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("memo block number %q: %w", trimmed, err)
	}

	return block, nil
}

// foxProText reads one FPT block: a big endian type and length, then the payload.
func (m *memoFile) foxProText(block int64, enc Encoding) (string, error) {
	var header [fptBlockHeaderSize]byte

	if _, err := m.f.ReadAt(header[:], block*int64(m.blockSize)); err != nil {
		return "", fmt.Errorf("read memo block %d: %w", block, err)
	}

	if binary.BigEndian.Uint32(header[0:4]) != fptTypeText {
		return "", nil
	}

	length := int(binary.BigEndian.Uint32(header[4:8]))
	if length <= 0 {
		return "", nil
	}

	payload := make([]byte, length)
	if _, err := m.f.ReadAt(payload, block*int64(m.blockSize)+fptBlockHeaderSize); err != nil {
		return "", fmt.Errorf("read memo block %d payload: %w", block, err)
	}

	return decode(payload, enc), nil
}

// dbaseText reads a .DBT memo, which runs until a 0x1A terminator instead of
// carrying a length.
func (m *memoFile) dbaseText(block int64, enc Encoding) (string, error) {
	var payload []byte
	buf := make([]byte, m.blockSize)

	for offset := block * int64(m.blockSize); ; offset += int64(m.blockSize) {
		n, err := m.f.ReadAt(buf, offset)
		if n == 0 {
			if err != nil && err != io.EOF {
				return "", fmt.Errorf("read memo block %d: %w", block, err)
			}
			break
		}

		if end := indexTerminator(buf[:n]); end >= 0 {
			payload = append(payload, buf[:end]...)
			break
		}

		payload = append(payload, buf[:n]...)

		if err == io.EOF {
			break
		}
	}

	return decode(payload, enc), nil
}

func indexTerminator(b []byte) int {
	for i, c := range b {
		if c == dbtTerminator {
			return i
		}
	}
	return -1
}
