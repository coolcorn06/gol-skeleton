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
	// Cells alive in the starting image; SDL needs them before the first turn.
	var initial []util.Cell
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			v := <-c.ioInput //waits for the next byte from IO and stores it in v
			gameMap[y][x] = v
			originalGameMap[y][x] = v
			if v != 0 {
				initial = append(initial, util.Cell{X: x, Y: y})
			}
		}
	}

	turn := 0
	c.events <- CellsFlipped{CompletedTurns: 0, Cells: initial}
	c.events <- StateChange{turn, Executing}

	// Counters shared with the ticker. The board itself is only touched by
	// this goroutine (plus reads by the workers), so it needs no lock.
	var mu sync.Mutex
	aliveCount := len(initial)
	completedTurns := 0

	// Ticker goroutine: every 2 seconds report the current alive count and turn.
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
			case <-ticker.C:
				if tickerClient != nil {
					var resp AliveResponse
					if err := tickerClient.Call("GolEngine.GetAliveCount", AliveRequest{}, &resp); err == nil {
						c.events <- AliveCellsCount{CompletedTurns: resp.Turn, CellsCount: resp.AliveCount}
						continue
					}
				}
				// fallback to local if RPC fails
				mu.Lock()
				count := aliveCount
				t := completedTurns
				mu.Unlock()
				c.events <- AliveCellsCount{CompletedTurns: t, CellsCount: count}
			}
		}
	}()

	var paused, quitRequested bool
	paused = false
	quitRequested = false
	// Distributed step 1: if an engine address is configured, the whole evolution
	// is delegated to a remote machine through one blocking RPC call.
	if addr := EngineAddress(); addr != "" {
		client, err := rpc.Dial("tcp", addr)
		if err != nil {
			log.Printf("[Controller] engine at %v failed to dial: %v", addr, err)
			close(done)
			<-tickerStopped
			c.events <- StateChange{0, Quitting}
			close(c.events)
			return
		}
		defer func() {
			if err := client.Close(); err != nil {
				log.Printf("[Controller] failed to close client: %v", err)
			}
		}()

		batchSize := 1

		type batchResult struct {
			world [][]uint8
			turns int
			err   error
		}

		for turn < p.Turns && !quitRequested {
			if paused {
				key := <-c.keyPresses
				handleKey(key, &paused, &quitRequested, p, c, gameMap, turn)
				continue
			}

			// send 1 batch to engine, waiting for results async
			toCompute := batchSize
			if p.Turns-turn < toCompute {
				toCompute = p.Turns - turn
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
					handleKey(key, &paused, &quitRequested, p, c, gameMap, turn)
					if quitRequested {
						break waiting
					} // no longer waiting for this batch
				case res := <-resultCh:
					if res.err != nil {
						log.Printf("[Controller} Engine call failed: %v", res.err)
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

					count := 0
					for y := range gameMap {
						for x := range gameMap[y] {
							if gameMap[y][x] != 0 {
								count++
							}
						}
					}
					mu.Lock()
					aliveCount = count
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
				handleKey(key, &paused, &quitRequested, p, c, gameMap, turn)
				continue
			}
			select {
			case key := <-c.keyPresses:
				handleKey(key, &paused, &quitRequested, p, c, gameMap, turn)
				continue
			default:
			}
			flipped := playTurn(p, gameMap)
			turn++

			mu.Lock()
			for _, cell := range flipped {
				if gameMap[cell.Y][cell.X] == 255 {
					aliveCount++
				} else {
					aliveCount--
				}
			}
			completedTurns = turn
			mu.Unlock()

			c.events <- CellsFlipped{CompletedTurns: turn, Cells: flipped}
			c.events <- TurnComplete{CompletedTurns: turn}
		}
	}

	// Stop the ticker before closing events.
	close(done)
	<-tickerStopped

	alive := make([]util.Cell, 0)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if gameMap[y][x] != 0 {
				alive = append(alive, util.Cell{X: x, Y: y})
			}
		}
	}
	c.events <- FinalTurnComplete{CompletedTurns: turn, Alive: alive}
	makePGM(p, c, gameMap, turn)

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

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

const flipBuffer = 256

func playTurn(p Params, board [][]uint8) []util.Cell {
	flips := make(chan util.Cell, flipBuffer)
	collected := make(chan []util.Cell, 1)

	go func() {
		var all []util.Cell
		for cell := range flips {
			all = append(all, cell)
		}
		collected <- all
	}()

	var wg sync.WaitGroup
	rows := p.ImageHeight / p.Threads
	for i := 0; i < p.Threads; i++ {
		startY, endY := i*rows, (i+1)*rows
		if i == p.Threads-1 {
			endY = p.ImageHeight // the last worker also takes the remainder
		}
		wg.Add(1)
		go func(startY, endY int) {
			defer wg.Done()
			for y := startY; y < endY; y++ {
				for x := 0; x < p.ImageWidth; x++ {
					cell := util.Cell{X: x, Y: y}
					if calculateLive(cell, p, board) != board[y][x] {
						flips <- cell
					}
				}
			}
		}(startY, endY)
	}
	wg.Wait()
	close(flips)
	all := <-collected

	// Reduce: safe only after every worker has stopped reading the board.
	for _, cell := range all {
		board[cell.Y][cell.X] ^= 255
	}
	return all
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

func handleKey(key rune, paused, quit *bool, p Params, c distributorChannels, gameMap [][]uint8, turn int) {
	switch key {
	case 's':
		makePGM(p, c, gameMap, turn)
	case 'p':
		*paused = !*paused
		if *paused {
			c.events <- StateChange{turn, Paused}
		} else {
			c.events <- StateChange{turn, Executing}
		}
	case 'q':
		*quit = true
	}
}
