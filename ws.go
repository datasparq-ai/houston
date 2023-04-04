package main

import (
	"encoding/json"
	"github.com/datasparq-ai/houston/model"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

type message struct {
	key     string
	Event   string `json:"event"`
	Content []byte `json:"content"`
}

func (m *message) Bytes() []byte {
	missionString := make(map[string]interface{})
	missionString["event"] = m.Event
	var contentAsJSObject map[string]interface{}
	// convert bytes to JSON serializable object so that it is sent as JSON
	err := json.Unmarshal(m.Content, &contentAsJSObject)
	if err != nil {
		// if message content can't be represented as JSON then send string
		missionString["content"] = string(m.Content)
	} else {
		missionString["content"] = contentAsJSObject
	}
	b, _ := json.Marshal(missionString)
	return b
}

// WebSocketHub maintains the set of active clients and broadcasts messages to the clients.
// clients are grouped by key and are expected to provide the key when creating a connection.
type WebSocketHub struct {
	clients    map[string]map[*WebSocketClient]bool // key -> client -> bool
	broadcast  chan message
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
}

func newWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		broadcast:  make(chan message),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		clients:    make(map[string]map[*WebSocketClient]bool),
	}
}

func (h *WebSocketHub) run() {
	for {
		select {
		case client := <-h.register:
			if h.clients[client.key] == nil {
				h.clients[client.key] = make(map[*WebSocketClient]bool)
			}
			h.clients[client.key][client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client.key][client]; ok {
				delete(h.clients[client.key], client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for key := range h.clients {
				// only broadcast to clients belonging to this key
				if message.key == key {
					for client := range h.clients[key] {
						select {
						case client.send <- message.Bytes():
						default:
							close(client.send)
							delete(h.clients[key], client)
						}
					}
				}
			}
		}
	}
}

// WebSocketClient is a middleman between the websocket connection and the hub.
// We only want to send messages relating to a key if the client has that key.
type WebSocketClient struct {
	id   string
	hub  *WebSocketHub
	conn *websocket.Conn
	send chan []byte
	key  string
}

// upgrader specifies parameters for upgrading an HTTP connection to a WebSocket connection.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (a *API) initWebSocket() {
	ws := newWebSocketHub()
	a.ws = ws.broadcast
	go ws.run()

	//a.router.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
	//  // TODO: create a temp token to correspond to the API key
	//  token := createRandomString(40)
	//  expiry := time.Now().Add(time.Hour)
	//  a.router.
	//    w.Write([]byte(token))
	//})

	a.router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {

		//TODO: websocket has no authentication methods so key in query params is being used.
		//      could replace with token.
		key := r.URL.Query().Get("a")

		// upgrade from http to ws protocol
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		if key == "" {
			payload, _ := json.Marshal(&model.KeyNotProvidedError{})
			conn.WriteMessage(websocket.TextMessage, payload)
			conn.Close()
			return
		}

		// check that key exists
		_, ok := a.db.Get(key, "u")
		if !ok {

			payload, _ := json.Marshal(&model.KeyNotFoundError{})
			conn.WriteMessage(websocket.TextMessage, payload)
			conn.Close()
			return
		}

		client := &WebSocketClient{id: "", hub: ws, conn: conn, send: make(chan []byte, 256), key: key}
		client.hub.register <- client

		// allow collection of memory referenced by the caller by doing all work in new goroutines
		go client.writePump()
		go client.readPump()

		a.ws <- message{key, "notice", []byte("New client connected")}

	})
}

// readPump pumps messages from the websocket connection to the hub.
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *WebSocketClient) readPump() {
	defer func() {
		//log.Printf("CONNECTION CLOSED - %s\n", c.id)
		c.hub.unregister <- c

		c.conn.Close()
	}()

	// send the number of connections to all
	//data := []byte(fmt.Sprintf("{\"type\":\"connectionCount\",\"data\":%v}", len(c.hub.clients)))
	//c.hub.broadcast <- data

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				//log.Printf("error: %v", err)
			}
			break
		}

		//log.Printf("MESSAGE %s - %s\n", c.id, msg)

		c.hub.broadcast <- message{c.key, "notice", msg}
	}
}

// writePump pumps messages from the hub to the websocket connection.
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(50 * time.Second) // must be more frequent than read deadline
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
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
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
