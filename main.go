package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}
var todoList []string

func getCmd(input string) string {
	inputArr := strings.Split(input, " ")
	return inputArr[0]
}

func getMessage(input string) string {
	inputArr := strings.Split(input, " ")
	var result string
	for i := 1; i < len(inputArr); i++ {
		result += inputArr[i]
	}
	return result
}

func updateTodoList(input string) {
	tmpList := todoList
	todoList = []string{}
	for _, val := range tmpList {
		if val == input {
			continue
		}
		todoList = append(todoList, val)
	}
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

	http.HandleFunc("/todo", func(w http.ResponseWriter, r *http.Request) {
		// Upgrade upgrades the HTTP server connection to the WebSocket protocol.
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade failed: ", err)
			return
		}
		defer conn.Close()

		// Continuosly read and write message
		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read failed:", err)
				break
			}
			input := string(message)
			cmd := getCmd(input)
			msg := getMessage(input)
			if cmd == "add" {
				todoList = append(todoList, msg)
			} else if cmd == "done" {
				updateTodoList(msg)
			}
			output := "Current Todos: \n"
			for _, todo := range todoList {
				output += "\n - " + todo + "\n"
			}
			output += "\n----------------------------------------"
			message = []byte(output)
			err = conn.WriteMessage(mt, message)
			if err != nil {
				log.Println("write failed:", err)
				break
			}
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "websockets.html")
	})

	log.Fatal(http.ListenAndServe(addr, nil))
}
