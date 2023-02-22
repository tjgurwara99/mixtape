package player_test

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/tjgurwara99/mixtape"
	"github.com/tjgurwara99/mixtape/player"
)

func ExampleRecord() {
	filePath := filepath.Join("testdata", "example_record") // this must be without the .json extension
	cassette := mixtape.New(filePath)
	player := player.New(cassette, player.Record, http.DefaultTransport)
	defer cassette.Save()
	c := &http.Client{
		Transport: player,
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	l, err := net.Listen("tcp", "127.0.0.1:52120")
	if err != nil {
		panic(err)
	}
	server.Listener = l
	server.Start()
	defer server.Close()
	resp, err := c.Get(server.URL)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
	// Output:
}

func ExampleReplay() {
	filePath := filepath.Join("testdata", "example_replay") // this must be without the .json extension
	cassette, err := mixtape.Load(filePath)
	player := player.New(cassette, player.Replay, http.DefaultTransport)
	c := &http.Client{
		Transport: player,
	}
	resp, err := c.Get("http://example.com/something-here")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
	// Output: Something here
}

func TestRecord(t *testing.T) {
	filePath := filepath.Join("testdata", "test_record") // this must be without the .json extension
	cassette := mixtape.New(filePath)
	player := player.New(cassette, player.Record, http.DefaultTransport)
	defer cassette.Save()
	c := &http.Client{
		Transport: player,
	}
	testServerURL := "127.0.0.1:52119"
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	l, err := net.Listen("tcp", testServerURL)
	if err != nil {
		t.Fatal(err)
	}
	server.Listener = l
	server.Start()
	defer server.Close()
	resp, err := c.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
}

func TestRecordWithBody(t *testing.T) {
	filePath := filepath.Join("testdata", "test_record_with_body") // this must be without the .json extension
	cassette := mixtape.New(filePath)
	cassette.Comparer = mixtape.CompareFuncWithBody // check the body as well
	songPlayer := player.New(cassette, player.Record, http.DefaultTransport)
	defer cassette.Save()
	c := &http.Client{
		Transport: songPlayer,
	}
	data := bytes.Buffer{}
	data.WriteString("Hello, World!")
	testServerURL := "127.0.0.1:52118"
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	l, _ := net.Listen("tcp", testServerURL)
	server.Listener = l
	server.Start()
	defer server.Close()
	req, err := http.NewRequest(http.MethodPost, server.URL, &data)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
}

func TestReplayWithBody(t *testing.T) {
	filePath := filepath.Join("testdata", "test_replay_with_body") // this must be without the .json extension
	cassette, err := mixtape.Load(filePath)
	if err != nil {
		t.Fatal(err)
	}
	cassette.Comparer = mixtape.CompareFuncWithBody // check the body as well
	player := player.New(cassette, player.Record, http.DefaultTransport)
	c := &http.Client{
		Transport: player,
	}
	data := bytes.Buffer{}
	data.WriteString("Hello, World!")
	req, err := http.NewRequest(http.MethodPost, "http://example.com/something-here", &data)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(respBody) != "Something here" {
		t.Errorf("expected: %s, got: %s", "Something here", string(respBody))
	}
}

func TestReplay(t *testing.T) {
	filePath := filepath.Join("testdata", "test_replay") // this must be without the .json extension
	cassette, err := mixtape.Load(filePath)
	if err != nil {
		t.Fatal(err)
	}
	player := player.New(cassette, player.Replay, http.DefaultTransport)
	c := &http.Client{
		Transport: player,
	}
	resp, err := c.Get("http://example.com/something-here")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "Something here" {
		t.Errorf("expected: %s, got: %s", "Something here", string(data))
	}
}
