package player_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
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
	err = os.Remove(cassette.FilePath)
	if err != nil {
		panic(err)
	}
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	resp, err := c.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	err = os.Remove(cassette.FilePath)
	if err != nil {
		t.Fatal(err)
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
