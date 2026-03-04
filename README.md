# DBF-To-CSV
Convert DBF files to CSV


# usage
```
❯ dbf-reader path/to/finput.DBF > /path/to/output.csv
```
```
❯ dbf-reader -include-deleted /path/to/input.DBF > /path/to/output.csv
```

# install
- get the .deb, or .rpm from the release tags
- or install using apt-repository:

```
curl -fsSL https://thanatossystemomegavi.github.io/DBF-To-CSV-APT/key.gpg -o /tmp/dbf-reader-key.gpg && sudo gpg --dearmor -o /usr/share/keyrings/dbf-reader.gpg /tmp/dbf-reader-key.gpg && rm /tmp/dbf-reader-key.gpg

sudo sh -c 'echo "deb [signed-by=/usr/share/keyrings/dbf-reader.gpg] https://thanatossystemomegavi.github.io/DBF-To-CSV-APT stable main" > /etc/apt/sources.list.d/dbf-reader.list'

sudo apt update && sudo apt install -y dbf-reader
```

