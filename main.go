package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
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

	http.HandleFunc("/signaling", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade failed: ", err)
			return
		}
		defer conn.Close()

		conn.SetReadLimit(MAX_MESSAGE_BYTES)

	MessageLoop:
		for {
			receivedMessage := Message{}
			err := conn.ReadJSON(&receivedMessage)
			if err != nil {
				log.Println("read failed:", err)
				break MessageLoop
			}

			switch receivedMessage.Type {
			case "ping":
				messageToSend := Message{
					Type: "pong",
				}
				err := conn.WriteJSON(&messageToSend)
				if err != nil {
					log.Println("write failed:", err)
					break MessageLoop
				}

			}
		}
	})

	log.Fatal(http.ListenAndServe(addr, nil))
}
