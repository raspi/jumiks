package jumiks

import (
	"github.com/raspi/jumiks/pkg/client"
	"github.com/raspi/jumiks/pkg/server"
	error2 "github.com/raspi/jumiks/pkg/server/error"
	"testing"
	"time"
)

// TestServerAndClient tests server which sends a message to client
func TestServerAndClient(t *testing.T) {
	timeout := time.After(5 * time.Second)

	want := []byte(`hello, world!`)

	uaddr := `@test-server`
	errs := make(chan error2.Error)
	srv, err := server.New(uaddr, 100, errs)
	if err != nil {
		t.Fail()
	}
	defer srv.Close()

	go srv.Listen()

	// Wait for setup
	time.Sleep(time.Second * 1)

	cerrs := make(chan error)

	received := make(chan []byte)

	clnt, err := client.New(uaddr,
		func(b []byte) {
			received <- b // send to chan
		}, cerrs)
	if err != nil {
		t.Fail()
	}
	defer clnt.Close()

	go clnt.Listen()

	// Wait for setup
	time.Sleep(time.Second * 1)

	srv.SendToAll(want)

	select {
	case got := <-received:
		if string(got) != string(want) {
			t.Fail()
		}
	case <-timeout:
		t.Fatalf(`test timed out`)
	}

}
