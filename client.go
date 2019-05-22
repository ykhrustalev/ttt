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
	id      int64
	conn    net.Conn
	service *Service

	reader *bufio.Reader
	writer *bufio.Writer

	logger *logrus.Entry

	mx sync.Mutex
}

var clientCnt int64

func NewClient(ctx context.Context, conn net.Conn, service *Service) *Client {
	clientCnt++
	name := fmt.Sprintf("client-%d", clientCnt)

	c := &Client{
		id:      clientCnt,
		conn:    conn,
		reader:  bufio.NewReader(conn),
		writer:  bufio.NewWriter(conn),
		service: service,
		logger:  logrus.WithField("name", name),
	}

	go c.listen(ctx)

	return c
}

func (c *Client) Id() int64 {
	return c.id
}

func (c *Client) listen(ctx context.Context) {
	defer c.conn.Close()

	c.logger.Info("new client")

	workerErr := make(chan error)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		wg.Done()

		for {
			line, _, err := c.reader.ReadLine() // todo: handle line split
			if err != nil {
				workerErr <- err
				close(workerErr)
				return
			}

			message := string(line)
			c.service.HandleMessage(c, message)

			c.logger.Infof("client says: %s", message)
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

func (c *Client) WriteErrorln(err error) {
	c.WriteErrorMessageln(err.Error())
}

func (c *Client) WriteErrorMessageln(message string) {
	c.Writeln(fmt.Sprintf("err: %s", message))
}

func (c *Client) Writeln(message string) {
	c.mx.Lock()
	defer c.mx.Unlock()

	_, err := c.writer.WriteString(message + "\n")
	if err != nil {
		c.logger.WithError(err).WithField("message", message).Error("failed to write the message")
	}

	err = c.writer.Flush()
	if err != nil {
		c.logger.WithError(err).WithField("message", message).Error("failed to flush the message")
	}
}
