package main

import (
	"encoding/json"
	"fmt"
	"github.com/robertkrimen/otto"
	"log"
	"net/http"
	"net/url"
)

const script = `
(function () {
    return {
        request: {
            // urlMatch
            urlMatch: function (url) {
                return url.lastIndexOf("baidu.com") !== -1
            },
            methodMatch: function (method) {
                return method === "GET" || method === "POST"
            },
            beforeRequest: function (request) {
                return request
            },
            afterRequest: function (request) {
                return
            }
        },
        response: {
            statusCode: function (code) {
                return code == 200 || code > 500
            },
            contentTypeMatch: function (contentType) {
                return /(text|application)\/(json|html)/.test(contentType)
            },
            onResponse: function (request, response) {
                return response
            }
        }
    }
})()
`

func main() {
	vm := otto.New()
	val, err := vm.Run(script)
	if err != nil {
		log.Println(err)
		return
	}

	obj := val.Object()
	request, err := obj.Get("request")
	if err != nil {
		log.Println(err)
		return
	}

	// isMatch
	{
		urlMatch, err := request.Object().Get("urlMatch")
		if err != nil {
			log.Println(err)
			return
		}
		u, _ := url.Parse("https://www.baidu.com")
		isMatch, err := urlMatch.Call(val, u.String())
		if err != nil {
			log.Println(err)
			return
		}
		if isMatch.IsBoolean() {
			ok, _ := isMatch.ToBoolean()
			fmt.Println(ok)
		}
	}

	// beforeRequest
	{
		beforeFunc, err := request.Object().Get("beforeRequest")
		if err != nil {
			log.Println(err)
			return
		}
		req := new(Request)
		req.Header = make(map[string]string)
		req.Header["Foo"] = "bar"
		req.Body = "hello world"
		result, err := beforeFunc.Call(val, req)
		if err != nil {
			log.Println(err)
			return
		}

		rr, err := valToRequest(result)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Println(rr)

	}
}

type Header struct {
	http.Header
}

func (h Header) MarshalJSON() ([]byte, error) {
	var data = make(map[string]string)
	for k := range h.Header {
		data[k] = h.Get(k)
	}
	return json.Marshal(data)
}

func (h Header) UnmarshalJSON(b []byte) error {
	var data = make(map[string]string)
	err := json.Unmarshal(b, &data)
	if err != nil {
		return err
	}
	if h.Header == nil {
		h.Header = make(http.Header)
	}
	for k, v := range data {
		h.Header.Add(k, v)
	}
	return nil
}

type Request struct {
	Header map[string]string `json:"header"`
	Body   string            `json:"body"`
}

func valToRequest(value otto.Value) (Request, error) {
	req := Request{}
	req.Header = make(map[string]string)
	header, err := value.Object().Get("Header")
	if err != nil {
		return Request{}, err
	}
	if header.IsObject() {
		keys := header.Object().Keys()
		for _, k := range keys {
			val, err := header.Object().Get(k)
			if err != nil {
				return Request{}, err
			}
			if val.IsString() {
				req.Header[k] = val.String()
			}
		}
	}
	body, err := value.Object().Get("Body")
	if err != nil {
		return Request{}, err
	}
	if body.IsString() {
		req.Body = body.String()
	}
	return req, nil
}
