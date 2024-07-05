package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for this example
	},
}

func handleWebSocket_h(c *gin.Context) {
	handleWebSocketConnection(c, sendConceptList)
}

func handlePeerWebSocket_h(c *gin.Context) {
	handleWebSocketConnection(c, sendPeerList)
}

func handleWebSocketConnection(c *gin.Context, sendFunc func(*websocket.Conn)) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("New WebSocket connection established")

	sendFunc(conn)

	go periodicSend(conn, sendFunc)

	keepAlive(conn)

	log.Printf("WebSocket connection closed")
}
