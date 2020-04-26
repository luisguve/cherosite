package livedata

import(
	pb "github.com/luisguve/cheropatilla/models/crudServiceClientPb" // TO DEFINE
)

// Hub maintains the set of active users and is responsible for broadcasting notifications
// to the users they are intended for and for marking the notifications as read when 
// users ask so.
type Hub struct {
	// onlineUsers is the collection of users that are currently active.
	onlineUsers map[string]*user

	// Register is a channel for registering users in the onlineUsers collection.
	Register chan *user

	// Unregister is a channel for unregistering users in the onlineUsers collection.
	Unregister chan string

	// ReadAllFromUser is a channel that marks all the notifications of the given
	// user as read.
	ReadAllFromUser chan string
}

func NewHub() *Hub {
	return &Hub{
		onlineUsers:		make(map[string]*User),
		Register:			make(chan *User),
		Unregister:			make(chan string),
		ReadAllFromUser:	make(chan string),
	}
}

// Run continuously listens for user registering/unregistering messages
func (h *Hub) Run(crudServiceClient *pb.CrudCheropatillaClient) {
	for {
		select {
		case user := <- h.Register:
			h.onlineUsers[user.id] = user
		case userId := <- h.Unregister:
			if user, ok := h.onlineUsers[userId]; ok {
				delete(h.onlineUsers, userId)
				close(user.sendNotif)
				close(user.sendOk)
			}
		case userId := <- h.ReadAllFromUser:
			if user, ok := h.onlineUsers[userId]; ok {
				go markAllAsRead(userId, user.sendOk, crudServiceClient)
			}
		}
	}
}

func (h *Hub) Broadcast(userId string, notif *pb.Notif) {
	// Check whether the user is online.
	if user, ok := h.onlineUsers[userId]; ok {
		select {
		case user.sendNotif <- notif:
		default:
			// The user connection is stuck or dead. Proceed to remove this user.
			h.Unregister <- userId
		}
	}
}

func markAllAsRead(userId string, sendOk chan bool, cc *pb.CrudCheropatillaClient) {
	_, err := cc.MarkAllAsRead(context.Background(), &pb.ReadNotifsRequest{userId})
	if err != nil {
		log.Println("Something went wrong while reading notifs: %v\n", err)
		sendOk <- false 
	} else {
		sendOk <- true
	}
}
