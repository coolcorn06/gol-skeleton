package gol

import (
	"log"
	"net"
	"net/rpc"
	"os"
	"time"
)

// EvolveRequest is what the local controller sends to the remote engine:
// the initial world plus everything the engine needs to evolve it.
type EvolveRequest struct {
	World       [][]byte
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

// EvolveResponse is the final world returned by the engine after all turns.
type EvolveResponse struct {
	World          [][]byte
	CompletedTurns int
}

// GolEngine is the RPC service that evolves Game of Life on a remote machine.
type GolEngine struct{}

// Evolve processes every requested turn in a single blocking call (distributed step 1).
func (e *GolEngine) Evolve(req EvolveRequest, resp *EvolveResponse) error {
	p := Params{
		Turns:       req.Turns,
		Threads:     req.Threads,
		ImageWidth:  req.ImageWidth,
		ImageHeight: req.ImageHeight,
	}
	if p.Threads < 1 {
		p.Threads = 1
	}
	start := time.Now()
	log.Printf("[AWS] request received: %dx%d, turns=%d, threads=%d",
		req.ImageWidth, req.ImageHeight, req.Turns, p.Threads)
	world := req.World
	for turn := 0; turn < p.Turns; turn++ {
		world = nextWorld(p, world)
	}
	resp.World = world
	resp.CompletedTurns = p.Turns
	log.Printf("[AWS] request done: %d turns in %v", p.Turns, time.Since(start))
	return nil
}

// StartEngine registers the engine and serves RPC requests until the process exits.
func StartEngine(addr string) error {
	if err := rpc.Register(new(GolEngine)); err != nil {
		return err
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Printf("[AWS] engine listening on %v", addr)
	rpc.Accept(listener)
	return nil
}

// EngineAddress returns the engine address from GOL_ENGINE, or "" to run locally.
// An environment variable is used because the tests call gol.Run directly and
// never go through main, so a command line flag could not reach them.
func EngineAddress() string {
	return os.Getenv("GOL_ENGINE")
}

func evolveRemotely(addr string, p Params, world [][]uint8) ([][]uint8, int, error) {
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		return nil, 0, err
	}
	defer client.Close()
	request := EvolveRequest{
		World:       world,
		Turns:       p.Turns,
		Threads:     p.Threads,
		ImageWidth:  p.ImageWidth,
		ImageHeight: p.ImageHeight,
	}
	var response EvolveResponse
	if err := client.Call("GolEngine.Evolve", request, &response); err != nil {
		return nil, 0, err
	}
	return response.World, response.CompletedTurns, nil
}
