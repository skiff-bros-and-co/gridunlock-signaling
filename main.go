package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/olahol/melody"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type Message struct {
	Type   string                 `json:"type"`
	Topics []string               `json:"topics,omitempty"`
	Topic  string                 `json:"topic,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

const MAX_MESSAGE_BYTES = 1024

var PONG_MESSAGE = []byte("{\"type\":\"pong\"}")

var clientIdCounter uint64 = 0

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

	m := melody.New()
	m.Config.MaxMessageSize = MAX_MESSAGE_BYTES
	m.Upgrader.CheckOrigin = validateOrigin

	http.HandleFunc("/signaling", func(w http.ResponseWriter, r *http.Request) {
		m.HandleRequest(w, r)
	})

	m.HandleConnect(func(s *melody.Session) {
		id := atomic.AddUint64(&clientIdCounter, 1)
		idString := strconv.FormatUint(id, 10)
		s.Set("id", idString)
		log.Println("client connected:", idString)
	})

	subscribers := cmap.New[[]*melody.Session]()
	m.HandleClose(func(s *melody.Session, i int, s2 string) error {
		id, idWasSet := s.Get("id")
		var idLogString string
		if idWasSet {
			log.Println("client disconnected:", id)
			idLogString = "(clientId:" + id.(string) + ")"
		} else {
			log.Println("client disconnected: unknown id")
			idLogString = "(unknownId)"
		}

		topic, topicWasSet := s.Get("topic")
		if topicWasSet {
			removeSubscriber(&subscribers, topic.(string), s, idLogString)
		}

		return nil
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		id, idWasSet := s.Get("id")
		if !idWasSet {
			log.Println("received message from unknown client")
			return
		}
		idLogString := "(clientId:" + id.(string) + ")"

		var receivedMessage Message
		err := json.Unmarshal(msg, &receivedMessage)
		if err != nil {
			log.Println("received invalid message:", string(msg), idLogString)
			return
		}

		processMessage(&subscribers, s, receivedMessage, msg, idLogString)
	})

	log.Fatal(http.ListenAndServe(addr, nil))
}

func processMessage(subscribers *cmap.ConcurrentMap[string, []*melody.Session], s *melody.Session, msg Message, rawMsg []byte, idLogString string) {
	existingTopic, topicWasSet := s.Get("topic")

	switch msg.Type {
	case "ping":
		err := s.Write(PONG_MESSAGE)
		if err != nil {
			log.Println("write failed:", err, idLogString)
			return
		}
	case "subscribe":
		if msg.Topics == nil || len(msg.Topics) != 1 {
			log.Println("received invalid subscribe message", msg, idLogString)
			return
		}
		if topicWasSet {
			if existingTopic.(string) == msg.Topics[0] {
				log.Println("received duplicate subscribe message", msg, idLogString)
				return
			} else {
				log.Println("attempted to subscribe to multiple topics", msg, existingTopic, idLogString)
				return
			}
		}
		s.Set("topic", msg.Topics[0])
		addSubscriber(subscribers, msg.Topics[0], s, idLogString)
	case "unsubscribe":
		s.UnSet("topic")
		removeSubscriber(subscribers, msg.Topics[0], s, idLogString)
	case "publish":
		if msg.Topic == "" || msg.Data == nil {
			log.Println("received invalid publish message", msg, idLogString)
			return
		}
		if msg.Topic != existingTopic {
			log.Println("received publish message for non-subscribed topic", msg, idLogString)
			return
		}

		peers, exists := subscribers.Get(msg.Topic)
		if !exists {
			log.Println("received publish message for non-existent topic", msg, idLogString)
			return
		}
		for _, peer := range peers {
			if peer == s {
				continue
			}

			err := peer.Write(rawMsg)
			if err != nil {
				log.Println("write to peer failed:", err, idLogString)
			}
		}
	}
}

func removeSubscriber(subscribers *cmap.ConcurrentMap[string, []*melody.Session], topic string, conn *melody.Session, logSuffix string) {
	subscribers.Upsert(topic, []*melody.Session{}, func(exists bool, valueInMap []*melody.Session, newValue []*melody.Session) []*melody.Session {
		if exists {
			for i, subscriber := range valueInMap {
				if subscriber == conn {
					return append(valueInMap[:i], valueInMap[i+1:]...)
				}
			}
		}
		return valueInMap
	})

	log.Println("removed subscriber from topic", topic, logSuffix)

	// Remove topic if no subscribers
	removedTopic := subscribers.RemoveCb(topic, func(_ string, valueInMap []*melody.Session, exists bool) bool {
		return exists && len(valueInMap) == 0
	})

	if removedTopic {
		log.Println("cleaned up topic", topic, logSuffix)
		log.Println("topic count:", subscribers.Count())
	}
}

func addSubscriber(subscribers *cmap.ConcurrentMap[string, []*melody.Session], topic string, conn *melody.Session, logSuffix string) {
	subscribers.Upsert(topic, []*melody.Session{conn}, func(exists bool, valueInMap []*melody.Session, newValue []*melody.Session) []*melody.Session {
		if exists {
			// Avoid duplicate subscribers
			for _, subscriber := range valueInMap {
				if subscriber == conn {
					return valueInMap
				}
			}

			return append(valueInMap, conn)
		} else {
			return newValue
		}
	})
	log.Println("added subscriber to topic", topic, logSuffix)
}

func validateOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		log.Println("received empty origin")
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
}
