// Copyright 2013 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

const (
	ok         = "ok\n"
	notAllowed = "Method not allowed\n"
)

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(ok))
})

func newRequest(method, url string) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	return req
}

func TestMethodHandler(t *testing.T) {
	tests := []struct {
		req     *http.Request
		handler http.Handler
		code    int
		allow   string // Contents of the Allow header
		body    string
	}{
		// No handlers
		{newRequest("GET", "/foo"), MethodHandler{}, http.StatusMethodNotAllowed, "", notAllowed},
		{newRequest("OPTIONS", "/foo"), MethodHandler{}, http.StatusOK, "", ""},

		// A single handler
		{newRequest("GET", "/foo"), MethodHandler{"GET": okHandler}, http.StatusOK, "", ok},
		{newRequest("POST", "/foo"), MethodHandler{"GET": okHandler}, http.StatusMethodNotAllowed, "GET", notAllowed},

		// Multiple handlers
		{newRequest("GET", "/foo"), MethodHandler{"GET": okHandler, "POST": okHandler}, http.StatusOK, "", ok},
		{newRequest("POST", "/foo"), MethodHandler{"GET": okHandler, "POST": okHandler}, http.StatusOK, "", ok},
		{newRequest("DELETE", "/foo"), MethodHandler{"GET": okHandler, "POST": okHandler}, http.StatusMethodNotAllowed, "GET, POST", notAllowed},
		{newRequest("OPTIONS", "/foo"), MethodHandler{"GET": okHandler, "POST": okHandler}, http.StatusOK, "GET, POST", ""},

		// Override OPTIONS
		{newRequest("OPTIONS", "/foo"), MethodHandler{"OPTIONS": okHandler}, http.StatusOK, "", ok},
	}

	for i, test := range tests {
		rec := httptest.NewRecorder()
		test.handler.ServeHTTP(rec, test.req)
		if rec.Code != test.code {
			t.Fatalf("%d: wrong code, got %d want %d", i, rec.Code, test.code)
		}
		if allow := rec.HeaderMap.Get("Allow"); allow != test.allow {
			t.Fatalf("%d: wrong Allow, got %s want %s", i, allow, test.allow)
		}
		if body := rec.Body.String(); body != test.body {
			t.Fatalf("%d: wrong body, got %q want %q", i, body, test.body)
		}
	}
}

func TestWriteLog(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		panic(err)
	}
	ts := time.Date(1983, 05, 26, 3, 30, 45, 0, loc)

	// A typical request with an OK response
	req := newRequest("GET", "http://example.com")
	req.RemoteAddr = "192.168.100.5"

	buf := new(bytes.Buffer)
	writeLog(buf, req, ts, http.StatusOK, 100)
	log := buf.String()

	expected := "192.168.100.5 - - [26/May/1983:03:30:45 +0200] \"GET  HTTP/1.1\" 200 100\n"
	if log != expected {
		t.Fatalf("wrong log, got %q want %q", log, expected)
	}

	// Request with an unauthorized user
	req.URL.User = url.User("kamil")

	buf.Reset()
	writeLog(buf, req, ts, http.StatusUnauthorized, 500)
	log = buf.String()

	expected = "192.168.100.5 - kamil [26/May/1983:03:30:45 +0200] \"GET  HTTP/1.1\" 401 500\n"
	if log != expected {
		t.Fatalf("wrong log, got %q want %q", log, expected)
	}
}
