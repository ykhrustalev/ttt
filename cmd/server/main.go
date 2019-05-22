package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/ykhrustalev/ttt/apps"
	"io"
	"log"
	"net"
	"sync"
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

func run() error {
	logger := logrus.WithField("name", "server")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apps.InitSignals(cancel)

	ln, err := net.Listen("tcp", arguments.Listen)
	if err != nil {
		return err
	}

	listenerErr := make(chan error)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		wg.Done()

		for {
			logger.Debug("listen loop", ln)

			conn, err := ln.Accept()
			if err != nil {
				listenerErr <- err
				close(listenerErr)
				return
			}

			go handle(ctx, conn)
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

var clientCnt = 0

func getLogger() *logrus.Entry {
	clientCnt++
	return logrus.WithField("name", fmt.Sprintf("client-%d", clientCnt))
}

func handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	logger := getLogger()

	logger.Info("new client")

	reader := bufio.NewReader(conn)

	workerErr := make(chan error)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		wg.Done()

		for {
			line, _, err := reader.ReadLine() // todo: handle line split
			if err != nil {
				workerErr <- err
				close(workerErr)
				return
			}

			logger.WithField("message", string(line)).Info("new message")
		}
	}()

	wg.Wait()

	select {
	case <-ctx.Done():
		logger.Info("exit signal on the server")
		return
	case err, ok := <-workerErr:
		if !ok {
			// should not happen
			logger.Info("connection channel closed")
			return
		}

		if err == nil {
			// should not happen
			logger.Info("connection closed, no error")
			return
		}

		if err == io.EOF {
			logger.Info("connection closed")
			return
		}

		logger.WithError(err).Error("failed to read from the client")
		return
	}
}
