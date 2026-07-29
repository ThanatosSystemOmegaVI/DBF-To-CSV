package main

import (
	"fmt"
	"os"

	"DBFreader/internal/version"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		fmt.Fprintf(os.Stderr, "Usage: dbf-reader-v\n\nPrint the installed dbf-reader version.\n")
		os.Exit(0)
	}

	fmt.Println(version.String())
}
