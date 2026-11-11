package gol

import (
	"fmt"
	"log"
	"net/rpc"
	"sync"
	"time"

	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8

	keyPresses <-chan rune // Newly added channel for key press
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	gameMap := make([][]uint8, p.ImageHeight)
	originalGameMap := make([][]uint8, p.ImageHeight) // Prepare to make paragraph for next phase
	for i := range gameMap {
		gameMap[i] = make([]uint8, p.ImageWidth)
		originalGameMap[i] = make([]uint8, p.ImageWidth)
	}
	c.ioCommand <- ioInput                                            // Set current mode as input
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight) // Read the image
	var alive []util.Cell
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			v := <-c.ioInput //waits for the next byte from IO and stores it in v
			gameMap[y][x] = v
			originalGameMap[y][x] = v
		}
	}
	// Generate initial map
	// alive is a list of living cells
	// TODO: Create a 2D slice to store the world.

	turn := 0
	c.events <- StateChange{turn, Executing}
	c.events <- CellsFlipped{CompletedTurns: 0, Cells: alive}

	/*for turn = 0; turn < p.Turns; turn++ {
		for y := 0; y < p.ImageHeight; y++ {
			for x := 0; x < p.ImageWidth; x++ {
				var currentCell util.Cell
				currentCell.Y = y
				currentCell.X = x
				newGameMap[y][x] = calculateLive(currentCell, p, gameMap)
			}
		}
		gameMap, newGameMap = newGameMap, gameMap
	}*/
	// Shared snapshot for the ticker: latest finished world and its turn number.
	// Like the mutex-protected shared variables in the concurrency labs.
	var mu sync.Mutex
	snapshot := gameMap
	completedTurns := 0

	// Ticker goroutine: every 2 seconds count alive cells in the snapshot
	// and report them, same select pattern as the quiz lab timer.
	done := make(chan struct{})
	tickerStopped := make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		defer close(tickerStopped)

		// Create a new Client for ticker under distributed mode
		var tickerClient *rpc.Client
		if addr := EngineAddress(); addr != "" {
			if cl, err := rpc.Dial("tcp", addr); err == nil {
				tickerClient = cl
				defer tickerClient.Close()
			}
		}

		for {
			select {
			case <-done:
				return
			case <-ticker.C://step 2: reporting arrive cells to the remote.
				if tickerClient != nil {//if client exist
					var resp AliveResponse
					if err := tickerClient.Call("GolEngine.GetAliveCount", AliveRequest{}, &resp); err == nil {
						c.events <- AliveCellsCount{CompletedTurns: resp.Turn, CellsCount: resp.AliveCount}
						continue
					}
				}
				// fallback to local if RPC fails
				mu.Lock()
				world := snapshot
				t := completedTurns
				mu.Unlock()
				count := 0
				for y := range world {
					for x := range world[y] {
						if world[y][x] == 255 {
							count++
						}
					}
				}
				c.events <- AliveCellsCount{CompletedTurns: t, CellsCount: count}
			}
		}
	}()

	var paused, quitRequested bool
	paused = false
	quitRequested = false
	// Distributed step 1: if an engine address is configured, the whole evolution
	// is delegated to a remote machine through one blocking RPC call.
	if addr := EngineAddress(); addr != "" {//if addr finds a port.
		client, err := rpc.Dial("tcp", addr)//connects to the engine.
		if err != nil {//if fail to connect.
			log.Printf("[Controller] engine at %v failed to dial: %v", addr, err)
			close(done)
			<-tickerStopped
			c.events <- StateChange{0, Quitting}
			close(c.events)
			return
		}
		defer func() {
			if err := client.Close(); err != nil {//closes the connection.
				log.Printf("[Controller] failed to close client: %v", err)
			}
		}()

		batchSize := 1

		type batchResult struct {
			world [][]uint8
			turns int
			err   error
		}

		for turn < p.Turns && !quitRequested {//main loop
			if paused {
				key := <-c.keyPresses
				handleKey(key, &paused, &quitRequested, p, c, gameMap, turn, client)
				continue
			}

			// send 1 batch to engine, waiting for results async
			toCompute := batchSize
			if p.Turns-turn < toCompute {
				toCompute = p.Turns - turn//how many turns left
			}
			req := EvolveRequest{World: gameMap, Turns: toCompute, Threads: p.Threads,
				ImageWidth: p.ImageWidth, ImageHeight: p.ImageHeight}
			resultCh := make(chan batchResult, 1)
			go func() {
				var resp EvolveResponse
				err := client.Call("GolEngine.Evolve", req, &resp)
				resultCh <- batchResult{resp.World, resp.CompletedTurns, err}
			}()

			// respond to keyPresses when waiting
		waiting:
			for {
				select {
				case key := <-c.keyPresses:
					handleKey(key, &paused, &quitRequested, p, c, gameMap, turn, client)
					if quitRequested {
						break waiting
					} // no longer waiting for this batch
				case res := <-resultCh:
					if res.err != nil {
						log.Printf("[Controller] Engine call failed: %v", res.err)
						break waiting
					}
					gameMap = res.world
					turn += res.turns
					var flipped []util.Cell
					for y := 0; y < p.ImageHeight; y++ {
						for x := 0; x < p.ImageWidth; x++ {
							if gameMap[y][x] != originalGameMap[y][x] {
								flipped = append(flipped, util.Cell{X: x, Y: y})
							}
						}
					}
					for y := 0; y < p.ImageHeight; y++ {
						for x := 0; x < p.ImageWidth; x++ {
							originalGameMap[y][x] = gameMap[y][x]
						}
					}

					mu.Lock()
					snapshot = gameMap
					completedTurns = turn
					mu.Unlock()
					c.events <- CellsFlipped{CompletedTurns: turn, Cells: flipped}
					c.events <- TurnComplete{CompletedTurns: turn}
					break waiting
				}
			}
		}
	} else {
		for turn < p.Turns && !quitRequested {
			if paused {
				key := <-c.keyPresses
				handleKey(key, &paused, &quitRequested, p, c, gameMap, turn, nil)
				continue
			}
			select {
			case key := <-c.keyPresses:
				handleKey(key, &paused, &quitRequested, p, c, gameMap, turn, nil)
				continue
			default:
			}
			gameMap = nextWorld(p, gameMap)
			var flipped []util.Cell
			for y := 0; y < p.ImageHeight; y++ {
				for x := 0; x < p.ImageWidth; x++ {
					if gameMap[y][x] != originalGameMap[y][x] {
						flipped = append(flipped, util.Cell{X: x, Y: y})
					}
				}
			}
			for y := 0; y < p.ImageHeight; y++ {
				for x := 0; x < p.ImageWidth; x++ {
					originalGameMap[y][x] = gameMap[y][x]
				}
			}
			turn++

			mu.Lock()
			snapshot = gameMap
			completedTurns = turn
			mu.Unlock()

			c.events <- CellsFlipped{CompletedTurns: turn, Cells: flipped}
			c.events <- TurnComplete{CompletedTurns: turn}
		}
	}

	// Stop the ticker before closing events.
	close(done)
	<-tickerStopped

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if gameMap[y][x] != 0 {
				var alive1 util.Cell
				alive1.X = x
				alive1.Y = y
				alive = append(alive, alive1)
			}
		}
	}
	c.events <- FinalTurnComplete{CompletedTurns: turn, Alive: alive}
	// TODO: Report the final state using FinalTurnCompleteEvent.
	if !quitRequested {
		makePGM(p, c, gameMap, turn)
	}
	

	// Make sure that the Io has finished any output before exiting.
	//c.ioCommand <- ioCheckIdle
	//<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func calculateLive(cell util.Cell, p Params, currentMap [][]uint8) uint8 {
	count := countNeighbours(currentMap, cell, p)

	// Alive: 2 or 3 alive neighbors -> alive
	// Dead: 3 alive neighbors -> alive
	// others -> dead
	if count == 3 || (currentMap[cell.Y][cell.X] != 0 && count == 2) {
		return 255
	}
	return 0
}

func countNeighbours(currentMap [][]uint8, cell util.Cell, p Params) int {
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			// in golang, mod to negative int returns negative result
			ny := (cell.Y + dy + p.ImageHeight) % p.ImageHeight
			nx := (cell.X + dx + p.ImageWidth) % p.ImageWidth
			if currentMap[ny][nx] != 0 {
				count++
			}
		}
	}
	return count
}

// nextWorld evolves one turn, splitting the board into p.Threads horizontal bands
// (the same divide-and-conquer pattern used in the concurrency lab's image filter).
func nextWorld(p Params, current [][]uint8) [][]uint8 {
	WorkerRow := p.ImageHeight / p.Threads
	channels := make([]chan [][]uint8, p.Threads)
	for i := 0; i < p.Threads; i++ {
		startY := i * WorkerRow
		endY := (i + 1) * WorkerRow
		if i == p.Threads-1 {
			endY = p.ImageHeight // the last worker also takes the remainder
		}
		channels[i] = make(chan [][]uint8)
		go worker(p, startY, endY, current, channels[i])
	}

	var next [][]uint8
	for i := 0; i < p.Threads; i++ {
		next = append(next, <-channels[i]...)
	}
	return next
}

func worker(p Params, startY, endY int, currentMap [][]uint8, calc chan<- [][]uint8) {
	newGameCell := make([][]uint8, endY-startY)
	for y := startY; y < endY; y++ {
		newGameCell[y-startY] = make([]uint8, p.ImageWidth)
		for x := 0; x < p.ImageWidth; x++ {
			var currentCell util.Cell
			currentCell.X = x
			currentCell.Y = y
			newGameCell[y-startY][x] = calculateLive(currentCell, p, currentMap)
		}
	}
	calc <- newGameCell
}

func makePGM(p Params, c distributorChannels, gameMap [][]uint8, turn int) {
	filename := fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, turn)

	c.ioCommand <- ioOutput
	c.ioFilename <- filename
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- gameMap[y][x]
		}
	}
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- ImageOutputComplete{CompletedTurns: turn, Filename: filename}
}

func handleKey(key rune, paused, quit *bool, p Params, c distributorChannels, gameMap [][]uint8, turn int, client *rpc.Client) {
	switch key {
	case 's':
		makePGM(p, c, gameMap, turn)
	case 'p':
		*paused = !*paused//flips
		if *paused {
			c.events <- StateChange{turn, Paused}
		} else {
			c.events <- StateChange{turn, Executing}
		}
	case 'q':
		*quit = true
	case 'k':
		makePGM(p, c, gameMap, turn)
		if client != nil{//client exists
			var resp AliveResponse
			err := client.Call("GolEngine.CleanUp", AliveRequest{}, &resp)
			if err != nil {log.Printf("[Controller] Engine call failed: %v", err)}
		}
		*quit = true
	}

}
