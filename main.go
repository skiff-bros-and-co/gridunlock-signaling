package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	cmap "github.com/orcaman/concurrent-map/v2"
)

const MAX_MESSAGE_BYTES = 1024

type Message struct {
	Type   string                 `json:"type"`
	Topics []string               `json:"topics,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			log.Println("Received empty origin")
			return false
		}

		// local dev builds
		if origin == " http://localhost:5173/" {
			return true
		}

		parsedOrigin, err := url.Parse(origin)
		if err != nil {
			log.Println("Received invalid origin:", origin)
			return false
		}

		if parsedOrigin.Scheme != "https" {
			log.Println("Received invalid origin:", origin)
			return false
		}

		if parsedOrigin.Hostname() == "gridunlockapp.com" {
			return true
		}

		// Dev builds
		if strings.HasSuffix(parsedOrigin.Hostname(), ".gridunlock-org.pages.dev") {
			return true
		}

		log.Println("Received invalid origin:", origin)
		return false
	},
}

func main() {
	buildType := os.Getenv("BUILD_TYPE")
	var addr string
	if buildType == "DEV" {
		addr = "127.0.0.1:8080"
		log.Println("Running in DEV mode")
	} else {
		addr = ":8080"
		log.Println("Running in PROD mode")
	}

	log.Println("Server started at http://localhost:8080")

	subscribers := cmap.New[[]*websocket.Conn]()

	http.HandleFunc("/signaling", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade failed: ", err)
			return
		}
		defer conn.Close()

		// Clear any subscriptions on disconnect
		defer func() {
			for _, topic := range subscribers.Keys() {
				removeSubscriber(&subscribers, topic, conn)
			}
		}()

		conn.SetReadLimit(MAX_MESSAGE_BYTES)
		messageLoop(conn, &subscribers)
	})

	log.Fatal(http.ListenAndServe(addr, nil))
}

func messageLoop(conn *websocket.Conn, subscribers *cmap.ConcurrentMap[string, []*websocket.Conn]) {
	for {
		receivedMessage := Message{}
		err := conn.ReadJSON(&receivedMessage)
		if err != nil {
			log.Println("read failed:", err)
			return
		}

		switch receivedMessage.Type {
		case "ping":
			messageToSend := Message{
				Type: "pong",
			}
			err := conn.WriteJSON(&messageToSend)
			if err != nil {
				log.Println("write failed:", err)
				return
			}
		case "subscribe":
			if receivedMessage.Topics == nil || len(receivedMessage.Topics) != 1 {
				log.Println("received invalid subscribe message", receivedMessage)
				return
			}
			addSubscriber(subscribers, receivedMessage.Topics[0], conn)
		case "unsubscribe":
			removeSubscriber(subscribers, receivedMessage.Topics[0], conn)
		case "publish":
			if receivedMessage.Topics == nil || len(receivedMessage.Topics) != 1 {
				log.Println("received invalid publish message", receivedMessage)
				return
			}
			peers, exists := subscribers.Get(receivedMessage.Topics[0])
			if !exists {
				log.Println("received publish message for non-existent topic", receivedMessage)
				return
			}
			for _, peer := range peers {
				if peer == conn {
					continue
				}

				err := peer.WriteJSON(&receivedMessage)
				if err != nil {
					log.Println("write to peer failed:", err)
				}
			}
		}
	}
}

func removeSubscriber(subscribers *cmap.ConcurrentMap[string, []*websocket.Conn], topic string, conn *websocket.Conn) {
	subscribers.Upsert(topic, []*websocket.Conn{}, func(exists bool, valueInMap []*websocket.Conn, newValue []*websocket.Conn) []*websocket.Conn {
		if exists {
			for i, subscriber := range valueInMap {
				if subscriber == conn {
					return append(valueInMap[:i], valueInMap[i+1:]...)
				}
			}
		}
		return valueInMap
	})

	// Remove topic if no subscribers
	subscribers.RemoveCb(topic, func(_ string, valueInMap []*websocket.Conn, exists bool) bool {
		return exists && len(valueInMap) == 0
	})
}

func addSubscriber(subscribers *cmap.ConcurrentMap[string, []*websocket.Conn], topic string, conn *websocket.Conn) {
	subscribers.Upsert(topic, []*websocket.Conn{conn}, func(exists bool, valueInMap []*websocket.Conn, newValue []*websocket.Conn) []*websocket.Conn {
		if exists {
			return append(valueInMap, conn)
		} else {
			return newValue
		}
	})
}
