package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestJSONRetriesServerErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < attempts {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	var got struct{ OK bool }
	c := Client{HTTP: srv.Client()}
	if err := c.JSON(context.Background(), srv.URL, &got); err != nil {
		t.Fatal(err)
	}
	if !got.OK || calls.Load() != attempts {
		t.Errorf("got %+v after %d calls", got, calls.Load())
	}
}

func TestJSONDoesNotRetryClientErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	var got struct{}
	if err := (Client{HTTP: srv.Client()}).JSON(context.Background(), srv.URL, &got); err == nil {
		t.Fatal("JSON succeeded, want error")
	}
	if calls.Load() != 1 {
		t.Errorf("made %d calls, want 1", calls.Load())
	}
}

func TestJSONDoesNotRetryMalformedBody(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	var got struct{}
	if err := (Client{HTTP: srv.Client()}).JSON(context.Background(), srv.URL, &got); err == nil {
		t.Fatal("JSON succeeded, want error")
	}
	if calls.Load() != 1 {
		t.Errorf("made %d calls, want 1", calls.Load())
	}
}
