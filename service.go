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
	clientById     sync.Map
	roomByName     sync.Map
	roomByClientId sync.Map

	logger *logrus.Entry
}

func NewService() *Service {
	return &Service{
		logger: logrus.WithField("name", "service"),
	}
}

func (s *Service) ConnectClient(ctx context.Context, conn net.Conn) {
	client := NewClient(ctx, conn, s)

	s.clientById.Store(client.Id(), client)

	client.Writeln("hello, use `join x` to join or create a room, type `quit` to exit")
	client.Writeln("")
}

var quitRegex = regexp.MustCompile("^(?i)(quit).*$")
var joinRegex = regexp.MustCompile("^(?i)(join)\\s+(\\w+)\\s*$")
var markRegex = regexp.MustCompile("^(?i)(mark)\\s+(\\w+)\\s*$")

func (s *Service) HandleMessage(client Client, message string) {
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

func (s *Service) HandleDisconnectClient(client Client) {
	s.handleDisconnectClient(client)
}

func (s *Service) handleDisconnectClient(client Client) {
	logger := s.logger.WithField("clientId", client.Id())

	s.clientById.Delete(client.Id())

	if err := client.Close(); err != nil {
		logger.WithError(err).Error("failed to close connection on client disconnect")
	}

	s.leaveRooms(client)

	s.roomByClientId.Delete(client.Id())

	logger.Info("disconnected")
	return
}

func (s *Service) leaveRooms(client Client) {
	room, ok := s.roomByClientId.Load(client.Id())
	if !ok {
		// not in any room
		return
	}

	s.roomByClientId.Delete(client.Id())

	room.(*Room).Leave(client)
}

func (s *Service) handleJoinRoom(client Client, roomName string) {
	logger := s.logger.WithFields(logrus.Fields{
		"roomName": roomName,
		"clientId": client.Id(),
	})

	// any existing rooms
	s.leaveRooms(client)

	room, ok := s.roomByName.Load(roomName)
	if !ok {
		// new room
		logger.Info("creating a new room")

		room = NewRoom(roomName, client)
		s.roomByName.Store(roomName, room.(*Room))
		s.roomByClientId.Store(client.Id(), room.(*Room))

		client.Writeln("joined new room, you are the first here")
		return
	}

	// existing room
	if err := room.(*Room).Join(client); err != nil {
		client.WriteErrorln(err)
		logger.WithError(err).Error("failed to join the room")
		return
	}

	s.roomByClientId.Store(client.Id(), room.(*Room))

	client.Writeln("joined the existing room")
	logger.Info("joined the existing room")

	room.(*Room).StartIfReady()
}

func (s *Service) handleMark(client Client, marker string) {
	logger := s.logger.WithField("clientId", client.Id())

	position, err := strconv.ParseInt(marker, 10, 64)
	if err != nil {
		logger.WithError(err).Error("invalid position")
		client.WriteErrorMessageln("marker should be int")
		return
	}

	if position < 1 || position > 9 {
		logger.WithError(err).Error("invalid position")
		client.WriteErrorMessageln("marker should be in range of 1-9")
		return
	}

	room, ok := s.roomByClientId.Load(client.Id())
	if !ok {
		logger.Error("not in any room")
		client.WriteErrorMessageln("you don't belong to any room yet")
		return
	}

	room.(*Room).AttemptMark(client, int(position))
}

func (s *Service) Close() {
	s.roomByName.Range(func(_, value interface{}) bool {
		room := value.(*Room)

		err := room.Close()
		if err != nil {
			s.logger.WithError(err).Error("failed to close a room")
		}

		return true
	})
}
