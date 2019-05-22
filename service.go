package ttt

import (
	"context"
	"github.com/sirupsen/logrus"
	"net"
	"regexp"
	"strconv"
	"sync"
)

type Service struct {
	clientById     map[int64]*Client
	roomByName     map[string]*Room
	roomByClientId map[int64]*Room

	logger *logrus.Entry

	mx sync.Mutex
}

func NewService() *Service {
	return &Service{
		clientById:     make(map[int64]*Client),
		roomByName:     make(map[string]*Room),
		roomByClientId: make(map[int64]*Room),

		logger: logrus.WithField("name", "service"),
	}
}

func (s *Service) ConnectClient(ctx context.Context, conn net.Conn) {
	s.mx.Lock()
	defer s.mx.Unlock()

	client := NewClient(ctx, conn, s)

	s.clientById[client.id] = client

	client.Writeln("hello, use `join x` to join or create a room, type `quit` to exit")
	client.Writeln("")
}

var quitRegex = regexp.MustCompile("^(?i)(quit).*$")
var joinRegex = regexp.MustCompile("^(?i)(join)\\s+(\\w+)\\s*$")
var markRegex = regexp.MustCompile("^(?i)(mark)\\s+(\\w+)\\s*$")

func (s *Service) HandleMessage(client *Client, message string) {
	//
	// no locks in this func
	//

	if quitRegex.MatchString(message) {
		s.handleDisconnectClient(client)
		return
	}

	if joinRegex.MatchString(message) {
		roomName := joinRegex.FindAllStringSubmatch(message, -1)[0][2]
		s.handleJoinRoom(client, roomName)
		return
	}

	if markRegex.MatchString(message) {
		marker := markRegex.FindAllStringSubmatch(message, -1)[0][2]
		s.handleMark(client, marker)
		return
	}

	client.WriteErrorMessageln("unknown command")
}

func (s *Service) handleDisconnectClient(client *Client) {
	s.mx.Lock()
	defer s.mx.Unlock()

	logger := s.logger.WithField("clientId", client.Id())

	delete(s.clientById, client.Id())

	if err := client.Close(); err != nil {
		logger.WithError(err).Error("failed to close connection on client disconnect")
	}

	if err := s.leaveRooms(client); err != nil {
		logger.WithError(err).Error("failed to leave the room on client disconnect")
	}

	delete(s.roomByClientId, client.Id())

	logger.Info("disconnected")
	return
}

func (s *Service) leaveRooms(client *Client) error {
	room, ok := s.roomByClientId[client.Id()]
	if !ok {
		// not in any room
		return nil
	}

	if err := room.Leave(client); err != nil {
		return err
	}

	delete(s.roomByClientId, client.Id())

	return nil
}

func (s *Service) handleJoinRoom(client *Client, roomName string) {
	s.mx.Lock()
	defer s.mx.Unlock()

	logger := s.logger.WithFields(logrus.Fields{
		"roomName": roomName,
		"clientId": client.Id(),
	})

	// any existing rooms
	if err := s.leaveRooms(client); err != nil {
		s.logger.WithFields(logrus.Fields{
			"olderRoom": roomName,
			"clientId":  client.Id(),
		}).WithError(err).Error("failed to leave the room on joining")
	}

	room, ok := s.roomByName[roomName]
	if !ok {
		// new room
		logger.Info("creating a new room")

		room = NewRoomWithName(roomName, client)
		s.roomByName[roomName] = room
		s.roomByClientId[client.Id()] = room

		client.Writeln("joined new room, you are the first here")
		return
	}

	// existing room
	if err := room.Join(client); err != nil {
		client.WriteErrorln(err)
		logger.WithError(err).Error("failed to join the room")
		return
	}

	s.roomByClientId[client.Id()] = room

	client.Writeln("joined the existing room")
	logger.Info("joined the existing room")

	room.StartIfReady()
}

func (s *Service) handleMark(client *Client, marker string) {
	s.mx.Lock()
	defer s.mx.Unlock()

	logger := s.logger.WithField("clientId", client.Id())

	position, err := strconv.ParseInt(marker, 10, 64)
	if err != nil {
		logger.WithError(err).Error("invalid position")
		client.Writeln("marker should be int")
		return
	}

	if position < 1 || position > 9 {
		logger.WithError(err).Error("invalid position")
		client.Writeln("marker should be in range of 1-9")
		return
	}

	room, ok := s.roomByClientId[client.Id()]
	if !ok {
		logger.Error("not in any room")
		client.WriteErrorMessageln("you don't belong to any room yet")
		return
	}

	room.AttemptMark(client, int(position))
}

func (s *Service) Close() {
	s.mx.Lock()
	defer s.mx.Unlock()

	for _, room := range s.roomByName {
		err := room.Close()
		if err != nil {
			s.logger.WithError(err).Error("failed to close a room")
		}
	}
	return
}
