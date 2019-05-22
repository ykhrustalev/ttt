package ttt

import (
	"bufio"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"sync"
)

type Client struct {
	conn net.Conn

	logger *logrus.Entry
}

var clientCnt = 0

func NewClient(conn net.Conn) *Client {
	clientCnt++
	return &Client{
		conn:   conn,
		logger: logrus.WithField("name", fmt.Sprintf("client-%d", clientCnt)),
	}
}

func (c *Client) Listen(ctx context.Context) {
	defer c.conn.Close()

	c.logger.Info("new client")

	reader := bufio.NewReader(c.conn)
	writer := bufio.NewWriter(c.conn)

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

			c.logger.WithField("message", string(line)).Info("new message")

			writer.Write([]byte("server\n"))
			writer.Flush()
		}
	}()

	wg.Wait()

	select {
	case <-ctx.Done():
		c.logger.Info("exit signal on the server")
		return
	case err, ok := <-workerErr:
		if !ok {
			// should not happen
			c.logger.Info("connection channel closed")
			return
		}

		if err == nil {
			// should not happen
			c.logger.Info("connection closed, no error")
			return
		}

		if err == io.EOF {
			c.logger.Info("connection closed")
			return
		}

		c.logger.WithError(err).Error("failed to read from the client")
		return
	}
}

func (c *Client) Close() error {
	return c.conn.Close()
}
