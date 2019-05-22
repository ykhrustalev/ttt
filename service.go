package ttt

import (
	"context"
	"github.com/sirupsen/logrus"
	"sync"
)

type Service struct {
	rooms   map[string]*Room
	clients []*Client

	logger *logrus.Entry

	mx sync.Mutex
}

func NewService() *Service {
	return &Service{rooms: make(map[string]*Room), logger: logrus.WithField("name", "service")}
}

func (s *Service) Register(ctx context.Context, client *Client) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.clients = append(s.clients, client)

	go client.Listen(ctx)
}

func (s *Service) Start(name string, client *Client) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	r, ok := s.rooms[name]

	if !ok {
		room := NewRoomWithName(name, client)
		s.rooms[name] = room
		return nil
	}

	return r.Join(client)
}

func (s *Service) ListFree() (r []*Room) {
	s.mx.Lock()
	defer s.mx.Unlock()

	for _, v := range s.rooms {
		if v.CanJoin() {
			r = append(r, v)
		}
	}
	return
}

func (s *Service) Close() {
	s.mx.Lock()
	defer s.mx.Unlock()

	for _, v := range s.rooms {
		err := v.Close()
		if err != nil {
			s.logger.WithError(err).Error("failed to close a room")
		}
	}
	return
}
