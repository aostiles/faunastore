package faunastore

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	f "github.com/fauna/faunadb-go/v3/faunadb"
)

func TestSave(t *testing.T) {
	client := f.NewFaunaClient(os.Getenv("FAUNA_PASS"))

	store, err := NewFaunaStore(client)
	if err != nil {
		t.Fatal("failed to create fauna store", err)
	}

	req, err := http.NewRequest("GET", "http://www.example.com", nil)
	if err != nil {
		t.Fatal("failed to create request", err)
	}
	w := httptest.NewRecorder()

	session, err := store.New(req, "hello")
	if err != nil {
		t.Fatal("failed to create session", err)
	}

	session.Values["key"] = "value"
	err = session.Save(req, w)
	if err != nil {
		t.Fatal("failed to save: ", err)
	}
}

func TestDelete(t *testing.T) {

	client := f.NewFaunaClient(os.Getenv("FAUNA_PASS"))

	store, err := NewFaunaStore(client)
	if err != nil {
		t.Fatal("failed to create fauna store", err)
	}

	req, err := http.NewRequest("GET", "http://www.example.com", nil)
	if err != nil {
		t.Fatal("failed to create request", err)
	}
	w := httptest.NewRecorder()

	session, err := store.New(req, "hello")
	if err != nil {
		t.Fatal("failed to create session", err)
	}

	session.Values["key"] = "value"
	err = session.Save(req, w)
	if err != nil {
		t.Fatal("failed to save session: ", err)
	}

	session.Options.MaxAge = -1
	err = session.Save(req, w)
	if err != nil {
		t.Fatal("failed to delete session: ", err)
	}
}

func TestLoad(t *testing.T) {

	client := f.NewFaunaClient(os.Getenv("FAUNA_PASS"))

	store, err := NewFaunaStore(client)
	if err != nil {
		t.Fatal("failed to create fauna store", err)
	}

	req, err := http.NewRequest("GET", "http://www.example.com", nil)
	if err != nil {
		t.Fatal("failed to create request", err)
	}
	w := httptest.NewRecorder()

	session, err := store.New(req, "hello")
	if err != nil {
		t.Fatal("failed to create session", err)
	}

	k, _ := generateRandomKey()
	session.Values["key"] = k
	err = session.Save(req, w)
	if err != nil {
		t.Fatal("failed to save session: ", err)
	}
	resp := w.Result()
	req, err = http.NewRequest("GET", "http://www.example.com", nil)

	for _, cookie := range resp.Cookies() {
		req.AddCookie(cookie)
	}

	session, err = store.New(req, "hello")
	if err != nil {
		t.Fatal("failed to create session", err)
	}
	if session.Values["key"] != k {
		t.Fatal("session keys not equal", session.Values["key"], k)
	}
	k2, _ := generateRandomKey()
	session.Values["key"] = k2
	err = session.Save(req, w)
	if err != nil {
		t.Fatal("failed to save session: ", err)
	}
	resp = w.Result()
	req, err = http.NewRequest("GET", "http://www.example.com", nil)

	for _, cookie := range resp.Cookies() {
		req.AddCookie(cookie)
	}

	session, err = store.New(req, "hello")
	if err != nil {
		t.Fatal("failed to create session", err)
	}
	if session.Values["key"] != k2 {
		t.Fatal("session keys not equal", session.Values["key"], k2)
	}
}
