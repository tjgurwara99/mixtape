package player

import (
	"net/http"

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
	if err != nil {
		return nil, err
	}
	if recording == nil {
		if r.mode == Replay {
			return nil, mixtape.ErrSongNotFound
		}
		resp, err := r.transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		recording, err = mixtape.NewSong(req, resp)
		if err != nil {
			return nil, err
		}
		r.cassette.AddSong(recording)
		return resp, nil
	}
	return recording.HTTPResponse()
}
