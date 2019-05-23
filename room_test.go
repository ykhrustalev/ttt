package ttt_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/ykhrustalev/ttt"
	"testing"
)

type mockClient struct {
	mock.Mock
}

func (m *mockClient) Id() int64 {
	return m.Called().Get(0).(int64)
}

func (m *mockClient) Close() error {
	return m.Called().Error(0)

}

func (m *mockClient) WriteErrorln(err error) {
	m.Called(err)
}

func (m *mockClient) WriteErrorMessageln(message string) {
	m.Called(message)
}

func (m *mockClient) Writeln(message string) {
	m.Called(message)
}

func withFreshRoom(t *testing.T, callback func(room *ttt.Room, c1 *mockClient)) {
	var c1 mockClient

	room := ttt.NewRoom("room", &c1)

	callback(room, &c1)

	c1.AssertExpectations(t)
}

func withFullRoom(t *testing.T, callback func(room *ttt.Room, c1 *mockClient, c2 *mockClient)) {
	var c1 mockClient
	var c2 mockClient

	room := ttt.NewRoom("room", &c1)

	require.NoError(t, room.Join(&c2))

	callback(room, &c1, &c2)

	c1.AssertExpectations(t)
	c2.AssertExpectations(t)
}

func TestRoom(t *testing.T) {
	t.Run("Name", func(t *testing.T) {
		withFreshRoom(t, func(room *ttt.Room, c1 *mockClient) {
			assert.Equal(t, "room", room.Name())
		})
	})

	t.Run("Join", func(t *testing.T) {
		t.Run("is part of the room", func(t *testing.T) {
			withFreshRoom(t, func(room *ttt.Room, c1 *mockClient) {
				assert.EqualError(t, room.Join(c1), "already in the room")
			})
		})

		t.Run("new to the room", func(t *testing.T) {
			withFreshRoom(t, func(room *ttt.Room, c1 *mockClient) {
				var c2 mockClient

				assert.NoError(t, room.Join(&c2))

				c2.AssertExpectations(t)
			})
		})
	})

	t.Run("Leave", func(t *testing.T) {
		t.Run("not part of the room", func(t *testing.T) {
			withFreshRoom(t, func(room *ttt.Room, c1 *mockClient) {
				var c2 mockClient
				c2.On("WriteErrorMessageln", "the client not part of the room").Once()

				room.Leave(&c2)

				c2.AssertExpectations(t)
			})
		})

		t.Run("leave 1st/1", func(t *testing.T) {
			withFreshRoom(t, func(room *ttt.Room, c1 *mockClient) {
				c1.On("Writeln", "you have left the room room").Once()

				room.Leave(c1)
			})
		})

		t.Run("leave 1st/2", func(t *testing.T) {
			withFreshRoom(t, func(room *ttt.Room, c1 *mockClient) {
				c1.On("Writeln", "you have left the room room").Once()

				var c2 mockClient
				c2.On("Writeln", "opponent has left, game is over").Once()

				assert.NoError(t, room.Join(&c2))
				room.Leave(c1)
			})
		})

		t.Run("leave 2nd/2", func(t *testing.T) {
			withFreshRoom(t, func(room *ttt.Room, c1 *mockClient) {
				c1.On("Writeln", "opponent has left, game is over").Once()

				var c2 mockClient
				c2.On("Writeln", "you have left the room room").Once()

				assert.NoError(t, room.Join(&c2))
				room.Leave(&c2)
			})
		})
	})

	t.Run("StartIfReady", func(t *testing.T) {
		t.Run("unready", func(t *testing.T) {
			withFreshRoom(t, func(room *ttt.Room, c1 *mockClient) {
				room.StartIfReady()
			})
		})

		t.Run("ready", func(t *testing.T) {
			withFullRoom(t, func(room *ttt.Room, c1 *mockClient, c2 *mockClient) {
				c1.On("Writeln", "room is full, game starts").Once()
				c2.On("Writeln", "room is full, game starts").Once()

				c1.On("Writeln", ` . | . | . 
-----------
 . | . | . 
-----------
 . | . | . `).Once()
				c2.On("Writeln", ` . | . | . 
-----------
 . | . | . 
-----------
 . | . | . `).Once()

				c1.On("Writeln", "your turn").Maybe()
				c2.On("Writeln", "your turn").Maybe()

				c1.On("Writeln", "waiting for the opponent to make his turn").Maybe()
				c2.On("Writeln", "waiting for the opponent to make his turn").Maybe()

				room.StartIfReady()
			})
		})
	})
}
