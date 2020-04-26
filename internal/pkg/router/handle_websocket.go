package router

import(
	"log"
	"net/http"

	"github.com/luisguve/cheropatilla/internal/pkg/livedata"
)

// Register new clients into the hub.
// Send notifications to registered and logged in users and receive signal
// to mark all notifications as read via websocket.
func (r *Router) handleLiveNotifs(w http.ResponseWriter, req *http.Request) {
	userId := r.currentUser(req)
	if userId == "" {
		// user has not logged in.
		http.Error(w, "not logged in", http.StatusForbidden)
		return
	}
	conn, err := r.upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Printf("Could not upgrade connection: %v\n", err)
		http.Error(w, "an error occurred", http.StatusInternalServerError)
		return
	}
	client := &livedata.Client{
		Hub. r.hub,
		Conn: conn,
		User: &livedata.User{
			Id:        userId,
			SendNotif: make(chan *pb.Notif, 256),
			SendOk:    make(chan bool),
		},
	}
	client.Hub.Register <- client.User
	go client.WritePump()
	go client.ReadPump()
}
