// Command replay rebuilds a local L2 book from a Binance-style JSONL capture.
// -match runs the finished synthetic script through the matcher.
// The engine package does not import a network stack.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/leaf482/toy-exchange/internal/engine"
	"github.com/leaf482/toy-exchange/internal/replay"
	"github.com/leaf482/toy-exchange/pkg/events"
)

func main() {
	path := flag.String("file", "", "JSONL capture of snapshot and diff lines")
	depth := flag.Int("depth", 5, "price levels to print on each side")
	script := flag.Bool("script", false, "print synthetic limits and cancels")
	match := flag.Bool("match", false, "run the synthetic script through the matcher")
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

	book, actions, err := replay.Play(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("lastUpdate %d\n", book.LastUpdate)
	fmt.Println("bids")
	printSide(book.BidsDesc(), *depth)
	fmt.Println("asks")
	printSide(book.AsksAsc(), *depth)
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
	if *match {
		if err := printMatch(actions, *depth); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func printMatch(actions []replay.Action, n int) error {
	ops := replay.EngineOps(actions)
	capN := len(ops) + 1
	ob, err := engine.New(capN, capN, capN)
	if err != nil {
		return err
	}
	dst := make([]events.Event, 0, events.MaxEventsFor(int32(capN)))
	for _, op := range ops {
		if _, err := ob.Apply(op, dst[:0]); err != nil {
			return err
		}
	}
	fmt.Println("engine")
	fmt.Println("bids")
	printEngine(ob, engine.SideBuy, n)
	fmt.Println("asks")
	printEngine(ob, engine.SideSell, n)
	return nil
}

func printEngine(ob *engine.OrderBook, side engine.Side, n int) {
	view := ob.Depth(side, n, make([]engine.LevelView, n))
	for _, lvl := range view {
		fmt.Printf("%d %d %d\n", lvl.Price, lvl.Qty, lvl.Count)
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
