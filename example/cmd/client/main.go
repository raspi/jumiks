package main

import (
	"fmt"
	"github.com/raspi/jumiks/pkg/client"
	"os"
	"time"
)

type ExampleClient struct {
	c     *client.Client
	delay time.Duration
}

func New(name string, errors chan error) (exclient *ExampleClient, err error) {
	exclient = &ExampleClient{
		delay: time.Millisecond * 500,
	}
	// Bind to ExampleClient.on_msg
	exclient.c, err = client.New(name, exclient.on_msg, errors)
	if err != nil {
		return nil, err
	}

	return exclient, nil
}

func (c *ExampleClient) Listen() {
	c.c.Listen()
}

// on_msg gets called every time there's a new message from client
func (c *ExampleClient) on_msg(b []byte) {
	fmt.Printf(`got %q`+"\n", string(b))
	time.Sleep(c.delay)
	c.delay += time.Millisecond * 50
}

func main() {
	errors := make(chan error)

	c, err := New("@test", errors)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, `error: %v`, err)
		os.Exit(1)
	}

	go c.Listen()

	for err := range errors {
		fmt.Printf(`got error: %v`, err)
	}
}
