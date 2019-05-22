package ttt

import (
	"bufio"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"regexp"
	"strings"
	"sync"
)

type Client struct {
	id      int64
	conn    net.Conn
	room    *Room
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

			c.handleMessage(string(line))

			c.logger.WithField("message", string(line)).Info("new message")
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
	c.mx.Lock()
	defer c.mx.Unlock()

	return c.conn.Close()
}

var quitRegex = regexp.MustCompile("^(?i)(quit).*$")
var joinRegex = regexp.MustCompile("^(?i)(join) (\\w+)$")
var markRegex = regexp.MustCompile("^(?i)(mark) (\\w+)$")

func (c *Client) handleMessage(message string) {
	// todo: state in a single place
	message = strings.ToLower(message)

	if quitRegex.MatchString(message) {
		c.service.DisconnectClient(c)
		return
	}

	if joinRegex.MatchString(message) {
		marker := joinRegex.FindAllStringSubmatch(message, -1)[0][0]

		c.writer.WriteString("joining " + marker)
		c.writer.Flush()

		err := c.service.JoinClientToRoom(marker, c)
		if err != nil {
			c.writer.WriteString("error " + err.Error())
			c.writer.Flush()
		}

		return
	}

	return
}

func (c *Client) Writeln(message string) {
	c.mx.Lock()
	defer c.mx.Unlock()

	_, err := c.writer.WriteString(message + "\n")
	if err != nil {
		c.logger.WithError(err).WithField("message", message).Error("failed to write the message")
	}
}
