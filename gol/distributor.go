package gol

import (
	"fmt"

	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8

	//keyPresses <-chan uint8 // Newly added channel for key press
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	gameMap := make([][]uint8, p.ImageHeight)
	newGameMap := make([][]uint8, p.ImageHeight) // Prepare to make paragraph for next phase
	for i := range gameMap {
		gameMap[i] = make([]uint8, p.ImageWidth)
		newGameMap[i] = make([]uint8, p.ImageWidth)
	}
	c.ioCommand <- ioInput                                            // Set current mode as input
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight) // Read the image
	var alive []util.Cell
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			v := <-c.ioInput //waits for the next byte from IO and stores it in v
			gameMap[y][x] = v
			newGameMap[y][x] = v
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
	channels := make([]chan [][]uint8, p.Threads)
	WorkerRow := p.ImageHeight / p.Threads
	for turn = 0; turn < p.Turns; turn++{
		for i := 0; i < p.Threads; i++ {
			startY := i * WorkerRow
			endY := (i+1) * WorkerRow
			if i == p.Threads -1 {
				endY = p.ImageHeight
			}
			channels[i] = make(chan [][]uint8)
			go worker(p, startY, endY, gameMap, channels[i])
			/*for y := startY; y < endY; y++{
				for x:= 0; x < p.ImageWidth; x++{
					var currentCell util.Cell
					currentCell.X = x
					currentCell.Y = y
					newGameMap[y][x] = calculateLive(currentCell, p, gameMap)
				}
			}*/
			
		}
		var newworld [][]uint8
		for i := 0; i < p.Threads; i++{
			newworld = append(newworld, <-channels[i]...)
		}
		gameMap = newworld

		//gameMap, newGameMap = newGameMap, gameMap
	}
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
	for y := startY; y < endY; y++{
		newGameCell[y-startY] = make([]uint8, p.ImageWidth)
		for x:= 0; x < p.ImageWidth; x++{
			var currentCell util.Cell
			currentCell.X = x				
			currentCell.Y = y
			newGameCell[y-startY][x] = calculateLive(currentCell, p, currentMap)
		}		
	}
	calc <- newGameCell
}
