package proxy

import (
	"bytes"
	"io"
	"net/http"
)

type ReadCloseReset interface {
	io.ReadCloser
	Reset() error
	ResetBytes([]byte) error
}

func readResponse(response *http.Response) (ReadCloseReset, error) {
	transferEncoding := response.Header.Get("Transfer-Encoding")
	var body []byte
	// chunked.
	if transferEncoding != "" {
		body = make([]byte, maxChunkSize)
	} else {
		body = make([]byte, response.ContentLength)
	}

	_, err := io.ReadFull(response.Body, body)
	if err != nil {
		return nil, err
	}

	rcr := &readCloseReset{b: body}
	rcr.buf = bytes.NewBuffer(body)
	rcr.responseBody = response.Body

	return rcr, nil
}

type readCloseReset struct {
	b   []byte
	buf *bytes.Buffer

	responseBody io.ReadCloser
}

func newReadCloseReset(b []byte) ReadCloseReset {

	return rcr
}

func (readCloseReset) Close() error {
	return nil
}

func (rcr *readCloseReset) Reset() error {
	rcr.buf.Reset()
	rcr.buf.Write(rcr.b)
	return nil
}

func (rcr *readCloseReset) ResetBytes(b []byte) error {
	rcr.b = b
	return rcr.Reset()
}

func (rcr *readCloseReset) Read(b []byte) (int, error) {
	return rcr.buf.Read(b)
}
