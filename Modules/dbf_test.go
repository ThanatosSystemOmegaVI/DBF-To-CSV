package Modules

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type testField struct {
	name   string
	typ    byte
	length uint8
}

type testRecord struct {
	deleted bool
	values  [][]byte
}

// buildDBF writes a minimal but valid DBF. headerPadding adds bytes between the
// field terminator and the first record, which is what a Visual FoxPro backlink
// block looks like.
func buildDBF(t *testing.T, path string, fields []testField, records []testRecord, headerPadding int) {
	t.Helper()

	headerLength := 32 + 32*len(fields) + 1 + headerPadding
	recordLength := 1
	for _, f := range fields {
		recordLength += int(f.length)
	}

	out := make([]byte, 32)
	out[0] = 0x03
	out[1], out[2], out[3] = 125, 7, 29
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(records)))
	binary.LittleEndian.PutUint16(out[8:10], uint16(headerLength))
	binary.LittleEndian.PutUint16(out[10:12], uint16(recordLength))

	for _, f := range fields {
		descriptor := make([]byte, 32)
		copy(descriptor, f.name)
		descriptor[11] = f.typ
		descriptor[16] = f.length
		out = append(out, descriptor...)
	}

	out = append(out, 0x0D)
	out = append(out, make([]byte, headerPadding)...)

	for _, rec := range records {
		marker := byte(' ')
		if rec.deleted {
			marker = '*'
		}
		out = append(out, marker)

		for i, f := range fields {
			value := make([]byte, f.length)
			for j := range value {
				value[j] = ' '
			}
			copy(value, rec.values[i])
			out = append(out, value...)
		}
	}

	out = append(out, 0x1A)

	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatalf("write dbf: %v", err)
	}
}

func readAll(t *testing.T, rd *Reader) []Record {
	t.Helper()

	var records []Record
	for {
		_, rec, _, err := rd.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("next: %v", err)
		}
		records = append(records, rec)
	}

	return records
}

// A header with padding after the terminator used to shift every record.
func TestReadsRecordsWhenHeaderHasPadding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "padded.dbf")
	fields := []testField{{"CODE", 'C', 4}}
	records := []testRecord{
		{values: [][]byte{[]byte("AAAA")}},
		{values: [][]byte{[]byte("BBBB")}},
	}

	buildDBF(t, path, fields, records, 263)

	rd, closeFn, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer closeFn()

	got := readAll(t, rd)
	if len(got) != 2 {
		t.Fatalf("got %d records, want 2", len(got))
	}
	if got[0]["CODE"] != "AAAA" || got[1]["CODE"] != "BBBB" {
		t.Fatalf("records misaligned: %q, %q", got[0]["CODE"], got[1]["CODE"])
	}
}
