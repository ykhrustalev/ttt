package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/ykhrustalev/ttt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("failed %v", err)
	}
}

var arguments struct {
	Listen string
}

func init() {
	flag.StringVar(&arguments.Listen, "listen", ":8999", "address to listen")
	flag.Parse()
}

func initSignals(onQuit func()) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		wg.Done()
		<-quit
		fmt.Println("quiting")
		onQuit()
	}()

	wg.Wait()
}

func run() error {
	logger := logrus.WithField("name", "server")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initSignals(cancel)

	ln, err := net.Listen("tcp", arguments.Listen)
	if err != nil {
		return err
	}

	listenerErr := make(chan error)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		wg.Done()

		service := ttt.NewService()
		defer service.Close()

		for {
			logger.Debug("listen loop", ln)

			conn, err := ln.Accept()
			if err != nil {
				listenerErr <- err
				close(listenerErr)
				return
			}

			service.ConnectClient(ctx, conn)
		}
	}()

	wg.Wait()

	select {
	case <-ctx.Done():
		logger.Info("exit signal")
		ln.Close()
		return nil
	case err, ok := <-listenerErr:
		if !ok {
			logger.Info("listener channel closed")
			return nil
		}

		logger.WithError(err).Error("error accepting connection")
		return err
	}
}
