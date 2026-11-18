package tests

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/stub"
)

func BenchmarkDistributed(b *testing.B) {
	p := gol.Params{ImageWidth: 512, ImageHeight: 512, Turns: 1000, Threads: 1}
	b.Run(fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, p.Turns), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			events := make(chan gol.Event)
			go gol.Run(p, events, nil)
			for range events {
			}
		}
	})
}

func TestGobSizes(t *testing.T) {
	m := make([][]uint8, 512)
	for i := range m { m[i] = make([]uint8, 512) }
	for _, n := range []int{1, 2, 4, 8} {
		var req, resp bytes.Buffer
		gob.NewEncoder(&req).Encode(stub.ComputeRequest{StartY: 0, EndY: 512 / n, CurrentMap: m, ImageHeight: 512, ImageWidth: 512})
		gob.NewEncoder(&resp).Encode(stub.ComputeResponse{Calc: m[:512/n]})
		t.Logf("N=%d req=%d resp=%d", n, req.Len(), resp.Len())
	}
}
