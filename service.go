package ttt

import (
	"context"
	"github.com/sirupsen/logrus"
	"net"
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

func (s *Service) AddClient(ctx context.Context, conn net.Conn) {
	s.mx.Lock()
	defer s.mx.Unlock()

	client := NewClient(ctx, conn, s)

	s.clientById[client.id] = client
}

func (s *Service) DisconnectClient(client *Client) {
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

func (s *Service) JoinClientToRoom(name string, client *Client) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	r, ok := s.roomByName[name]

	if !ok {
		room := NewRoomWithName(name, client)
		s.roomByName[name] = room
		return nil
	}

	return r.Join(client)
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
