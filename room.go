package ttt

import "C"
import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"math/rand"
	"sync"
)

var roomsCnt = 0

func nextRoomName() string {
	roomsCnt++
	return fmt.Sprintf("%d", roomsCnt)
}

type Room struct {
	name string
	c1   *Client
	c2   *Client

	c1Marker string
	c2Marker string

	c1Turn bool

	results [9]string

	logger *logrus.Entry

	mx sync.Mutex
}

func NewRoom(c *Client) *Room {
	return NewRoomWithName(nextRoomName(), c)
}

func NewRoomWithName(name string, c *Client) *Room {
	return &Room{
		c1:     c,
		name:   name,
		logger: logrus.WithField("name", fmt.Sprintf("room-%s", name)),
	}
}

func (r *Room) Name() string {
	return r.name
}

func (r *Room) Join(c *Client) error {
	r.mx.Lock()
	defer r.mx.Unlock()

	if r.c1 == nil {
		r.c1 = c
		return nil
	}

	if r.c2 == nil {
		r.c2 = c
		return nil
	}

	return errors.New("room is already occupied")
}

func (r *Room) Leave(c *Client) error {
	r.mx.Lock()
	defer r.mx.Unlock()

	if r.c1 == c {
		r.c1 = nil
		return nil
	}

	if r.c2 == c {
		r.c2 = nil
		return nil
	}

	return errors.New("the client not part of the room")
}

func randBool() bool {
	return rand.Intn(2) == 0
}

func (r *Room) isFull() bool {
	return r.c1 != nil && r.c2 != nil
}

func (r *Room) StartIfReady() {
	r.mx.Lock()
	defer r.mx.Unlock()

	if r.isFull() {
		r.start()
	}
}

func (r *Room) start() {
	if randBool() {
		r.c1Turn = true
		r.c1Marker = "x"
		r.c2Marker = "0"
	} else {
		r.c1Turn = false
		r.c1Marker = "0"
		r.c2Marker = "x"
	}

	var current, waiting *Client

	if r.c1Turn {
		current = r.c1
		waiting = r.c2
	} else {
		current = r.c2
		waiting = r.c1
	}

	current.Writeln("game started")
	current.Writeln("your turn")

	waiting.Writeln("game started")
	waiting.Writeln("waiting for the opponent to make his turn")
}

func (r *Room) Close() error {
	r.mx.Lock()
	defer r.mx.Unlock()

	for _, c := range []*Client{r.c1, r.c2} {
		_ = c.Close()
	}

	return nil
}
