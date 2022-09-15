package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	c "github.com/ostafen/clover"
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type Client struct {
	hub *WsHub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

type WsHub struct {
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	clients    map[*Client]bool
}
type MessageContainer struct {
	Data CoolMessage `json:"alldata"`
}
type CoolMessage struct {
	Text string `json:"text"`
	Time int64  `json:"time"`
	User string `json:"user"`
}
type CoolMessageDb struct {
	Text string `clover:"text"`
	Time int64  `clover:"time"`
	User string `clover:"user"`
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		if err != nil {
			c.conn.Close()
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		println("message!" + string(message))
		//var message2 = []byte("hello lol!!!")

		c.hub.broadcast <- message
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *WsHub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			println("clients now:", len(h.clients))

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			var storehere CoolMessage
			parseerr := json.Unmarshal(message, &storehere)
			if parseerr != nil {
				print("cant parse the incoming msg")
				break
			}

			var coool = CoolMessage{string(storehere.Text), time.Now().Unix(), storehere.User}
			var cooolDb = CoolMessageDb{string(storehere.Text), time.Now().Unix(), storehere.User}

			db, opendberr := c.Open("clover-db")
			if opendberr != nil {
				fmt.Println(fmt.Printf("Error %v", opendberr))
			}
			doc := c.NewDocument()
			doc.Set("text", cooolDb.Text)
			doc.Set("user", cooolDb.User)
			doc.Set("time", cooolDb.Time)

			// InsertOne returns the id of the inserted document
			docId, inserterr := db.InsertOne("messages", doc)
			if inserterr != nil {
				fmt.Println(fmt.Printf("Error %v", inserterr))
			}
			fmt.Println(docId)

			db.Close()
			var messages []CoolMessage
			messages = append(messages, coool)
			var coolmsgbytes, err = json.Marshal(messages)
			if err != nil {
				break
			}

			for client := range h.clients {
				select {
				case client.send <- coolmsgbytes:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func main() {
	hub := &WsHub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}

	db, _ := c.Open("clover-db")
	hasMessages, err := db.HasCollection("messages")
	if err != nil {
		panic("cant get collection")
	}
	if !hasMessages {
		db.CreateCollection("messages")
	}
	db.Close()

	go hub.run()

	http.HandleFunc("/ws", func(rw http.ResponseWriter, r *http.Request) {
		wsEndpoint(rw, r, hub)
	})
	http.HandleFunc("/people", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, fmt.Sprint((len(hub.clients))))
	})
	log.Println("Running golang backend!")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func wsEndpoint(w http.ResponseWriter, r *http.Request, hub *WsHub) {

	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	// upgrade this connection to a WebSocket
	// connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}
	log.Println("Client Connected")

	// err = ws.WriteMessage(1, []byte("Hi Client!"))
	// if err != nil {
	// 	log.Println(err)
	// }
	// reader(ws)

	client := &Client{hub: hub, conn: ws, send: make(chan []byte, 256)}
	fmt.Printf("welcome %+v\n", client.conn.RemoteAddr())

	client.hub.register <- client

	db, _ := c.Open("clover-db")
	docs, err := db.Query("messages").Sort(c.SortOption{"time", -1}).FindAll()
	var dbmessages []CoolMessage
	for _, doc := range docs {
		print(doc)
		//doc.Unmarshal(&storehere)
		//fmt.Printf("have?.. %v", doc.Unmarshal(&storehere)) doesnt work
		timeposted, ok := doc.Get("time").(int64)

		if !ok {
			panic("cant parse the time")
		}
		var coool = CoolMessage{string(fmt.Sprintf("%v", doc.Get("text"))), timeposted, fmt.Sprintf("%v", doc.Get("user"))}
		dbmessages = append(dbmessages, coool)

	}

	marshal, marshalerr := json.Marshal(dbmessages)
	if marshalerr != nil {
		panic("err")
	}
	client.send <- marshal

	if err != nil {
		log.Println("Error!")
	}
	db.Close()

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()

}

func reader(conn *websocket.Conn) {
	for {
		// read in a message
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		// print out incoming message
		fmt.Println("incoming message: " + string(p))
	}
}
