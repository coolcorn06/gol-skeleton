package gol

import (
	"log"
	"net"
	"net/rpc"
	"os"
	"sync"
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

type EngineState struct {
	mu         sync.Mutex
	aliveCount int
	turn       int
}

var engineState = &EngineState{}

type AliveRequest struct{}
type AliveResponse struct {
	AliveCount int
	Turn       int
}

func (e *GolEngine) GetAliveCount(req AliveRequest, resp *AliveResponse) error {
	engineState.mu.Lock()
	resp.AliveCount = engineState.aliveCount
	resp.Turn = engineState.turn
	engineState.mu.Unlock()
	return nil
}

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

	world := req.World
	for turn := 0; turn < p.Turns; turn++ {
		world = nextWorld(p, world)

		n := 0
		for y := range world {
			for x := range world[y] {
				if world[y][x] != 0 {
					n++
				}
			}
		}
		engineState.mu.Lock()
		engineState.aliveCount = n
		engineState.turn = turn + 1
		engineState.mu.Unlock()
	}

	resp.World = world
	resp.CompletedTurns = p.Turns
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
	log.Printf("[Engine] Game of Life engine listening on %v", addr)
	rpc.Accept(listener)
	return nil
}

// EngineAddress returns the engine address from GOL_ENGINE, or "" to run locally.
// An environment variable is used because the tests call gol.Run directly and
// never go through main, so a command line flag could not reach them.
func EngineAddress() string {
	return os.Getenv("GOL_ENGINE")
}

// evolveRemotely asks the engine to compute all turns and returns the final world.
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
