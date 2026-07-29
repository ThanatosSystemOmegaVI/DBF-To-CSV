package Modules

import "strings"

// Encoding selects how raw field bytes are turned into text.
type Encoding int

const (
	// EncodingWindows1252 decodes bytes as Windows-1252, the code page these files
	// actually use. The language driver byte is 0x00 (undeclared) in the Gawin
	// exports, so it cannot be trusted; 0x80 decodes to € and 0xE9/0xEB/0xF6 to
	// é/ë/ö, which is the only reading that yields sensible Dutch text.
	EncodingWindows1252 Encoding = iota

	// EncodingRaw passes bytes through untouched, matching the behaviour of
	// releases before 1.1.0. Output is then not valid UTF-8.
	EncodingRaw
)

// windows1252Upper maps the 0x80-0x9F range, the only part where Windows-1252
// differs from Latin-1. A byte without an assigned character keeps its Latin-1
// code point rather than becoming a replacement character, so nothing is lost.
var windows1252Upper = [32]rune{
	0x20AC, 0x81, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0x8D, 0x017D, 0x8F,
	0x90, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0x9D, 0x017E, 0x0178,
}

// ParseEncoding maps a command line value to an Encoding.
func ParseEncoding(name string) (Encoding, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "cp1252", "windows-1252", "windows1252":
		return EncodingWindows1252, true
	case "raw", "none":
		return EncodingRaw, true
	default:
		return EncodingWindows1252, false
	}
}

// decode turns raw field bytes into a string using the reader's encoding.
func decode(raw []byte, enc Encoding) string {
	if enc == EncodingRaw {
		return string(raw)
	}

	ascii := true
	for _, b := range raw {
		if b > 0x7F {
			ascii = false
			break
		}
	}
	if ascii {
		return string(raw)
	}

	var sb strings.Builder
	sb.Grow(len(raw))
	for _, b := range raw {
		switch {
		case b < 0x80:
			sb.WriteByte(b)
		case b < 0xA0:
			sb.WriteRune(windows1252Upper[b-0x80])
		default:
			sb.WriteRune(rune(b))
		}
	}

	return sb.String()
}
