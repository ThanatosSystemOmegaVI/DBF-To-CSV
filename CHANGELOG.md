# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.1]

### Added

- `-v` shows the version


## [1.2.0]

### Added

- `-deleted-column` appends an `is_deleted` column holding `1` for a deleted record
  and `0` for the rest. Combined with `-include-deleted` this keeps deleted records
  in the export while letting the consumer filter them, instead of forcing a choice
  between losing them and not being able to tell them apart.

- `-encoding` selects how field bytes are decoded: `cp1252` (default) or `raw` for
  the byte-for-byte behaviour of earlier releases.
- Test suite covering header padding, memo resolution, encoding and the deleted
  marker, including fixtures that reproduce each fixed bug.

### Fixed

- **Memo fields exported their block number instead of their text.** A field of
  type `M` holds only a reference into the sibling `.FPT`/`.DBT` file, which was
  never opened, so a memo column exported as `1`, `2`, `3`. Those files are now
  read and the text is exported. A missing memo file is not an error: the raw
  reference is kept.
- **Field bytes were never decoded.** Output was raw single-byte data, so a CSV
  containing accented characters was not valid UTF-8 and could break consumers on
  import. Bytes are now decoded as Windows-1252, which is what these files use —
  the language driver byte is `0x00` (undeclared), and only this code page yields
  sensible text (`0x80` is `€`, `0xE9`/`0xEB`/`0xF6` are `é`/`ë`/`ö`).
- **Records were read from the wrong offset when the header carried padding.** The
  reader assumed the record area started right after the field terminator instead
  of seeking to the header length. On a Visual FoxPro file with a 263-byte backlink
  block, a two-record file read as 54 garbage records. It now seeks.
- NUL bytes used as field padding are trimmed along with whitespace, so they no
  longer end up inside exported values.

### Changed

- Memo resolution and Windows-1252 decoding change the content of exported columns
  for files that use them. `-encoding raw` restores the old byte handling; memo
  resolution is a bug fix and is not optional.

## [1.1.0]

### Added

- `-include-deleted` to keep records marked as deleted, which were silently
  skipped before.
- `row_index` as the first column, holding the position of the record in the file
  so that find and offset become possible.

## [1.0.0]

### Added

- Convert a DBF file to CSV, on stdout or to a file via `-i`/`-o`.
