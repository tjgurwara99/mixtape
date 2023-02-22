package mixtape

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

type Cassette struct {
	Name               string                             `json:"-"`
	FilePath           string                             `json:"-"`
	Songs              []*Song                            `json:"songs"`
	Comparer           func(*http.Request, *Request) bool `json:"-"`
	nextRecordingIndex int
	sync.RWMutex
}

func (c *Cassette) Equal(other *Cassette) bool {
	if len(c.Songs) != len(other.Songs) {
		return false
	}
	for i, song := range c.Songs {
		if !song.Equal(other.Songs[i]) {
			return false
		}
	}
	return true
}

func (c *Cassette) Save() error {
	c.Lock()
	defer c.Unlock()

	dir := filepath.Dir(c.FilePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	nextId := 0
	songs := make([]*Song, len(c.Songs))
	for _, song := range c.Songs {
		song.ID = nextId
		songs[nextId] = song
		nextId++
	}
	c.Songs = songs

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	_, err = os.Stat(c.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			f, err := os.Create(c.FilePath)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = f.Write(data)
			return err
		}
	}

	oldCassette, cErr := Load(c.Name)
	if cErr != nil {
		return cErr
	}
	if oldCassette.Equal(c) {
		return nil
	}
	return os.WriteFile(c.FilePath, data, 0644)
}

func DefaultCompareFunc(r *http.Request, recording *Request) bool {
	return r.Method == recording.Method && r.URL.String() == recording.URL
}

func CompareFuncWithBody(r *http.Request, recording *Request) bool {
	if !DefaultCompareFunc(r, recording) {
		return false
	}
	if r.Body == nil && recording.Body == "" {
		return true
	}
	if r.Body == nil {
		return false
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	return string(body) == recording.Body
}

func New(name string) *Cassette {
	return &Cassette{
		Name:               name,
		FilePath:           name + ".json",
		nextRecordingIndex: 0,
		Comparer:           DefaultCompareFunc,
	}
}

func Load(name string) (*Cassette, error) {
	c := New(name)
	data, err := os.ReadFile(c.FilePath)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, c)
	if err != nil {
		return nil, err
	}
	c.nextRecordingIndex = len(c.Songs)
	return c, err
}

func (c *Cassette) AddSong(r *Song) {
	c.Lock()
	defer c.Unlock()
	r.ID = c.nextRecordingIndex
	c.Songs = append(c.Songs, r)
	c.nextRecordingIndex++
}

type Song struct {
	ID       int       `json:"id"`
	Request  *Request  `json:"request"`
	Response *Response `json:"response"`
}

func (s *Song) Equal(other *Song) bool {
	if s.ID != other.ID {
		return false
	}
	if !s.Request.Equal(other.Request) {
		return false
	}
	if !s.Response.Equal(other.Response) {
		return false
	}
	return true
}

func NewSong(req *http.Request, resp *http.Response) (*Song, error) {
	r, err := fromHTTPRequestToRequest(req)
	if err != nil {
		return nil, err
	}

	rr, err := fromHTTPResponseToResponse(resp)
	if err != nil {
		return nil, err
	}

	return &Song{
		Request:  r,
		Response: rr,
	}, nil
}

type Request struct {
	Method           string
	URL              string
	Proto            string // "HTTP/1.0"
	ProtoMajor       int    // 1
	ProtoMinor       int    // 0
	Header           http.Header
	Body             string
	ContentLength    int64
	TransferEncoding []string
	Close            bool
	Host             string
	Form             url.Values
	PostForm         url.Values
	MultipartForm    *multipart.Form
	Trailer          http.Header
	RemoteAddr       string
	RequestURI       string
	TLS              *tls.ConnectionState
}

func (r *Request) Equal(other *Request) bool {
	if r.Method != other.Method {
		return false
	}
	if r.URL != other.URL {
		return false
	}
	if r.Proto != other.Proto {
		return false
	}
	if r.ProtoMajor != other.ProtoMajor {
		return false
	}
	if r.ProtoMinor != other.ProtoMinor {
		return false
	}
	if !reflect.DeepEqual(r.Header, other.Header) {
		return false
	}
	if r.Body != other.Body {
		return false
	}
	if r.ContentLength != other.ContentLength {
		return false
	}
	if !reflect.DeepEqual(r.TransferEncoding, other.TransferEncoding) {
		return false
	}
	if r.Close != other.Close {
		return false
	}
	if r.Host != other.Host {
		return false
	}
	if !reflect.DeepEqual(r.Form, other.Form) {
		return false
	}
	if !reflect.DeepEqual(r.PostForm, other.PostForm) {
		return false
	}
	if !reflect.DeepEqual(r.MultipartForm, other.MultipartForm) {
		return false
	}
	if !reflect.DeepEqual(r.Trailer, other.Trailer) {
		return false
	}
	if r.RemoteAddr != other.RemoteAddr {
		return false
	}
	if r.RequestURI != other.RequestURI {
		return false
	}
	if !reflect.DeepEqual(r.TLS, other.TLS) {
		return false
	}
	return true
}

// Response represents the response from a server which can be saved in json/yaml format.
// Majority of the properties in this struct could be classed as members of http.Response object
// however, there is a subtle distinction between http.Response and this Response. Mostly
// related to the intrinsic differences between certain fields such as
// `Request.GetBody` field and `Body` field.
type Response struct {
	Status           string // e.g. "200 OK"
	StatusCode       int    // e.g. 200
	Proto            string // e.g. "HTTP/1.0"
	ProtoMajor       int    // e.g. 1
	ProtoMinor       int    // e.g. 0
	Header           http.Header
	Body             string
	ContentLength    int64
	TransferEncoding []string
	Close            bool
	Trailer          http.Header
}

func (r *Response) Equal(other *Response) bool {
	if r.Status != other.Status {
		return false
	}
	if r.StatusCode != other.StatusCode {
		return false
	}
	if r.Proto != other.Proto {
		return false
	}
	if r.ProtoMajor != other.ProtoMajor {
		return false
	}
	if r.ProtoMinor != other.ProtoMinor {
		return false
	}
	for k, v := range r.Header {
		if k == "Date" {
			continue
		}
		if !reflect.DeepEqual(v, other.Header[k]) {
			return false
		}
	}
	if r.Body != other.Body {
		return false
	}
	if r.ContentLength != other.ContentLength {
		return false
	}
	if !reflect.DeepEqual(r.TransferEncoding, other.TransferEncoding) {
		return false
	}
	if r.Close != other.Close {
		return false
	}
	if !reflect.DeepEqual(r.Trailer, other.Trailer) {
		return false
	}
	return true
}

func (r *Song) HTTPRequest() (*http.Request, error) {
	return r.Request.toHTTPRequest()
}

func (r *Song) HTTPResponse() (*http.Response, error) {
	return r.Response.toHTTPResponse()
}

// toHTTPRequest converts a cassette.Request to a http.Request
func (r *Request) toHTTPRequest() (*http.Request, error) {
	url, err := url.Parse(r.URL)
	if err != nil {
		return nil, err
	}
	return &http.Request{
		Method:           r.Method,
		URL:              url,
		Proto:            r.Proto,
		ProtoMajor:       r.ProtoMajor,
		ProtoMinor:       r.ProtoMinor,
		Header:           r.Header,
		Body:             io.NopCloser(strings.NewReader(r.Body)),
		ContentLength:    r.ContentLength,
		TransferEncoding: r.TransferEncoding,
		Close:            r.Close,
		Host:             r.Host,
		Form:             r.Form,
		PostForm:         r.PostForm,
		MultipartForm:    r.MultipartForm,
		Trailer:          r.Trailer,
		RemoteAddr:       r.RemoteAddr,
	}, nil
}

func fromHTTPRequestToRequest(r *http.Request) (*Request, error) {
	var body string
	if r.Body != nil {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		body = string(b)
	}
	return &Request{
		Method:           r.Method,
		URL:              r.URL.String(),
		Proto:            r.Proto,
		ProtoMajor:       r.ProtoMajor,
		ProtoMinor:       r.ProtoMinor,
		Header:           r.Header,
		Body:             body,
		ContentLength:    r.ContentLength,
		TransferEncoding: r.TransferEncoding,
		Close:            r.Close,
		Trailer:          r.Trailer,
		Host:             r.Host,
		Form:             r.Form,
		PostForm:         r.PostForm,
		MultipartForm:    r.MultipartForm,
		RemoteAddr:       r.RemoteAddr,
		RequestURI:       r.RequestURI,
		TLS:              r.TLS,
	}, nil
}

// toHTTPResponse converts a cassette.Response to a http.Response
func (r *Response) toHTTPResponse() (*http.Response, error) {
	return &http.Response{
		Status:           r.Status,
		StatusCode:       r.StatusCode,
		Proto:            r.Proto,
		ProtoMajor:       r.ProtoMajor,
		ProtoMinor:       r.ProtoMinor,
		Header:           r.Header,
		Body:             io.NopCloser(strings.NewReader(r.Body)),
		ContentLength:    r.ContentLength,
		TransferEncoding: r.TransferEncoding,
		Close:            r.Close,
		Trailer:          r.Trailer,
	}, nil
}

func fromHTTPResponseToResponse(r *http.Response) (*Response, error) {
	var body string
	if r.Body != nil {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		body = string(b)
	}
	return &Response{
		Status:           r.Status,
		StatusCode:       r.StatusCode,
		Proto:            r.Proto,
		ProtoMajor:       r.ProtoMajor,
		ProtoMinor:       r.ProtoMinor,
		Header:           r.Header,
		Body:             body,
		ContentLength:    r.ContentLength,
		TransferEncoding: r.TransferEncoding,
		Close:            r.Close,
		Trailer:          r.Trailer,
	}, nil
}

type Error string

func (e Error) Error() string {
	return string(e)
}

const ErrSongNotFound Error = "not found"

func (c *Cassette) FindSong(r *http.Request) (*Song, error) {
	c.Lock()
	defer c.Unlock()
	for _, song := range c.Songs {
		if c.Comparer(r, song.Request) {
			return song, nil
		}
	}
	return nil, ErrSongNotFound
}
