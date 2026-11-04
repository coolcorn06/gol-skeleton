package gol

import (
	"fmt"
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
	for i := range gameMap {
		gameMap[i] = make([]uint8, p.ImageWidth)
	}
	c.ioCommand <- ioInput                                            // Set current mode as input
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight) // Read the image
	var alive []util.Cell
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			v := <-c.ioInput //waits for the next byte from IO and stores it in v
			gameMap[y][x] = v
		}
	}
	// Generate initial map
	// alive is a list of living cells
	// TODO: Create a 2D slice to store the world.

	turn := 0
	c.events <- CellsFlipped{CompletedTurns: 0, Cells: alive}
	c.events <- StateChange{turn, Executing}
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
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
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

	channels := make([]chan [][]uint8, p.Threads)
	WorkerRow := p.ImageHeight / p.Threads
	for turn < p.Turns && !quitRequested { // Be aware, the condition is modified
		// When pausing, "s" and "q" can still work normally
		if paused {
			key := <-c.keyPresses
			handleKey(key, &paused, &quitRequested, p, c, gameMap, turn)
			continue
		}

		// When not pausing, check if there is keyPresses required to process
		select {
		case key := <-c.keyPresses:
			handleKey(key, &paused, &quitRequested, p, c, gameMap, turn)
			continue
		default:
		}
		for i := 0; i < p.Threads; i++ {
			startY := i * WorkerRow
			endY := (i + 1) * WorkerRow
			if i == p.Threads-1 {
				endY = p.ImageHeight
			}
			channels[i] = make(chan [][]uint8)
			go worker(p, startY, endY, gameMap, channels[i])

		} // execute current turn
		var newworld [][]uint8
		for i := 0; i < p.Threads; i++ {
			newworld = append(newworld, <-channels[i]...)
		}
		gameMap = newworld
		turn++ //Respond to modification of condition of for loop

		// Publish the finished world for the ticker and report the completed turn.
		mu.Lock()
		snapshot = gameMap
		completedTurns = turn + 1
		mu.Unlock()
		c.events <- CellsFlipped{CompletedTurns: turn, Cells: alive}
		c.events <- TurnComplete{CompletedTurns: turn + 1}

		//gameMap, newGameMap = newGameMap, gameMap
	}
	// Stop the ticker before closing events.
	close(done)
	<-tickerStopped
	// TODO: Execute all turns of the Game of Life.

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
	makePGM(p, c, gameMap, turn)
	// TODO: Report the final state using FinalTurnCompleteEvent.

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func calculateLive(cell util.Cell, p Params, currentMap [][]uint8) uint8 { //Decide stat of each cell in next turn
	var hasLeft, hasRight, hasUp, hasDown bool
	hasLeft, hasRight, hasUp, hasDown = true, true, true, true //Detect if that cell has neighbors on that direction
	var count int                                              //Detect neighboring alive cells
	if cell.X == 0 {
		hasLeft = false
	}
	if cell.X == p.ImageWidth-1 {
		hasRight = false
	}
	if cell.Y == 0 {
		hasUp = false
	}
	if cell.Y == p.ImageHeight-1 {
		hasDown = false
	}

	if hasLeft && hasUp {
		if currentMap[cell.Y-1][cell.X-1] != 0 {
			count++
		}
	}
	if hasUp {
		if currentMap[cell.Y-1][cell.X] != 0 {
			count++
		}
	}
	if hasUp && hasRight {
		if currentMap[cell.Y-1][cell.X+1] != 0 {
			count++
		}
	}
	if hasLeft {
		if currentMap[cell.Y][cell.X-1] != 0 {
			count++
		}
	}
	if hasRight {
		if currentMap[cell.Y][cell.X+1] != 0 {
			count++
		}
	}
	if hasLeft && hasDown {
		if currentMap[cell.Y+1][cell.X-1] != 0 {
			count++
		}
	}
	if hasDown {
		if currentMap[cell.Y+1][cell.X] != 0 {
			count++
		}
	}
	if hasRight && hasDown {
		if currentMap[cell.Y+1][cell.X+1] != 0 {
			count++
		}
	}
	if !hasLeft {
		if currentMap[cell.Y][p.ImageWidth-1] != 0 {
			count++
		}
	}
	if !hasUp {
		if currentMap[p.ImageHeight-1][cell.X] != 0 {
			count++
		}
	}
	if !hasRight {
		if currentMap[cell.Y][0] != 0 {
			count++
		}
	}
	if !hasDown {
		if currentMap[0][cell.X] != 0 {
			count++
		}
	}
	if !hasLeft && !hasUp {
		if currentMap[p.ImageHeight-1][p.ImageWidth-1] != 0 {
			count++
		}
	}
	if !hasLeft && !hasDown {
		if currentMap[0][p.ImageWidth-1] != 0 {
			count++
		}
	}
	if !hasRight && !hasUp {
		if currentMap[p.ImageHeight-1][0] != 0 {
			count++
		}
	}
	if !hasRight && !hasDown {
		if currentMap[0][0] != 0 {
			count++
		}
	}
	if !hasLeft && hasUp {
		if currentMap[cell.Y-1][p.ImageWidth-1] != 0 {
			count++
		}
	}
	if !hasLeft && hasDown {
		if currentMap[cell.Y+1][p.ImageWidth-1] != 0 {
			count++
		}
	}
	if !hasRight && hasUp {
		if currentMap[cell.Y-1][0] != 0 {
			count++
		}
	}
	if !hasRight && hasDown {
		if currentMap[cell.Y+1][0] != 0 {
			count++
		}
	}
	if !hasUp && hasLeft {
		if currentMap[p.ImageHeight-1][cell.X-1] != 0 {
			count++
		}
	}
	if !hasUp && hasRight {
		if currentMap[p.ImageHeight-1][cell.X+1] != 0 {
			count++
		}
	}
	if !hasDown && hasLeft {
		if currentMap[0][cell.X-1] != 0 {
			count++
		}
	}
	if !hasDown && hasRight {
		if currentMap[0][cell.X+1] != 0 {
			count++
		}
	}
	//count the number of surrounding live cells

	if currentMap[cell.Y][cell.X] != 0 {
		if count == 2 || count == 3 {
			return 255
		}
		return 0
	}
	if count == 3 {
		return 255
	}
	return 0 //final decision of current cell, with "else" omitted
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
