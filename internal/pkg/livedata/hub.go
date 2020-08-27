package livedata

import (
	"context"
	"log"

	pbUsers "github.com/luisguve/cheroproto-go/userapi"
	pbDataFormat "github.com/luisguve/cheroproto-go/dataformat"
)

// Hub maintains the set of active users and is responsible for broadcasting
// notifications to the users they are intended for and for marking the
// notifications as read when  users ask so.
type Hub struct {
	// onlineUsers is the collection of users that are currently active.
	onlineUsers map[string]*User

	// Register is a channel for registering users in the onlineUsers collection.
	Register chan *User

	// Unregister is a channel for unregistering users in the onlineUsers collection.
	Unregister chan string

	// ReadAllFromUser is a channel that marks all the notifications of the given
	// user as read.
	ReadAllFromUser chan string

	// Client to perform user-related crud operations, mostly involving notification
	// management.
	usersClient pbUsers.CrudUsersClient
}

func NewHub(client pbUsers.CrudUsersClient) *Hub {
	return &Hub{
		onlineUsers:     make(map[string]*User),
		Register:        make(chan *User),
		Unregister:      make(chan string),
		ReadAllFromUser: make(chan string),
		usersClient:     client,
	}
}

// Run continuously listens for user registering/unregistering messages
func (h *Hub) Run() {
	for {
		select {
		case user := <-h.Register:
			h.onlineUsers[user.Id] = user
		case userId := <-h.Unregister:
			if user, ok := h.onlineUsers[userId]; ok {
				delete(h.onlineUsers, userId)
				close(user.SendNotif)
				close(user.SendOk)
			}
		case userId := <-h.ReadAllFromUser:
			if user, ok := h.onlineUsers[userId]; ok {
				go h.markAllAsRead(userId, user.SendOk)
			}
		}
	}
}

func (h *Hub) Broadcast(userId string, notif *pbDataFormat.Notif) {
	// Check whether the user is online.
	if user, ok := h.onlineUsers[userId]; ok {
		select {
		case user.SendNotif <- notif:
		default:
			// The user connection is stuck or dead. Proceed to remove this user.
			h.Unregister <- userId
		}
	}
}

func (h *Hub) markAllAsRead(userId string, sendOk chan bool) {
	_, err := h.usersClient.MarkAllAsRead(context.Background(), &pbUsers.ReadNotifsRequest{UserId: userId})
	if err != nil {
		log.Println("Could not send request to mark all notifs as read: %v\n", err)
		sendOk <- false
	} else {
		sendOk <- true
	}
}
