// Command replay rebuilds a local L2 book from a Binance-style JSONL capture.
// It does not place orders and it does not import the matching hot path's network stack.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/leaf482/toy-exchange/internal/replay"
)

func main() {
	path := flag.String("file", "", "JSONL capture of snapshot and diff lines")
	depth := flag.Int("depth", 5, "price levels to print on each side")
	script := flag.Bool("script", false, "print synthetic limits and cancels")
	flag.Parse()
	if *path == "" {
		fmt.Fprintln(os.Stderr, "missing -file")
		os.Exit(2)
	}
	data, err := os.ReadFile(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var session replay.Session
	synth := replay.NewSynth()
	var actions []replay.Action
	err = session.Replay(data, func() {
		if *script {
			actions = append(actions, synth.Push(session.Book.Bids, session.Book.Asks, int64(session.Book.LastUpdate))...)
		}
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("lastUpdate %d\n", session.Book.LastUpdate)
	fmt.Println("bids")
	printSide(session.Book.BidsDesc(), *depth)
	fmt.Println("asks")
	printSide(session.Book.AsksAsc(), *depth)
	if *script {
		fmt.Println("script")
		for _, a := range actions {
			if a.Kind == replay.ActionCancel {
				fmt.Printf("CANCEL %d %d\n", a.CancelID, a.Time)
				continue
			}
			side := "B"
			if a.Side == replay.SideSell {
				side = "S"
			}
			fmt.Printf("LIMIT %s %d %d %d %d\n", side, a.ID, a.Price, a.Qty, a.Time)
		}
	}
}

func printSide(levels []replay.Level, n int) {
	if n > len(levels) {
		n = len(levels)
	}
	for _, lvl := range levels[:n] {
		fmt.Printf("%d %d\n", lvl.Price, lvl.Qty)
	}
}
