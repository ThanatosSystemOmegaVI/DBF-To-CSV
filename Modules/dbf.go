package Modules

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type Field struct {
	Name     string
	Type     byte
	Length   uint8
	Decimals uint8
	Offset   int // offset within the record payload (after deletion flag)
}

type Header struct {
	Version      byte
	LastUpdate   time.Time
	NumRecords   uint32
	HeaderLength uint16
	RecordLength uint16
}

type Reader struct {
	r      *bufio.Reader
	hdr    Header
	fields []Field
}

func Open(path string) (*Reader, func() error, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	closer := func() error { return f.Close() }

	br := bufio.NewReaderSize(f, 256*1024)
	rd := &Reader{r: br}

	if err := rd.readHeader(); err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	if err := rd.readFields(); err != nil {
		_ = f.Close()
		return nil, nil, err
	}

	// Ensure we are positioned at start of record area.
	// HeaderLength counts from file start to first record.
	// We've read some bytes already; simplest is to seek, but we’re using bufio.
	// So: reopen with os.File and Seek in a different constructor if you want perfect positioning.
	// This minimal version assumes we’ve consumed exactly headerLength bytes after readFields.
	// (readFields reads until 0x0D and then consumes the header terminator; matches typical DBF.)

	return rd, closer, nil
}

func (rd *Reader) Header() Header     { return rd.hdr }
func (rd *Reader) Fields() []Field    { return append([]Field(nil), rd.fields...) }
func (rd *Reader) NumRecords() uint32 { return rd.hdr.NumRecords }

func (rd *Reader) readHeader() error {
	// 32-byte header
	var b [32]byte
	if _, err := io.ReadFull(rd.r, b[:]); err != nil {
		return fmt.Errorf("read header: %w", err)
	}

	rd.hdr.Version = b[0]

	// last update: YY MM DD, with YY = years since 1900 (common)
	yy := int(b[1]) + 1900
	mm := time.Month(b[2])
	dd := int(b[3])
	rd.hdr.LastUpdate = time.Date(yy, mm, dd, 0, 0, 0, 0, time.UTC)

	rd.hdr.NumRecords = binary.LittleEndian.Uint32(b[4:8])
	rd.hdr.HeaderLength = binary.LittleEndian.Uint16(b[8:10])
	rd.hdr.RecordLength = binary.LittleEndian.Uint16(b[10:12])

	return nil
}

func (rd *Reader) readFields() error {
	// Field descriptors are 32 bytes each until 0x0D.
	offset := 0
	for {
		peek, err := rd.r.Peek(1)
		if err != nil {
			return fmt.Errorf("peek field terminator: %w", err)
		}
		if peek[0] == 0x0D {
			// consume terminator
			_, _ = rd.r.ReadByte()
			// there may be 0x00 padding to reach header length; ignore here
			return nil
		}

		var d [32]byte
		if _, err := io.ReadFull(rd.r, d[:]); err != nil {
			return fmt.Errorf("read field descriptor: %w", err)
		}

		// name: 11 bytes, null-terminated
		nameBytes := d[0:11]
		if i := bytes.IndexByte(nameBytes, 0x00); i >= 0 {
			nameBytes = nameBytes[:i]
		}
		name := strings.TrimSpace(string(nameBytes))

		typ := d[11]
		length := d[16]
		dec := d[17]

		rd.fields = append(rd.fields, Field{
			Name:     name,
			Type:     typ,
			Length:   length,
			Decimals: dec,
			Offset:   offset,
		})
		offset += int(length)
	}
}

// Record is a raw record payload, per field.
type Record map[string]string

// Next reads the next record. It returns (nil, io.EOF) when done.
func (rd *Reader) Next() (Record, bool, error) {
	// One record is RecordLength bytes. First byte is deletion flag.
	recLen := int(rd.hdr.RecordLength)
	if recLen <= 1 {
		return nil, false, fmt.Errorf("invalid record length: %d", recLen)
	}

	buf := make([]byte, recLen)
	_, err := io.ReadFull(rd.r, buf)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, false, io.EOF
		}
		return nil, false, fmt.Errorf("read record: %w", err)
	}

	deleted := buf[0] == '*'
	payload := buf[1:]

	out := make(Record, len(rd.fields))
	for _, f := range rd.fields {
		start := f.Offset
		end := start + int(f.Length)
		if end > len(payload) || start < 0 {
			return nil, deleted, fmt.Errorf("field %s out of bounds", f.Name)
		}
		raw := payload[start:end]

		// For a first version: return trimmed string; type-specific parsing can be layered on.
		out[f.Name] = strings.TrimSpace(string(raw))
	}

	return out, deleted, nil
}
