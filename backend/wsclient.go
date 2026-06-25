package main

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

func main() {
	c, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/api/v1/ws", nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	msg := `{"type":"translate","payload":{"text":"How are you?","engine":"gemini"}}`
	err = c.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		log.Fatal("write:", err)
	}

	for i := 0; i < 4; i++ {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
	}
}
