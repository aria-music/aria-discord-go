package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/aria-music/aria"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal)
	signal.Notify(sig, os.
		Interrupt, os.Kill)

	go func() {
		<-sig
		cancel()
	}()

	aria.Run(ctx)
}
