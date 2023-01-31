package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	b64 "encoding/base64"

	"cloud.google.com/go/firestore"
	vision "cloud.google.com/go/vision/apiv1"
	"cloud.google.com/go/vision/v2/apiv1/visionpb"
	firebase "firebase.google.com/go"
	"github.com/gorilla/websocket"
	"github.com/twpayne/go-geom"
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
	Data ChatMessage `json:"alldata"`
}
type ChatMessage struct {
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
			var storehere ChatMessage
			parseerr := json.Unmarshal(message, &storehere)
			if parseerr != nil {
				print("cant parse the incoming msg")
				break
			}

			var coool = ChatMessage{string(storehere.Text), time.Now().Unix(), storehere.User}

			_, err := h.fsclient.Collection("messages").NewDoc().Set(h.ctx, coool)
			if err != nil {
				// Handle any errors in an appropriate way, such as returning them.
				log.Printf("An error has occurred: %s", err)
			}

			var messages []ChatMessage
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

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "nothing fun here")
	})

	http.HandleFunc("/ws", func(rw http.ResponseWriter, r *http.Request) {
		wsEndpoint(rw, r, hub)
	})

	http.HandleFunc("/imagefile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		fmt.Println("GO!")
		_, _, err := r.FormFile("image")
		switch err {
		case nil:
			ReceiveFile(w, r)
		case http.ErrMissingFile:
			log.Println("no file")
			return
		default:
			log.Println(err)
		}
	})

	log.Println("Running golang backend!")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type ImageFixResponse struct {
	Image string `json:"Image"`
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

	detresult, detectErr := detectText(w, myReader)
	//fmt.Printf("SEND THIS %s", detresult)
	jsonBody := []byte(detresult)
	bodyReader := bytes.NewReader(jsonBody)
	// Go to .NET to get back painted picture
	resp, err := http.Post(fmt.Sprintf("https://localhost:44397/ImageFixer?Image=%s", "lul"), "application/json", bodyReader)
	if err != nil {
		fmt.Println("NOT GOOD........")
		panic(err)
	}
	body, resperror := ioutil.ReadAll(resp.Body)
	if resperror != nil {
		fmt.Println(resp.Body)
		panic(resperror)
	}
	defer resp.Body.Close()

	var imageFix ImageFixResponse
	//fmt.Println(fmt.Sprintf("body: %v", string(body)))
	decodeerr := json.Unmarshal(body, &imageFix)
	if err != nil {
		panic(decodeerr)
	}
	picbytes, _ := b64.StdEncoding.DecodeString(imageFix.Image)

	//picbytes := []byte(imageFix.Image)
	reader := bytes.NewReader(picbytes)
	decodedIntoImage, errJpg := jpeg.Decode(reader)
	if errJpg != nil {
		return
	}
	if err != nil {
		return
	}

	w2, err := os.Create("outputFile.jpg")
	if err != nil {
		return
	}
	err = jpeg.Encode(w2, decodedIntoImage, nil)

	var returnObj ImageitemResponse
	var imageitemparsed Imageitem
	returnObj.Image = string(picbytes)
	returnObj.Texts = imageitemparsed.Texts

	// jsonagain, marshallerr := json.Marshal(returnObj)
	// if marshallerr != nil {
	// 	panic("cant marshal")
	// }
	// Print back the base64 picture as string from bytes
	fmt.Fprintf(w, detresult)
	if err != nil {
		return
	}

	//fmt.Println(string(imageFix.Image))
	//fmt.Println(fmt.Sprintf("imagefix: %v", imageFix))

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
	var dbmessages []*ChatMessage

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

		var c ChatMessage
		doc.DataTo(&c)
		fmt.Printf("Document data: %#v\n", c)
		//var coool = CoolMessage{string(fmt.Sprintf("%v", doc.Data("")("text"))), timeposted, fmt.Sprintf("%v", doc.Get("user"))}
		dbmessages = append(dbmessages, &c)
	}

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

type Imageitem struct {
	Image []byte     `json:"image"`
	Texts []TextInfo `json:"textInfo"`
}
type ImageitemResponse struct {
	Image string     `json:"image"`
	Texts []TextInfo `json:"textInfo"`
}
type TextInfo struct {
	Text         string                 `json:"text"`
	Boundingpoly *visionpb.BoundingPoly `json:"boundingpoly"`
}

func addBlackRectangle(theimage image.Image, x1, y1, x2, y2 int) (err error) {
	rect1 := theimage.Bounds()
	rect2 := image.Rect(x1, y1, x2, y2)
	rect2.Inset(-10)
	if !rect2.In(rect1) {
		err = fmt.Errorf("error: rectangle outside image")
		return
	}
	rgba := image.NewRGBA(rect1)
	for x := rect1.Min.X; x <= rect1.Max.X; x++ {
		for y := rect1.Min.Y; y <= rect1.Max.Y; y++ {
			p := image.Pt(x, y)
			if p.In(rect2) {
				rgba.Set(x, y, color.Black)
			} else {
				rgba.Set(x, y, theimage.At(x, y))
			}
		}
	}

	outputFile := "hej.jpg"
	w, err := os.Create(outputFile)
	defer w.Close()

	err = jpeg.Encode(w, rgba, nil)
	return
}

func drawRectangle(img draw.Image, color color.Color, poly *geom.Polygon) {

	cords := poly.Coords()
	for _, c := range cords {
		//fmt.Println((fmt.Sprintf("%v", c)))

		for i, v := range c {
			// fmt.Println((fmt.Sprintf("this is what we got for X: %v", int(v.Clone().X()))))
			// fmt.Println((fmt.Sprintf("this is what we got for Y: %v", int(v.Clone().Y()))))
			img.Set(i, int(v.Clone().X()), color)
			img.Set(i, int(v.Clone().Y()), color)

			fmt.Println((fmt.Sprintf("next...")))

		}

	}
	// for i := x1; i < x2; i++ {
	// 	img.Set(i, y1, color)
	// 	img.Set(i, y2, color)
	// }

	// for i := y1; i <= y2; i++ {
	// 	img.Set(x1, i, color)
	// 	img.Set(x2, i, color)
	// }
}

func addRectangleToFace(img draw.Image, poly *geom.Polygon) draw.Image {
	myColor := color.RGBA{255, 0, 255, 255}

	//min := poly.Min
	//max := poly.Max

	drawRectangle(img, myColor, poly)

	return img
}

// detectText gets text from the Vision API for an image at the given file path.
func detectText(w io.Writer, reader io.Reader) (string, error) {
	ctx := context.Background()

	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return "", err
	}

	visionimage, err := vision.NewImageFromReader(reader)
	if err != nil {
		return "", err
	}
	annotations, err := client.DetectTexts(ctx, visionimage, nil, 10)
	if err != nil {
		return "", err
	}

	textInfos := make([]TextInfo, 0)

	if len(annotations) == 0 {
		fmt.Fprintln(w, "No text found.")
	} else {

		//rects = make([]img.Rectangle, 0)

		result := &Imageitem{
			Image: visionimage.Content,
		}

		decodedvisionImage, errJpg := jpeg.Decode(bytes.NewReader(visionimage.Content))

		// convert as usable image
		b := decodedvisionImage.Bounds()
		drawedvisionimage := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		draw.Draw(drawedvisionimage, drawedvisionimage.Bounds(), decodedvisionImage, b.Min, draw.Src)

		//err := addBlackRectangle(decodedvisionImage, 100, 100, 200, 200)

		for _, annotation := range annotations {
			if annotation.Description == "" || annotation.BoundingPoly == nil {
				continue
			}
			obj := &TextInfo{
				Text:         annotation.Description,
				Boundingpoly: annotation.BoundingPoly,
			}

			// vert := annotation.BoundingPoly.GetVertices()

			// // poly := geom.NewPolygon(geom.XY).MustSetCoords([][]geom.Coord{
			// // 	{{float64(vert[0].X), float64(vert[0].Y)}, {float64(vert[1].X), float64(vert[1].Y)}, {float64(vert[2].X), float64(vert[2].X)}},
			// // })
			// // //myRectangle := image.Rect(int(vert[0].X), int(vert[0].Y), int(vert[len(vert)-1].X), int(vert[len(vert)-1].Y))

			// // imgres := addRectangleToFace(drawedvisionimage, poly)
			// // dst := addRectangleToFace(imgres, poly)

			// outputFile, err := os.Create("dst.png")
			// png.Encode(outputFile, dst)

			//outputFile.Close()
			if err != nil {
				panic(err.Error())
			}

			textInfos = append(textInfos, *obj)
		}

		if err != nil {
			log.Fatal(err)
		}
		if errJpg != nil {
			fmt.Println(err)
			return "", nil
		}
		//imgdraw.DrawMask(jpgI, img1.Bounds(), img1, img.Point{0, 0}, draw.Src)

		result.Texts = textInfos
		result.Image = visionimage.Content

		//fmt.Fprintf(w, "%v", textInfos)
		ba, err := json.Marshal(result)
		if err != nil {
			fmt.Println(err)
		}

		// Print result back to writer

		// https://localhost:44397/

		//fmt.Fprintf(w, "%v", )
		fmt.Println(fmt.Sprintln("sending", string(ba)))
		return string(ba), nil
	}

	return "", nil
}
