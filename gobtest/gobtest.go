package main

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"strings"
)

type butlerPacket struct {
	Kind   int
	Perc   float32
	Buffer []byte
	Sub    interface{}
}

type butlerSubPacket struct {
	Message string
}

func (bp butlerPacket) String() string {
	return fmt.Sprintf("Kind (%d) Perc (%.2f%%) Buffer (%x) Sub (%s)", bp.Kind, bp.Perc, bp.Buffer, bp.Sub)
}

func (bsp butlerSubPacket) String() string {
	return fmt.Sprintf("[Message (%s)]", bsp.Message)
}

func main() {
	pr, pw := io.Pipe()

	gob.Register(butlerPacket{})
	gob.Register(butlerSubPacket{})
	genc := gob.NewEncoder(pw)

	go func() {
		defer pw.Close()
		for i := 1; i <= 3; i++ {
			err := genc.Encode(butlerPacket{
				Kind:   i,
				Perc:   41.07 + 2.03*float32(i),
				Buffer: []byte{0x1, 0x5, byte(0x78 + i)},
				Sub:    butlerSubPacket{Message: strings.Repeat("oh", i)},
			})
			if err != nil {
				panic(err)
			}
		}
	}()

	gdec := gob.NewDecoder(pr)

	for {
		var sdt butlerPacket

		err := gdec.Decode(&sdt)

		if err != nil {
			if err == io.EOF {
				log.Println("eof :(")
				break
			}
			panic(err)
		}

		log.Printf("got sdt: %s\n", sdt)
	}
}
