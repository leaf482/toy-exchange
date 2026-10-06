// Command replay rebuilds a local L2 book from a Binance-style JSONL capture.
// It does not place orders and it does not import the matching hot path's network stack.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/leaf482/toy-exchange/internal/replay"
)

func main() {
	path := flag.String("file", "", "JSONL capture of snapshot and diff lines")
	depth := flag.Int("depth", 5, "price levels to print on each side")
	flag.Parse()
	if *path == "" {
		fmt.Fprintln(os.Stderr, "missing -file")
		os.Exit(2)
	}
	f, err := os.Open(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	var book replay.Book
	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		msg, err := replay.Decode([]byte(line))
		if err != nil {
			fmt.Fprintf(os.Stderr, "line %d: %v\n", lineNo, err)
			os.Exit(1)
		}
		if err := book.Apply(msg); err != nil {
			fmt.Fprintf(os.Stderr, "line %d: %v\n", lineNo, err)
			os.Exit(1)
		}
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("lastUpdate %d\n", book.LastUpdate)
	fmt.Println("bids")
	printSide(book.BidsDesc(), *depth)
	fmt.Println("asks")
	printSide(book.AsksAsc(), *depth)
}

func printSide(levels []replay.Level, n int) {
	if n > len(levels) {
		n = len(levels)
	}
	for _, lvl := range levels[:n] {
		fmt.Printf("%d %d\n", lvl.Price, lvl.Qty)
	}
}
