package aria

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// Run is a entrypoint called from main
func Run(ctx context.Context) {
	// setup logger
	setupLogger()

	config, err := newConfig()
	if err != nil {
		log.Printf("failed to load config: %v\n", err)
		return
	}

	newLauncher(config).launch(ctx)
}

func setupLogger() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

type launcher struct {
	config *config

	cliToBot chan *packet
	botToCli chan *request
	stream   chan []byte
}

func newLauncher(config *config) *launcher {
	return &launcher{
		config:   config,
		cliToBot: make(chan *packet),
		botToCli: make(chan *request),
		stream:   make(chan []byte),
	}
}

func (l *launcher) launch(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	wg := sync.WaitGroup{}
	defer wg.Wait()

	// when error is reported, shutdown all.
	errChan := make(chan error)
	go func() {
		select {
		case <-ctx.Done():
			log.Printf("shutting down errChan watcher")
			return
		case err := <-errChan:
			log.Printf("reported error: %v\n", err)
			cancel()
		}
	}()

	wg.Add(1)
	go func() {
		l.launchBot(ctx, errChan)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		l.launchClient(ctx, errChan)
		wg.Done()
	}()
}

func (l *launcher) launchBot(ctx context.Context, errChan chan<- error) {
	var start time.Time
	b, err := newBot(l.config, l.cliToBot, l.botToCli, l.stream)
	if err != nil {
		errChan <- fmt.Errorf("failed to initialize bot: %w", err)
		return
	}

	for {
		if time.Now().Sub(start) < 5*time.Minute {
			log.Printf("bot cooldown: 1min...")
			select {
			case <-ctx.Done():
				log.Printf("launch cancelled: bot")
				return
			case <-time.After(time.Minute):
			}
		}

		select {
		case <-ctx.Done():
			log.Printf("stopped: bot")
			return
		default:
			log.Printf("starting: bot")
			start = time.Now()
			b.run(ctx)
			log.Printf("crashed: bot")
		}
	}
}

func (l *launcher) launchClient(ctx context.Context, errChan chan<- error) {
	var start time.Time
	for {
		// do restart cooldown
		if start != (time.Time{}) { // avoid cooldown for first launch
			cooldown := 10 * time.Second
			if time.Now().Sub(start) < 5*time.Minute {
				cooldown = time.Minute
			}

			log.Printf("client cooldown: %v...", cooldown)
			select {
			case <-ctx.Done():
				log.Printf("launch cancelled: client")
				return
			case <-time.After(cooldown):
			}
		}

		c, err := newClient(l.config, l.cliToBot, l.botToCli, l.stream)
		if err != nil {
			errChan <- fmt.Errorf("failed to initialize client: %w", err)
			return
		}

		select {
		case <-ctx.Done():
			log.Printf("stopped: client")
			return
		default:
			log.Printf("starting: client")
			start = time.Now()
			c.run(ctx)
			log.Printf("crashed: client")
		}
	}
}
