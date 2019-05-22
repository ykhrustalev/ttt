package ttt

import (
	"context"
	"github.com/sirupsen/logrus"
	"net"
	"regexp"
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
}

var quitRegex = regexp.MustCompile("^(?i)(quit).*$")
var joinRegex = regexp.MustCompile("^(?i)(join) (\\w+)$")
var markRegex = regexp.MustCompile("^(?i)(mark) (\\w+)$")

func (s *Service) HandleMessage(client *Client, message string) {
	//
	// no locks in this func
	//

	if quitRegex.MatchString(message) {
		s.disconnectClient(client)
		return
	}

	if joinRegex.MatchString(message) {
		roomName := joinRegex.FindAllStringSubmatch(message, -1)[0][0]
		s.joinClientToRoom(client, roomName)
		return
	}

	client.WriteErrorMessageln("unknown command")
}

func (s *Service) disconnectClient(client *Client) {
	s.mx.Lock()
	defer s.mx.Unlock()

	logger := s.logger.WithField("clientId", client.Id())

	delete(s.clientById, client.Id())

	if err := client.Close(); err != nil {
		logger.WithError(err).Error("failed to close connection on client disconnect")
	}

	room, ok := s.roomByClientId[client.Id()]
	if !ok {
		// not in any room
		logger.Info("disconnected")
		return
	}

	if err := room.Leave(client); err != nil {
		logger.WithField("roomName", room.Name()).WithError(err).Error("failed to leave the room on client disconnect")
	}

	logger.Info("disconnected")
	return
}

func (s *Service) AddRoom(room *Room) { // todo: review
	s.mx.Lock()
	defer s.mx.Unlock()

	s.roomByName[room.Name()] = room
}

func (s *Service) RemoveRoom(room *Room) { // todo: review
	s.mx.Lock()
	defer s.mx.Unlock()

	delete(s.roomByName, room.Name())
}

func (s *Service) joinClientToRoom(client *Client, roomName string) {
	s.mx.Lock()
	defer s.mx.Unlock()

	room, ok := s.roomByName[roomName]

	logger := s.logger.WithFields(logrus.Fields{
		"roomName": roomName,
		"clientId": client.Id(),
	})

	if !ok {
		logger.Info("creating a new room")

		room = NewRoomWithName(roomName, client)
		s.roomByName[roomName] = room
		s.roomByClientId[client.Id()] = room

		client.Writeln("joined new room, you are the first here")
		return
	}

	err := room.Join(client)
	if err != nil {
		logger.WithError(err).Error("failed to join the room")

		client.WriteErrorln(err)
		return
	}

	s.roomByClientId[client.Id()] = room
	client.Writeln("joined the existing room")

	logger.Info("joined the existing room")
}

func (s *Service) Close() {
	s.mx.Lock()
	defer s.mx.Unlock()

	for _, v := range s.roomByName {
		err := v.Close()
		if err != nil {
			s.logger.WithError(err).Error("failed to close a room")
		}
	}
	return
}
