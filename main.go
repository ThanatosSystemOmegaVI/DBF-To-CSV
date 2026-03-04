package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	dbf "DBFreader/Modules"
)

func usage() {
	fmt.Fprintf(os.Stderr, `dbf-reader: convert .DBF to CSV

Usage:
  dbf-reader [options] <input.DBF>            (writes CSV to stdout)
  dbf-reader -i <input.DBF> -o <output.csv>  (writes CSV to a file)

Examples:
  dbf-reader /path/to/input.DBF > /path/to/output.csv
  dbf-reader -i /path/to/input.DBF -o /path/to/output.csv
  dbf-reader -include-deleted /path/to/input.DBF > /path/to/output.csv
`)
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)

	inFlag := flag.String("i", "", "Input DBF file path (optional if provided as positional arg)")
	outFlag := flag.String("o", "", "Output CSV file path (optional; default is stdout)")
	includeDeleted := flag.Bool("include-deleted", false, "Include records marked as deleted (*)")
	flag.Usage = usage
	flag.Parse()

	// Determine input path
	inPath := *inFlag
	if inPath == "" {
		if flag.NArg() < 1 {
			usage()
			os.Exit(2)
		}
		inPath = flag.Arg(0)
	}

	// Open DBF
	rd, closeFn, err := dbf.Open(inPath)
	if err != nil {
		log.Fatal(err)
	}
	defer closeFn()

	fields := rd.Fields()

	// Output destination
	var out io.Writer = os.Stdout
	var outFile *os.File
	if *outFlag != "" {
		outFile, err = os.Create(*outFlag)
		if err != nil {
			log.Fatal(err)
		}
		defer outFile.Close()
		out = outFile
	}

	// CSV writer
	w := csv.NewWriter(out)
	defer w.Flush()

	// Header
	header := make([]string, len(fields)+1)
	header[0] = "row_index"
	for i, f := range fields {
		header[i+1] = f.Name
	}
	if err := w.Write(header); err != nil {
		log.Fatal(err)
	}

	// Records
	for {
		rownum, rec, deleted, err := rd.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		// Skip deleted/tombstoned lines only if include-deleted is false
		if deleted && !*includeDeleted {
			continue
		}

		row := make([]string, len(fields)+1)
		row[0] = fmt.Sprintf("%d", rownum)

		for i, f := range fields {
			row[i+1] = rec[f.Name]
		}
		if err := w.Write(row); err != nil {
			log.Fatal(err)
		}
	}

	if err := w.Error(); err != nil {
		log.Fatal(err)
	}
}
