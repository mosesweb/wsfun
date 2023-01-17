package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	vision "cloud.google.com/go/vision/apiv1"
	firebase "firebase.google.com/go"
	"github.com/gorilla/websocket"
	"google.golang.org/api/iterator"
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
	fsclient   *firestore.Client
	ctx        context.Context
}
type MessageContainer struct {
	Data CoolMessage `json:"alldata"`
}
type CoolMessage struct {
	Text string `json:"text"`
	Time int64  `json:"time"`
	User string `json:"user"`
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

			_, err := h.fsclient.Collection("messages").NewDoc().Set(h.ctx, coool)
			if err != nil {
				// Handle any errors in an appropriate way, such as returning them.
				log.Printf("An error has occurred: %s", err)
			}

			var messages []CoolMessage
			messages = append(messages, coool)
			var coolmsgbytes, perr = json.Marshal(messages)
			if perr != nil {
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

	// Use the application default credentials
	ctx := context.Background()
	conf := &firebase.Config{ProjectID: "mychat-359818"}
	firestoreApp, err := firebase.NewApp(ctx, conf)
	if err != nil {
		panic("not good firebase.NewApp cant be loaded")
	}
	firestoreclient, err := firestoreApp.Firestore(ctx)
	if err != nil {
		panic("not good firestore cant be loaded")
	}

	hub := &WsHub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		fsclient:   firestoreclient,
		ctx:        ctx,
	}

	if err != nil {
		log.Fatalln(err)
	}

	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("INIT FIRESTORE!")
	log.Println("INITED FIRESTORE")

	go hub.run()

	http.HandleFunc("/ws", func(rw http.ResponseWriter, r *http.Request) {
		wsEndpoint(rw, r, hub)
	})
	http.HandleFunc("/people", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, fmt.Sprint((len(hub.clients))))
	})

	http.HandleFunc("/imagefile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Println("GO!")
		ReceiveFile(w, r)
		_, _, err := r.FormFile("image")
		if err != nil {
			fmt.Println("NOT GOOD!")
			panic(err)
		}

	})
	log.Println("Running golang backend!")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
func ReceiveFile(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20) // limit your max input length!
	var buf bytes.Buffer
	// in your case file would be fileupload
	file, header, err := r.FormFile("image")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	name := strings.Split(header.Filename, ".")
	fmt.Printf("File name %s\n", name[0])
	// Copy the file data to my buffer
	_, errr := io.Copy(&buf, file)
	if errr != nil {
		panic(errr)
	}
	// do something with the contents...
	// I normally have a struct defined and unmarshal into a struct, but this will
	// work as an example
	contents := buf.String()
	//fmt.Println(contents)
	// I reset the buffer in case I want to use it again
	// reduces memory allocations in more intense projects

	myReader := strings.NewReader(contents)
	fmt.Fprintln(w, "lets go..1")

	detectErr := detectText(w, myReader)
	if detectErr != nil {
		panic(detectErr)
	}
	buf.Reset()
	// do something else
	// etc write header
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
	var dbmessages []*CoolMessage

	iter := hub.fsclient.Collection("messages").OrderBy("Time", firestore.Desc).Limit(50).Documents(hub.ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		fmt.Println(doc.Data())

		var c CoolMessage
		doc.DataTo(&c)
		fmt.Printf("Document data: %#v\n", c)
		//var coool = CoolMessage{string(fmt.Sprintf("%v", doc.Data("")("text"))), timeposted, fmt.Sprintf("%v", doc.Get("user"))}
		dbmessages = append(dbmessages, &c)
	}

	// docs, err := db.Query("messages").Sort(c.SortOption{"time", -1}).FindAll()
	// for _, doc := range docs {
	// 	print(doc)
	// 	//doc.Unmarshal(&storehere)
	// 	//fmt.Printf("have?.. %v", doc.Unmarshal(&storehere)) doesnt work
	// 	timeposted, ok := doc.Get("time").(int64)

	// 	if !ok {
	// 		panic("cant parse the time")
	// 	}
	// 	var coool = CoolMessage{string(fmt.Sprintf("%v", doc.Get("text"))), timeposted, fmt.Sprintf("%v", doc.Get("user"))}
	// 	dbmessages = append(dbmessages, coool)

	// }

	marshal, marshalerr := json.Marshal(dbmessages)
	if marshalerr != nil {
		panic("err")
	}
	client.send <- marshal

	if err != nil {
		log.Println("Error!")
	}

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

// detectText gets text from the Vision API for an image at the given file path.
func detectText(w io.Writer, reader io.Reader) error {
	ctx := context.Background()
	fmt.Fprintln(w, "lets go..")

	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return err
	}

	image, err := vision.NewImageFromReader(reader)
	if err != nil {
		return err
	}
	annotations, err := client.DetectTexts(ctx, image, nil, 10)
	if err != nil {
		return err
	}

	if len(annotations) == 0 {
		fmt.Fprintln(w, "No text found.")
	} else {
		fmt.Fprintln(w, "Text:")
		for _, annotation := range annotations {
			fmt.Fprintf(w, "%q\n", annotation.Description)
		}
	}

	return nil
}
