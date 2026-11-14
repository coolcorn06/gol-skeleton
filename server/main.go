//it starts the broker.
package main

import (
	"log"
	"uk.ac.bris.cs/gameoflife/gol"
)

func main() {
	if err := gol.StartBroker(":8030"); err != nil {
		log.Fatal(err)
	}
}