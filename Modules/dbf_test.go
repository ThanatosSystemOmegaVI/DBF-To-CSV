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

// Field bytes are Windows-1252: 0x80 is the euro sign, not a control character.
func TestDecodesWindows1252(t *testing.T) {
	path := filepath.Join(t.TempDir(), "encoding.dbf")

	buildDBF(t, path, []testField{{"OMS", 'C', 12}}, []testRecord{
		{values: [][]byte{{'P', 'r', 'i', 'j', 's', ' ', 0x80, '3', '6', ',', 0xE9}}},
	}, 0)

	rd, closeFn, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer closeFn()

	if got := readAll(t, rd)[0]["OMS"]; got != "Prijs €36,é" {
		t.Fatalf("got %q, want %q", got, "Prijs €36,é")
	}
}

func TestRawEncodingLeavesBytesUntouched(t *testing.T) {
	path := filepath.Join(t.TempDir(), "raw.dbf")

	buildDBF(t, path, []testField{{"OMS", 'C', 2}}, []testRecord{
		{values: [][]byte{{0xE9, 'x'}}},
	}, 0)

	rd, closeFn, err := OpenWithEncoding(path, EncodingRaw)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer closeFn()

	if got := readAll(t, rd)[0]["OMS"]; got != "\xe9x" {
		t.Fatalf("got %q, want the undecoded bytes", got)
	}
}

func TestParseEncoding(t *testing.T) {
	for _, tc := range []struct {
		in    string
		want  Encoding
		valid bool
	}{
		{"", EncodingWindows1252, true},
		{"cp1252", EncodingWindows1252, true},
		{"Windows-1252", EncodingWindows1252, true},
		{"raw", EncodingRaw, true},
		{"utf8", EncodingWindows1252, false},
	} {
		got, ok := ParseEncoding(tc.in)
		if ok != tc.valid || got != tc.want {
			t.Fatalf("ParseEncoding(%q) = %v, %v; want %v, %v", tc.in, got, ok, tc.want, tc.valid)
		}
	}
}

// buildFPT writes a FoxPro memo file whose block n holds texts[n].
func buildFPT(t *testing.T, path string, blockSize int, texts map[int]string) {
	t.Helper()

	highest := 0
	for block := range texts {
		if block > highest {
			highest = block
		}
	}

	out := make([]byte, blockSize*(highest+1))
	binary.BigEndian.PutUint32(out[0:4], uint32(highest+1))
	binary.BigEndian.PutUint16(out[6:8], uint16(blockSize))

	for block, text := range texts {
		offset := block * blockSize
		binary.BigEndian.PutUint32(out[offset:offset+4], fptTypeText)
		binary.BigEndian.PutUint32(out[offset+4:offset+8], uint32(len(text)))
		copy(out[offset+8:], text)
	}

	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatalf("write fpt: %v", err)
	}
}

// A memo field must export its text, not the block number it references.
func TestMemoFieldResolvesToText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "memo.dbf")

	buildDBF(t, path, []testField{{"ARTOMS", 'M', 10}}, []testRecord{
		{values: [][]byte{[]byte("         1")}},
		{values: [][]byte{[]byte("         2")}},
		{values: [][]byte{[]byte("          ")}},
	}, 0)
	buildFPT(t, filepath.Join(dir, "memo.FPT"), 512, map[int]string{
		1: "Bevestiging sturen naar info@example.nl",
		2: "Zie bijlage voor wienersprosse",
	})

	rd, closeFn, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer closeFn()

	got := readAll(t, rd)
	if got[0]["ARTOMS"] != "Bevestiging sturen naar info@example.nl" {
		t.Fatalf("first memo = %q", got[0]["ARTOMS"])
	}
	if got[1]["ARTOMS"] != "Zie bijlage voor wienersprosse" {
		t.Fatalf("second memo = %q", got[1]["ARTOMS"])
	}
	if got[2]["ARTOMS"] != "" {
		t.Fatalf("empty memo reference = %q, want empty", got[2]["ARTOMS"])
	}
}

// Without a memo file the raw reference is kept rather than failing the export.
func TestMemoFieldWithoutMemoFileKeepsRawValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nomemo.dbf")

	buildDBF(t, path, []testField{{"ARTOMS", 'M', 10}}, []testRecord{
		{values: [][]byte{[]byte("         7")}},
	}, 0)

	rd, closeFn, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer closeFn()

	if got := readAll(t, rd); got[0]["ARTOMS"] != "7" {
		t.Fatalf("got %q, want the raw reference 7", got[0]["ARTOMS"])
	}
}
