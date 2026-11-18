package tests

import (
	"fmt"
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
)

// BenchmarkMapReduce measures one full run of a fixed number of turns.
func BenchmarkMapReduce(b *testing.B) {
	const turns = 100
	for _, size := range []int{64, 256, 512} {
		for _, threads := range []int{1, 2, 4, 8} {
			p := gol.Params{Turns: turns, Threads: threads, ImageWidth: size, ImageHeight: size}
			name := fmt.Sprintf("%dx%dx%d-%d", size, size, turns, threads)
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					events := make(chan gol.Event)
					go gol.Run(p, events, nil)
					for range events {
					}
				}
			})
		}
	}
}
