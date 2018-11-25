package main

import (
	"context"
	"log"
	"os"

	"github.com/itchio/mitch"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	app  = kingpin.New("mitch", "mitch is a (m)ock (itch).io server.")
	port int
)

func flags() {
	app.Flag("port", "Port to listen on").Short('p').Default("0").IntVar(&port)
}

func main() {
	flags()

	kingpin.MustParse(app.Parse(os.Args[1:]))

	ctx := context.Background()
	s, err := mitch.NewServer(ctx, mitch.WithPort(port))
	if err != nil {
		panic(err)
	}
	log.Printf("Now listening on %s", s.Address())
	log.Printf("(Ctrl+C to exit)")
	<-ctx.Done()
}
