package main

import (
	"log"

	"code.dopame.me/veonik/squircy3/event"

	"code.dopame.me/veonik/squirssi"
)

func main() {
	ev := event.NewDispatcherLimit(512)
	srv, err := squirssi.NewServer(ev)
	if err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer srv.Close()

	srv.Start()
}