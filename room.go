package ttt

import "C"
import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
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

func (r *Room) CanJoin() bool {
	r.mx.Lock()
	defer r.mx.Unlock()

	return r.c1 == nil || r.c2 == nil
}

func (r *Room) Close() error {
	r.mx.Lock()
	defer r.mx.Unlock()

	for _, c := range []*Client{r.c1, r.c2} {
		_ = c.Close()
	}

	return nil
}
