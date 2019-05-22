package ttt

import "C"
import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"math/rand"
	"strings"
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

	results    [9]string
	markersCnt int
	isOver     bool

	logger *logrus.Entry

	mx sync.Mutex
}

func NewRoom(name string, c *Client) *Room {
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

func (r *Room) Leave(c *Client) {
	r.mx.Lock()
	defer r.mx.Unlock()

	if r.c1 == c {
		r.c1 = nil
	} else if r.c2 == c {
		r.c2 = nil
	} else {
		c.WriteErrorMessageln("the client not part of the room")
		return
	}

	c.Writeln(fmt.Sprintf("you have left the room %s", r.Name()))

	if r.c1 != nil {
		r.notifyOpponentLef(r.c1)
	}

	if r.c2 != nil {
		r.notifyOpponentLef(r.c2)
	}

	r.isOver = true // disable the game
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
		r.c2Marker = "o"
	} else {
		r.c1Turn = false
		r.c1Marker = "o"
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

	current.Writeln("room is full, game starts")
	waiting.Writeln("room is full, game starts")

	r.notifyClients()
}

func (r *Room) isClientTurn(client *Client) bool {
	if r.c1Turn {
		return r.c1 == client
	}

	return r.c2 == client
}

func (r *Room) AttemptMark(client *Client, position int) {
	r.mx.Lock()
	defer r.mx.Unlock()

	if r.isOver {
		client.Writeln("game is over, no further turns")
		return
	}

	if !r.isClientTurn(client) {
		client.Writeln("please, hold on, your opponent is still thinking")
		return
	}

	index := position - 1

	if r.results[index] != "" {
		client.Writeln("this field is already occupied")
		return
	}

	marker := r.c2Marker
	if r.c1Turn {
		marker = r.c1Marker
	}

	r.results[index] = marker
	r.markersCnt++

	if r.markersCnt >= 9 {
		r.isOver = true
		r.notifyOnResults()
	} else {
		r.swapTurn()
		r.notifyClients()
	}
}

func (r *Room) swapTurn() {
	r.c1Turn = !r.c1Turn
}

func replaceEmpty(row []string) (r []string) {
	for _, x := range row {
		if x == "" {
			r = append(r, ".")
		} else {
			r = append(r, x)
		}
	}
	return
}
func buildRow(row []string) string {
	return strings.Join(replaceEmpty(row), "|")
}

func (r *Room) printForClient(client *Client) {
	msg := strings.Join([]string{
		buildRow(r.results[0:3]),
		"-----",
		buildRow(r.results[3:6]),
		"-----",
		buildRow(r.results[6:9]),
	}, "\n")

	client.Writeln(msg)
}

func (r *Room) notifyOnTurn(client *Client) {
	client.Writeln("your turn")
}

func (r *Room) notifyOnWait(client *Client) {
	client.Writeln("waiting for the opponent to make his turn")
}

func (r *Room) notifyGameOver(client *Client) {
	client.Writeln("game over")
}

func (r *Room) notifyOpponentLef(client *Client) {
	client.Writeln("opponent has left, game is over")
}

func (r *Room) notifyClients() {
	r.printForClient(r.c1)
	r.printForClient(r.c2)

	if r.c1Turn {
		r.notifyOnTurn(r.c1)
		r.notifyOnWait(r.c2)
	} else {
		r.notifyOnTurn(r.c2)
		r.notifyOnWait(r.c1)
	}
}

func (r *Room) notifyOnResults() {
	r.printForClient(r.c1)
	r.printForClient(r.c2)
	r.notifyGameOver(r.c1)
	r.notifyGameOver(r.c2)
}

func (r *Room) Close() error {
	r.mx.Lock()
	defer r.mx.Unlock()

	_ = r.c1.Close()
	_ = r.c2.Close()

	return nil
}
