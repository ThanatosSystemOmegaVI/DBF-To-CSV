# DBF-To-CSV
Convert DBF files to CSV

The first column of the output is `row_index`, the position of the record in the
file, numbered from 1. Memo fields (type `M`) are resolved through the sibling
`.FPT`/`.DBT` file. Field bytes are decoded as Windows-1252, so the output is
valid UTF-8.

# usage
```
❯ dbf-reader path/to/finput.DBF > /path/to/output.csv
```
```
❯ dbf-reader -include-deleted /path/to/input.DBF > /path/to/output.csv
```

Keep deleted records and mark them, instead of choosing between dropping them and
not being able to tell them apart:
```
❯ dbf-reader -include-deleted -deleted-column /path/to/input.DBF > /path/to/output.csv
```

Export the raw bytes instead of decoding them, as earlier releases did:
```
❯ dbf-reader -encoding raw /path/to/input.DBF > /path/to/output.csv
```

# options
| flag | default | meaning |
| --- | --- | --- |
| `-i` | | input DBF path (may also be given as a positional argument) |
| `-o` | stdout | output CSV path |
| `-include-deleted` | off | include records marked as deleted (`*`) |
| `-deleted-column` | off | append an `is_deleted` column: `1` when deleted, `0` otherwise |
| `-encoding` | `cp1252` | how to decode field bytes: `cp1252` or `raw` |

# install
- get the .deb, or .rpm from the release tags
- or install using apt-repository:

```
curl -fsSL https://thanatossystemomegavi.github.io/DBF-To-CSV-APT/key.gpg -o /tmp/dbf-reader-key.gpg && sudo gpg --dearmor -o /usr/share/keyrings/dbf-reader.gpg /tmp/dbf-reader-key.gpg && rm /tmp/dbf-reader-key.gpg

sudo sh -c 'echo "deb [signed-by=/usr/share/keyrings/dbf-reader.gpg] https://thanatossystemomegavi.github.io/DBF-To-CSV-APT stable main" > /etc/apt/sources.list.d/dbf-reader.list'

sudo apt update && sudo apt install -y dbf-reader
```

# development
```
❯ go test ./...
❯ go build -o dbf-reader .
```

See [CHANGELOG.md](CHANGELOG.md) for what changed per release.
