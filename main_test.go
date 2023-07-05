package main

import (
	"net/http"
	"testing"
)

// Path: main_test.go

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
