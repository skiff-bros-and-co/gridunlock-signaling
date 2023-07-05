package main

import (
	"net/http"
	"testing"

	"github.com/olahol/melody"
	cmap "github.com/orcaman/concurrent-map/v2"
)

func TestAddSubscriber(t *testing.T) {
	subscribers := cmap.New[[]*melody.Session]()
	s := &melody.Session{}
	addSubscriber(&subscribers, "test", s, "test-client")

	topic, _ := subscribers.Get("test")
	if len(topic) != 1 {
		t.Error("Subscriber was not added")
	}
	if topic[0] != s {
		t.Error("Incorrect subscriber was added")
	}
}

func TestAddSubscriber_Multiple(t *testing.T) {
	subscribers := cmap.New[[]*melody.Session]()
	s := &melody.Session{}

	addSubscriber(&subscribers, "test", s, "test-client")
	addSubscriber(&subscribers, "test", s, "test-client")

	topic, _ := subscribers.Get("test")
	if len(topic) != 1 {
		t.Error("Subscriber was not added")
	}
	if topic[0] != s {
		t.Error("Incorrect subscriber was added")
	}
}

func TestAddSubscriber_Duplicates(t *testing.T) {
	subscribers := cmap.New[[]*melody.Session]()
	s := &melody.Session{}

	addSubscriber(&subscribers, "test", s, "test-client")
	addSubscriber(&subscribers, "test", s, "test-client")

	topic, _ := subscribers.Get("test")
	if len(topic) != 1 {
		t.Error("Subscriber was errently double added")
	}
}

func TestRemoveSubscriber_Base(t *testing.T) {
	subscribers := cmap.New[[]*melody.Session]()
	s := &melody.Session{}

	addSubscriber(&subscribers, "test", s, "test-client")
	removeSubscriber(&subscribers, "test", s, "test-client")

	if len(subscribers.Keys()) > 0 {
		t.Error("Subscriber was not removed")
	}
}

func TestRemoveSubscriber_Multiple(t *testing.T) {
	subscribers := cmap.New[[]*melody.Session]()
	s1 := &melody.Session{}
	s2 := &melody.Session{}
	s3 := &melody.Session{}

	addSubscriber(&subscribers, "test", s1, "test-client")
	addSubscriber(&subscribers, "test", s2, "test-client")
	addSubscriber(&subscribers, "test", s3, "test-client")
	removeSubscriber(&subscribers, "test", s2, "test-client")

	topic, _ := subscribers.Get("test")
	if len(topic) != 2 {
		t.Error("Unexpected number of subscribers", topic)
	}
	if topic[0] != s1 {
		t.Error("S1 is incorrect")
	}
	if topic[1] != s3 {
		t.Error("S3 is incorrect")
	}

	removeSubscriber(&subscribers, "test", s1, "test-client")
	removeSubscriber(&subscribers, "test", s3, "test-client")

	_, stillHasSubscribers := subscribers.Get("test")
	if stillHasSubscribers {
		t.Error("Somehow some subscribers remain")
	}
}

// Test origin validation
func TestValidateOrigin(t *testing.T) {
	if !validateOrigin(&http.Request{Header: http.Header{"Origin": []string{"http://localhost:5173"}}}) {
		t.Error("local dev origin was rejected")
	}

	if !validateOrigin(&http.Request{Header: http.Header{"Origin": []string{"https://gridunlockapp.com"}}}) {
		t.Error("Valid origin was rejected")
	}

	if !validateOrigin(&http.Request{Header: http.Header{"Origin": []string{"https://test.gridunlock-org.pages.dev"}}}) {
		t.Error("Valid origin was rejected")
	}

	if validateOrigin(&http.Request{Header: http.Header{"Origin": []string{"http://localhost:8081"}}}) {
		t.Error("Invalid origin was accepted")
	}

	if validateOrigin(&http.Request{Header: http.Header{"Origin": []string{"http://gridunlockapp.com"}}}) {
		t.Error("non-https origin was accepted")
	}

	if validateOrigin(&http.Request{Header: http.Header{"Origin": []string{"https://google.com"}}}) {
		t.Error("Invalid origin was accepted")
	}

	if validateOrigin(&http.Request{Header: http.Header{"Origin": []string{""}}}) {
		t.Error("Empty origin was accepted")
	}

	if validateOrigin(&http.Request{Header: http.Header{}}) {
		t.Error("Missing origin was accepted")
	}
}
