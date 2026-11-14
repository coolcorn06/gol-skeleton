package gol

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strings"
	"sync"
	"time"

	"uk.ac.bris.cs/gameoflife/stub"
)

// EvolveRequest is what the local controller sends to the remote engine:
// the initial world plus everything the engine needs to evolve it.
type EvolveRequest struct {
	World [][]byte
	Turns int
	//StartTurn   int // turns already completed before this batch
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
type GolBroker struct{}

type BrokerState struct {
	mu         sync.Mutex
	cond       *sync.Cond
	aliveCount int
	turn       int
	world      [][]byte
	paused     bool
	workers    []*rpc.Client
	stopped    bool
}

// StopEvolve breaks the Evolve
func (e *GolBroker) StopEvolve(req AliveRequest, resp AliveRequest) error {
	brokerState.mu.Lock()
	brokerState.stopped = true
	brokerState.mu.Unlock()
	brokerState.cond.Broadcast() // wake up "paused" Evolve
	return nil
}

var brokerState = &BrokerState{}
var workers []*rpc.Client

type AliveRequest struct{}
type AliveResponse struct {
	AliveCount int
	Turn       int
}

type PauseResponse struct {
	Paused bool
	Turn   int
}

func (e *GolBroker) GetAliveCount(req AliveRequest, resp *AliveResponse) error {
	brokerState.mu.Lock()
	resp.AliveCount = brokerState.aliveCount
	resp.Turn = brokerState.turn
	brokerState.mu.Unlock()
	return nil
}

func (e *GolBroker) CleanUp(req AliveRequest, resp *EvolveResponse) error {
	go func() {
		time.Sleep(2 * time.Microsecond)
		os.Exit(0)
	}()
	resp.World = brokerState.world
	resp.CompletedTurns = brokerState.turn
	return nil
}

func connectToWorkers(ImageHeight int, ImageWidth int, currentmap [][]uint8) [][]uint8 {
	Threads := len(workers)
	WorkerRow := ImageHeight / Threads
	channels := make([]chan [][]uint8, Threads)
	for i := 0; i < Threads; i++ {
		startY := i * WorkerRow
		endY := (i + 1) * WorkerRow
		if i == Threads-1 {
			endY = ImageHeight // the last worker also takes the remainder
		}
		channels[i] = make(chan [][]uint8)

		go func(i int, req stub.ComputeRequest) {

			var resp stub.ComputeResponse
			err := workers[i].Call(("GolWorker.Compute"), req, &resp)
			if err != nil {
				log.Printf("[Broker] worker %d failed: %v", i, err)
				channels[i] <- nil // mark as failed
				return
			}
			channels[i] <- resp.Calc
		}(i, stub.ComputeRequest{StartY: startY, EndY: endY, ImageHeight: ImageHeight, ImageWidth: ImageWidth, CurrentMap: currentmap})

	}

	var next [][]uint8
	for i := 0; i < Threads; i++ {
		next = append(next, <-channels[i]...)
	}
	return next
}

// Evolve processes every requested turn in a single blocking call (distributed step 1).
// calculates all cells for all turns and returns final world and the num of turns.
func (e *GolBroker) Evolve(req EvolveRequest, resp *EvolveResponse) error {
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
		brokerState.mu.Lock()
		for brokerState.paused && !brokerState.stopped {
			brokerState.cond.Wait()
		}
		stopped := brokerState.stopped
		brokerState.mu.Unlock()
		if stopped {
			break
		}

		newworld := connectToWorkers(p.ImageHeight, p.ImageWidth, world)
		world = newworld
		//world = nextWorld(p, world)

		n := 0
		for y := range world {
			for x := range world[y] {
				if world[y][x] != 0 {
					n++
				}
			}
		}
		brokerState.mu.Lock()
		brokerState.aliveCount = n
		brokerState.turn = turn + 1
		brokerState.world = world
		brokerState.mu.Unlock()
	}

	resp.World = world
	resp.CompletedTurns = p.Turns
	return nil
}

// StartEngine registers the engine and serves RPC requests until the process exits.
func StartBroker(addr string) error {
	//brokerState.cond = sync.NewCond(&brokerState.mu)
	//workeraddress := strings.Split(os.Getenv("GOL_WORKERS"), ",")
	raw := os.Getenv("GOL_WORKERS")
	if raw == "" {
		return fmt.Errorf("GOL_WORKERS is not set")
	}
	for _, waddr := range strings.Split(raw, ",") {
		waddr = strings.TrimSpace(waddr)
		if waddr == "" {
			continue
		}
		client, err := rpc.Dial("tcp", waddr)
		if err != nil {
			return err
		}
		workers = append(workers, client)
	}
	if err := rpc.Register(new(GolBroker)); err != nil { //makes the engine
		return err
	}
	listener, err := net.Listen("tcp", addr) //opens network port. addr=port num
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
	return os.Getenv("GOL_BROKER")
}

func (e *GolBroker) Screenshot(req AliveRequest, resp *EvolveResponse) error {
	brokerState.mu.Lock()
	resp.World = brokerState.world
	resp.CompletedTurns = brokerState.turn
	brokerState.mu.Unlock()
	return nil
}

func (e *GolBroker) Pause(req AliveRequest, resp *PauseResponse) error {
	brokerState.mu.Lock()
	brokerState.paused = !brokerState.paused
	if !brokerState.paused {
		brokerState.cond.Broadcast()
	}
	resp.Turn = brokerState.turn
	resp.Paused = brokerState.paused
	brokerState.mu.Unlock()

	return nil
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
	if err := client.Call("GolBroker.Evolve", request, &response); err != nil {
		return nil, 0, err
	}
	return response.World, response.CompletedTurns, nil
}
