package proxy

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"github.com/andybalholm/brotli"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
)

var tunnelEstablishedResponseLine = []byte("HTTP/1.1 200 Connection established\r\n\r\n")

const AppName = "GO-ANYPROXY"

const maxChunkSize = 32 * 1024 * 1024

type Proxy struct {
	filter ScriptFilter
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleHTTPS(w, r)
	} else {
		p.handleHTTP(w, r)
	}
}

func (p *Proxy) handleHTTPS(w http.ResponseWriter, r *http.Request) {
	conf, err := generateTLSByHost(r.URL.Host)
	if err != nil {
		log.Println("[warn] ", err)
		return
	}
	httpConn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		return
	}
	_, err = httpConn.Write(tunnelEstablishedResponseLine)
	if err != nil {
		log.Println(err)
	}

	conn := tls.Server(httpConn, conf)
	if err := conn.Handshake(); err != nil {
		log.Println("[warn] failed to handshake", err)
		return
	}
	request, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		return
	}
	request.RequestURI = ""
	request.URL.Host = r.URL.Host
	request.URL.Scheme = "https"
	p.handleRequest(conn, request)
}

func (p *Proxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	httpConn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		return
	}
	p.handleRequest(httpConn, r)
}

func (p *Proxy) handleRequest(conn net.Conn, r *http.Request) {
	var filterRequest *Request
	var filterResponse *Response
	var skip bool
	newReq := copyRequest(r)
	if p.filter != nil {
		filterRequest = new(Request)
		filterRequest.HttpRequest = r
		filterRequest.URL = r.URL.String()
		filterRequest.Method = r.Method
		filterRequest.Header = make(map[string]string)
		for k := range r.Header {
			filterRequest.Header[k] = newReq.Header.Get(k)
		}
		if r.Body != nil {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return
			}
			filterRequest.Body = string(body)
		}
		skip = p.filter.FilterRequest(filterRequest)
		if len(filterRequest.Body) != 0 {
			newReq.Body = ioutil.NopCloser(bytes.NewBuffer([]byte(filterRequest.Body)))
		}
	}
	applyRequestContentLength(newReq)
	response, err := http.DefaultTransport.RoundTrip(newReq)
	if err != nil {
		return
	}
	// decode.
	isStreamMode, err := decodeResponse(response)
	if err != nil {
		return
	}
	if isStreamMode {
		response.Write(conn)
		response.Body.Close()
		return
	}
	if p.filter != nil && !skip {
		filterResponse = new(Response)
		filterResponse.StatusCode = response.StatusCode
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return
		}
		filterResponse.Body = string(body)
		filterResponse.Header = make(map[string]string)
		for k := range response.Header {
			filterResponse.Header[k] = response.Header.Get(k)
			response.Header.Del(k)
		}
		p.filter.FilterResponse(filterRequest, filterResponse)
		if len(filterResponse.Header) != 0 {
			for k, v := range filterResponse.Header {
				response.Header.Add(k, v)
			}
		}
		if len(filterResponse.Body) != 0 {
			response.Body = ioutil.NopCloser(bytes.NewBuffer([]byte(filterResponse.Body)))
		}
		response.StatusCode = filterResponse.StatusCode
		response.ContentLength = int64(len(filterResponse.Body))
	}
	if r.URL.String() == "https://unpkg.zhimg.com:443/@cfe/sentry-script@0.0.10/dist/init.js" {
		fmt.Println(1)
	}
	response.Header.Add("PROXY-URL", r.URL.String())
	response.Header.Set("Connection", "Close")
	response.Header.Del("Content-Encoding")

	err = response.Write(conn)
	if err != nil {
		log.Println(r.URL.String(), err)
	}
	response.Body.Close()
}

func copyRequest(r *http.Request) *http.Request {
	req := new(http.Request)
	*req = *r
	req.RequestURI = ""
	req.URL.Host = r.URL.Host
	req.URL.Scheme = r.URL.Scheme

	return req
}

type streamReader struct {
	buf          *bytes.Buffer
	responseBody io.ReadCloser
}

func (sr *streamReader) Read(p []byte) (int, error) {
	if sr.buf.Len() > 0 {
		if n, err := sr.buf.Read(p); err != nil {
			if err == io.EOF {
				b := p[n:]
				nn, err := sr.responseBody.Read(b)
				return n + nn, err
			}
		}
	}

	return sr.responseBody.Read(p)
}

func decodeResponse(response *http.Response) (bool, error) {
	var buf = bytes.NewBuffer(nil)
	for {
		b := make([]byte, 2048)
		n, err := response.Body.Read(b)
		if err != nil {
			if err == io.EOF {
				buf.Write(b[:n])
				break
			}
			return false, err
		}
		buf.Write(b[:n])
		// stream mode.
		if buf.Len() > maxChunkSize {
			response.Body = ioutil.NopCloser(&streamReader{buf: buf, responseBody: response.Body})
			return true, nil
		}
	}
	var reader io.Reader
	var contentLength int64
	encoding := response.Header.Get("Content-Encoding")
	switch encoding {
	case "br":
		reader = brotli.NewReader(buf)
	case "gzip":
		r, err := gzip.NewReader(buf)
		if err != nil {
			return false, err
		}
		reader = r
	case "deflate":
		reader = flate.NewReader(buf)
	default:
		reader = buf
	}

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return false, err
	}
	contentLength = int64(len(data))
	response.Body = ioutil.NopCloser(bytes.NewBuffer(data))
	response.Header.Del("Content-Encoding")
	response.Header.Del("Content-Length")
	response.Header.Del("Transfer-Encoding")
	response.ContentLength = contentLength
	return false, nil
}

func applyRequestContentLength(r *http.Request) {
	if header := r.Header.Get("Transfer-Encoding"); header == "chunked" {
		r.Header.Del("Content-Length")
		r.ContentLength = -1
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	r.ContentLength = int64(len(body))
	return
}

func New(js string, cert, key string) *Proxy {
	filter := newJavascriptFilter(js)
	pxy := new(Proxy)
	pxy.filter = filter

	if _, _, err := loadCa(cert, key); err != nil {
		panic(err)
	}

	return pxy
}

func (p *Proxy) Run(addr string) error {
	return http.ListenAndServe(addr, p)
}
