package main

import (
	"fmt"
	"os"
)

func printErr(msg string) {
	fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m\n", msg)
}

func printNote(msg string) {
	fmt.Fprintf(os.Stderr, "\033[33m%s\033[0m\n", msg)
}
