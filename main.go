package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/novalagung/gubrak/v2"
)

type M map[string]interface{}

const MESSAGE_NEW_USER = "New User"
const MESSAGE_CHAT = "Chat"
const MESSAGE_LEAVE = "Leave"

var connections = make([]*WebSocketConnection, 0)

type SocketPayload struct {
	Message string
}

type SocketResponse struct {
	From    string
	Type    string
	Message string
}

type WebSocketConnection struct {
	*websocket.Conn
	Username string
}

func handleIO(currentCon *WebSocketConnection, connections []*WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR", fmt.Sprintf("%v", r))
		}
	}()

	broadcastMessage(currentCon, MESSAGE_NEW_USER, "")

	for {
		payload := SocketPayload{}
		err := currentCon.ReadJSON(&payload)
		if err != nil {
			if strings.Contains(err.Error(), "websocket: close") {
				broadcastMessage(currentCon, MESSAGE_LEAVE, "")
				ejectConnection(currentCon)
				return
			}
			log.Println("ERROR", err.Error())
			continue
		}
		broadcastMessage(currentCon, MESSAGE_CHAT, payload.Message)
	}
}

func ejectConnection(currentConn *WebSocketConnection) {
	filtered := gubrak.From(connections).Reject(func(each *WebSocketConnection) bool {
		return each == currentConn
	}).Result()
	connections = filtered.([]*WebSocketConnection)
}

func broadcastMessage(currentConn *WebSocketConnection, kind, message string) {
	for _, eachConn := range connections {
		if eachConn == currentConn {
			continue
		}

		eachConn.WriteJSON(SocketResponse{
			From:    currentConn.Username,
			Type:    kind,
			Message: message,
		})
	}
}

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		content, err := os.ReadFile("index.html")
		if err != nil {
			http.Error(w, "tidak bisa membaca file", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%s", content)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		currentGorillaCon, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
		if err != nil {
			http.Error(w, "Tidak bisa membuka websocket", http.StatusBadRequest)
			return
		}
		username := r.URL.Query().Get("username")
		currentCon := WebSocketConnection{Conn: currentGorillaCon, Username: username}
		connections = append(connections, &currentCon)

		go handleIO(&currentCon, connections)
	})

	fmt.Println("Server berjalan pada http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
