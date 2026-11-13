package main
import (
	"flag"
	"log"
)

func main(){
	port := flag.String("port", "8040", "port to listen on")
	flag.Parse()
	if err := StartWorker(":" + *port); err != nil{
		log.Fatal(err)
	}
}