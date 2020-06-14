package livedata

import(
	"bytes"
	"log"
	"net/http"
	"time"

	pbDataFormat "github.com/luisguve/cheroproto-go/dataformat"
	"github.com/gorilla/websocket"
)

type Client struct {
	Hub *Hub
	Conn *websocket.Conn
	User *User
}

type User struct {
	Id        string
	SendNotif chan *pbDataFormat.Notif
	SendOk    chan bool
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 8

	// Maximum buffer size for read
	ReadBufferSize = 1024

	// Maximum buffer size for write
	WriteBufferSize = 1024
)

func (c *CLient) ReadPump() {
	defer func() {
		c.Hub.unregister <- c.User.Id
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.AbnormalClosure) {
				log.Printf("error: %v\n", err)
			}
			break
		}
		c.Hub.ReadAllFromUser <- c.User.Id
	}
}

func (c *Client) WritePump {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case notif, ok := <- c.User.sendNotif:
			c.Conn.SetWriteDeadline(writeWait)
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("error: %v\n", err)
				return
			}
			notifJSON, err := json.Marshal(notif)
			if err != nil {
				log.Printf("error: %v\n", err)
				return
			}
			w.Write(notifJSON)
			if len(c.User.sendNotif) >= 1 {
				notifsJSON := mergeNotifs(c.User.sendNotif)
				for notifJSON = range notifsJSON {
					w.Write(notifJSON)
				}				
			}
			if err = w.Close(); err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
		case ok := <- c.User.sendOk:
			result := strconv.FormatBool(ok)
			msg := []byte(`{"ok":"`+result+`"}`)
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case <- ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, []byte()); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		}
	}
}

// mergeNotifs receives notifications from the buffered channel and discards all
// the notifications (type *pbDataFormat.Notif) with the same ID but the last
// notification, then marshals the resulting notifications into JSON []byte, using
// Marshal from encoding/json and returns a []byte for each notification.
func mergeNotifs(receive chan *pbDataFormat.Notif) (notifsJSON [][]byte) {
	n := len(receive)
	notifs := []*pbDataFormat.Notif{}
	// fill notifs slice
	for i := 0; i < n; i++ {
		notifs = append(notifs, <-receive)
	}
	// merged will contain only the last duplicate notif, if there are duplicates
	merged := make(map[string]*pbDataFormat.Notif)
	for notif := range notifs {
		merged[notif.Id] = notif
	}
	// empty notifs slice
	notifs = []*pbDataFormat.Notif{}
	// fill again notifs slice
	for _, notif := range merged {
		notifs = append(notifs, notif)
	}
	// marshal into JSON and fill slice of JSON notifs
	for notif := range notifs {
		notifJSON, err := json.Marshal(notif)
		if err != nil {
			log.Printf("Could not marshal notif: %v\n", err)
			continue
		}
		notifsJSON = append(notifsJSON, notifJSON)
	}
	return
}