package main

import (
	"fmt"
	"github.com/raspi/jumiks/pkg/server"
	error2 "github.com/raspi/jumiks/pkg/server/error"
	"time"
)

func main() {
	var errors chan error2.Error

	l, err := server.New("@test", 5, errors)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	go l.Listen()

	counter := 0

	for {
		select {
		case err := <-errors:
			fmt.Printf(`got error %v`+"\n", err)
		default:
			msg := fmt.Sprintf("hello, world %d!", counter)
			l.SendToAll([]byte(msg))
			fmt.Printf(`sent %q`+"\n", msg)

			time.Sleep(time.Millisecond * 500)
			counter++
		}
	}

}
