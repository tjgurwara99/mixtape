package player

import (
	"io"
	"net/http"
	"strings"

	"github.com/tjgurwara99/mixtape"
)

type Mode int

const (
	Record Mode = iota
	Replay
	PassThrough
)

type Player struct {
	cassette  *mixtape.Cassette
	mode      Mode
	transport http.RoundTripper
}

func New(cassette *mixtape.Cassette, mode Mode, transport http.RoundTripper) *Player {
	return &Player{
		cassette:  cassette,
		mode:      mode,
		transport: transport,
	}
}

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrTransportNotSet Error = "transport not set"
)

func (r *Player) RoundTrip(req *http.Request) (*http.Response, error) {
	if r.mode == PassThrough {
		if r.transport == nil {
			return nil, ErrTransportNotSet
		}
		return r.transport.RoundTrip(req)
	}
	recording, err := r.cassette.FindSong(req)
	if err == mixtape.ErrSongNotFound {
		err = nil
	}
	if err != nil && r.mode == Record {
		return nil, err
	}
	if recording != nil {
		return recording.HTTPResponse()
	}
	if r.mode == Replay {
		return nil, mixtape.ErrSongNotFound
	}
	var body []byte
	if req.Body != nil {
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
	}
	req.Body = io.NopCloser(strings.NewReader(string(body)))
	resp, err := r.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(strings.NewReader(string(body)))
	recording, err = mixtape.NewSong(req, resp)
	if err != nil {
		return nil, err
	}
	r.cassette.AddSong(recording)
	return recording.HTTPResponse()
}
