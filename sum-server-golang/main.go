package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"

	"github.com/gorilla/websocket"
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
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		print("message!" + string(message))
		myobj := messageToClient{
			Message: string(message),
			Date:    "thing",
		}
		mymsg, err := json.Marshal(myobj)
		if err != nil {
			print("error!")
			print(err)
		}
		c.hub.addToDb(myobj.Message, myobj.Date)
		c.hub.broadcast <- mymsg
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.

type messageToClient struct {
	Date    string `json:"date"`
	Message string `json:"message"`
}

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
			print("register!")
			h.clients[client] = true

			initData := h.selectFromDb()
			// convert int64 to []byte
			//buf := make([]byte, initData)
			newmsg := messageToClient{
				Message: fmt.Sprintf("You are behind... %v", initData),
				Date:    "2022-06-04",
			}

			jsonobj, err := json.Marshal(newmsg)
			checkErr(err)
			client.send <- jsonobj

			print("HERE THEN", len(h.clients))
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *WsHub) addToDb(message string, date string) {
	db, err := sql.Open("sqlite3", "./db.db")
	checkErr(err)
	println("add to db!")
	stmt, err := db.Prepare("INSERT INTO messages (Message, Date) values(?,?)")
	checkErr(err)
	res, err := stmt.Exec(message, date)
	checkErr(err)
	print(res.RowsAffected())
	db.Close()
	h.selectFromDb()
}

func (h *WsHub) selectFromDb() int {
	db, err := sql.Open("sqlite3", "./db.db")
	checkErr(err)
	res, err := db.Query("SELECT * FROM messages")
	checkErr(err)
	checkErr(err)
	var messages []*messageToClient
	for res.Next() {
		m := new(messageToClient)
		res.Scan(&m.Message, &m.Date) // scan contents of the current row into the instance

		messages = append(messages, m)
	}
	db.Close()

	outdata := len(messages)
	fmt.Println("last..")
	fmt.Println(outdata)
	checkErr(err)
	return outdata
}

func (h *WsHub) DbSetup() {
	db, err := sql.Open("sqlite3", "./db.db")
	checkErr(err)
	res, err := db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		Message TEXT NOT NULL,
		Date TEXT NOT NULL
	)`)
	checkErr(err)
	print(res.RowsAffected())
	db.Close()
}
func checkErr(err error) {
	if err != nil {
		print(err)
		panic(err)
	}
}
func main() {
	hub := &WsHub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
	go hub.run()
	hub.DbSetup()
	http.HandleFunc("/ws", func(rw http.ResponseWriter, r *http.Request) {
		wsEndpoint(rw, r, hub)
	})

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
	print("clients: ", len(hub.clients))
	// err = ws.WriteMessage(1, []byte("Hi Client!"))
	// if err != nil {
	// 	log.Println(err)
	// }
	// reader(ws)

	client := &Client{hub: hub, conn: ws, send: make(chan []byte, 256)}
	client.hub.register <- client

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

		// if err := conn.WriteMessage(messageType, p); err != nil {
		// 	log.Println(err)
		// 	return
		// }
	}
}
