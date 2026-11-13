package worker

import (
	"uk.ac.bris.cs/gameoflife/stub" 
	"uk.ac.bris.cs/gameoflife/util"
	"net/rpc"
	"net"
	"log"
	"os"
)



type GolWorker struct{
}

func (e *GolWorker) Compute(req stub.ComputeRequest, resp *stub.ComputeResponse)error{
	newGameCell := make([][]uint8, req.EndY-req.StartY)
	for y := req.StartY; y < req.EndY; y++ {
		newGameCell[y-req.StartY] = make([]uint8, req.ImageWidth)
		for x := 0; x < req.ImageWidth; x++ {
			var currentCell util.Cell
			currentCell.X = x
			currentCell.Y = y
			newGameCell[y-req.StartY][x] = calculateLive(currentCell, req.ImageHeight,req.ImageWidth, req.CurrentMap)
		}
	}
	resp.Calc = newGameCell
	return nil
}

func calculateLive(cell util.Cell, ImageHeight int, ImageWidth int, currentMap [][]uint8) uint8 {
	count := countNeighbours(currentMap, cell, ImageHeight, ImageWidth)

	// Alive: 2 or 3 alive neighbors -> alive
	// Dead: 3 alive neighbors -> alive
	// others -> dead
	if count == 3 || (currentMap[cell.Y][cell.X] != 0 && count == 2) {
		return 255
	}
	return 0
}

func countNeighbours(currentMap [][]uint8, cell util.Cell, ImageHeight int, ImageWidth int) int {
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			// in golang, mod to negative int returns negative result
			ny := (cell.Y + dy + ImageHeight) % ImageHeight
			nx := (cell.X + dx + ImageWidth) % ImageWidth
			if currentMap[ny][nx] != 0 {
				count++
			}
		}
	}
	return count
}

func StartEngine(addr string) error {
	if err := rpc.Register(new(GolWorker)); err != nil {//makes the engine
		return err
	}
	listener, err := net.Listen("tcp", addr)//opens network port. addr=port num
	if err != nil {
		return err
	}
	log.Printf("[worker] Game of Life worker listening on %v", addr)
	rpc.Accept(listener)
	return nil
}

func WorkerAddress()string{
	return os.Getenv("GOL_WORKERS")
}