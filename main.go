package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write the response to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	log.Println("Starting server on port 8080")
	http.HandleFunc("/", serveWs)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	log.Println("Connection made")

	//Upgrade the http connection to a websocket
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	ws, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return // If there's an error we want to bail
	}

	connection := Connection{websocket: *ws}
	connection.init()
}

func wsReader(ws *websocket.Conn, mc chan string) {
	defer ws.Close()

	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			log.Println("Error when reading websocket message")
			log.Println(err)
			break
		}
		log.Println("Recieved message: " + string(msg))
		mc <- string(msg)
	}
}

func wsWriter(ws *websocket.Conn, mc chan string) {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		ws.Close()
	}()
	for {
		select {
		case msg := <- mc:
			log.Println("Writing: " + msg)
			if err := ws.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				return
			}
		case <-pingTicker.C:
			log.Println("Sending ping")
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

type Connection struct {
	websocket websocket.Conn
}

func (c *Connection) init() {
	log.Println("Initialising connection")

	messageChannel := make(chan string)

	//Create readers and writers
	go wsWriter(&c.websocket, messageChannel)
	go wsReader(&c.websocket, messageChannel)
}