package tests

import (
	"fmt"
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
)

func BenchmarkGol(b *testing.B) {
	tests := gol.Params{ImageWidth: 512, ImageHeight: 512, Turns: 1000}
	 for threads := 1; threads <= 16; threads++ {
        tests.Threads = threads
        name := fmt.Sprintf("%dx%dx%d-%d", tests.ImageWidth, tests.ImageHeight, tests.Turns, tests.Threads)
        b.Run(name, func(b *testing.B) {
            for i := 0; i < b.N; i++ {
                events := make(chan gol.Event)
                go gol.Run(tests, events, nil)
                for range events {
                }                      // wait until Run closes the channel
            }
        })
    }
}
