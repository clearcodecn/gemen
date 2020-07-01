package proxy

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/robertkrimen/otto"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Request struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Header      map[string]string `json:"header"`
	Body        string            `json:"body"`
	HttpRequest *http.Request     `json:"-"`
}

type Response struct {
	StatusCode int               `json:"status_code"`
	Header     map[string]string `json:"header"`
	Body       string            `json:"body"`
}

type JavascriptFilter struct {
	script string
}

func (js *JavascriptFilter) FilterRequest(req *Request) bool {
	if js.script == "" {
		return true
	}
	script, err := ioutil.ReadFile(js.script)
	if err != nil {
		log.Println("[warn] failed to read script: ", err.Error())
		return true
	}

	vm := otto.New()
	val, err := vm.Run(script)
	if err != nil {
		log.Println("[warn] failed to compile script: ", err.Error())
		return true
	}

	reqObject, err := val.Object().Get("request")
	if err != nil {
		return true
	}
	if !reqObject.IsObject() {
		return true
	}

	match, err := js.urlMatch(reqObject, req)
	if err != nil {
		log.Println("[warn] failed to execute urlMatch:", err)
		return true
	}
	if !match {
		return true
	}
	match, err = js.methodMatch(reqObject, req)
	if err != nil {
		log.Println("[warn] failed to execute methodMatch:", err)
		return true
	}
	if err := js.beforeRequest(reqObject, req); err != nil {
		log.Println("[warn] failed to execute beforeRequest: ", err)
		return true
	}
	return false
}

func (js *JavascriptFilter) FilterResponse(request *Request, response *Response) {
	if js.script == "" {
		return
	}
	script, err := ioutil.ReadFile(js.script)
	if err != nil {
		log.Println("[warn] failed to read script: ", err.Error())
		return
	}

	vm := otto.New()
	val, err := vm.Run(script)
	if err != nil {
		log.Println("[warn] failed to compile script: ", err.Error())
		return
	}
	respObject, err := val.Object().Get("response")
	if err != nil {
		return
	}
	if !respObject.IsObject() {
		return
	}

	match, err := js.contentTypeMatch(val, request, response)
	if err != nil {
		return
	}
	if !match {
		return
	}
	if err := js.onResponse(respObject, request, response); err != nil {
		log.Println("[warn] failed to filter response")
		return
	}
}

func (js *JavascriptFilter) urlMatch(vm otto.Value, r *Request) (bool, error) {
	urlMatch, err := vm.Object().Get("urlMatch")
	if err != nil {
		return false, err
	}
	if !urlMatch.IsFunction() {
		return true, nil
	}

	url := r.HttpRequest.URL.String()
	isMatch, err := urlMatch.Call(vm, url)
	if err != nil {
		return false, err
	}
	if isMatch.IsBoolean() {
		ok, _ := isMatch.ToBoolean()
		return ok, nil
	}
	return true, nil
}

func (js *JavascriptFilter) methodMatch(vm otto.Value, r *Request) (bool, error) {
	urlMatch, err := vm.Object().Get("urlMatch")
	if err != nil {
		return false, err
	}
	if !urlMatch.IsFunction() {
		return true, nil
	}
	isMatch, err := urlMatch.Call(vm, r.HttpRequest.Method)
	if err != nil {
		return false, err
	}
	if isMatch.IsBoolean() {
		ok, _ := isMatch.ToBoolean()
		return ok, nil
	}
	return true, nil
}

func (js *JavascriptFilter) beforeRequest(vm otto.Value, r *Request) error {
	beforeRequest, err := vm.Object().Get("beforeRequest")
	if err != nil {
		return err
	}
	if !beforeRequest.IsFunction() {
		return nil
	}
	filterRequest, err := beforeRequest.Call(vm, r)
	if err != nil {
		return err
	}
	if !filterRequest.IsObject() {
		return nil
	}
	headers, err := filterRequest.Object().Get("Header")
	if err != nil {
		return err
	}
	if headers.IsObject() {
		keys := headers.Object().Keys()
		for _, k := range keys {
			val, err := headers.Object().Get(k)
			if err != nil {
				return err
			}
			r.Header[k] = val.String()
		}
	}

	body, err := filterRequest.Object().Get("Body")
	if err != nil {
		return err
	}
	if body.IsObject() {
		r.Body = body.String()
	}

	query, err := filterRequest.Object().Get("query")
	if err != nil {
		return err
	}
	if query.IsObject() {
		keys := query.Object().Keys()
		for _, k := range keys {
			val, err := query.Object().Get(k)
			if err != nil {
				return err
			}
			r.HttpRequest.URL.Query().Set(k, val.String())
		}
	}
	return nil
}

func (js *JavascriptFilter) contentTypeMatch(vm otto.Value, r *Request, response *Response) (bool, error) {
	contentTypeMatch, err := vm.Object().Get("contentTypeMatch")
	if err != nil {
		return false, err
	}
	if !contentTypeMatch.IsFunction() {
		return true, nil
	}
	isMatch, err := contentTypeMatch.Call(vm, response.Header["Content-Type"])
	if err != nil {
		return false, err
	}
	if isMatch.IsBoolean() {
		ok, _ := isMatch.ToBoolean()
		return ok, nil
	}
	return true, nil
}

func (js *JavascriptFilter) onResponse(vm otto.Value, request *Request, response *Response) error {
	onResponse, err := vm.Object().Get("onResponse")
	if err != nil {
		return err
	}
	if !onResponse.IsFunction() {
		return nil
	}
	result, err := onResponse.Call(vm, request, response)
	if err != nil {
		return err
	}
	if !result.IsObject() {
		return nil
	}

	// code .
	code, err := result.Object().Get("StatusCode")
	if err != nil {
		return err
	}
	if code.IsNumber() {
		n, _ := code.ToInteger()
		response.StatusCode = int(n)
	}

	header, err := result.Object().Get("Header")
	if err != nil {
		return err
	}
	if header.IsObject() {
		keys := header.Object().Keys()
		for _, k := range keys {
			val, err := header.Object().Get(k)
			if err != nil {
				return err
			}
			response.Header[k] = val.String()
		}
	}

	// body.
	body, err := result.Object().Get("Body")
	if err != nil {
		return err
	}

	if body.IsString() {
		response.Body = body.String()
	}

	// inject javascript.
	inject, err := result.Object().Get("inject")
	if err != nil {
		return err
	}
	if inject.IsObject() {
		javascript, err := inject.Object().Get("js")
		if err != nil {
			return err
		}
		var injectJs []string
		if javascript.IsObject() {
			if javascript.Class() == "Array" {
				keys := javascript.Object().Keys()
				for _, k := range keys {
					f, err := javascript.Object().Get(k)
					if err != nil {
						return err
					}
					if f.IsString() {
						injectJs = append(injectJs, f.String())
					}
				}
			}
		}
		if javascript.IsString() {
			injectJs = append(injectJs, javascript.String())
		}
		css, err := inject.Object().Get("css")
		if err != nil {
			return err
		}
		var injectCss []string
		if css.IsObject() {
			if css.Class() == "Array" {
				keys := css.Object().Keys()
				for _, k := range keys {
					f, err := css.Object().Get(k)
					if err != nil {
						return err
					}
					if f.IsString() {
						injectCss = append(injectCss, f.String())
					}
				}
			}
		}
		if css.IsString() {
			injectCss = append(injectCss, css.String())
		}
		jsContent := getInjectObjects(injectJs)
		cssContent := getInjectObjects(injectCss)

		if len(jsContent) != 0 || len(cssContent) != 0 {
			jquery, err := goquery.NewDocumentFromReader(bytes.NewBufferString(response.Body))
			if err != nil {
				return err
			}
			head := jquery.Find("head").First()
			if len(jsContent) != 0 {
				for _, c := range jsContent {
					head.AppendHtml(fmt.Sprintf("<script>%s</script>\n", c))
				}
			}

			if len(cssContent) != 0 {
				for _, c := range cssContent {
					head.AppendHtml(fmt.Sprintf("<style>%s</style>\n", c))
				}
			}

			html, err := jquery.Html()
			if err != nil {
				return err
			}
			response.Body = html
		}
	}
	return nil
}

func getInjectObjects(injectObjects []string) []string {
	var content []string
	for _, ij := range injectObjects {
		if ij == "" {
			continue
		}
		if hasHttpPrefix(ij) {
			remoteData, err := fetchRemote(ij)
			if err != nil {
				log.Println("[warn] fetch remote javascript failed", ij, err)
				continue
			}
			content = append(content, remoteData)
		}
		abs, _ := filepath.Abs(ij)
		fi, err := os.Stat(abs)
		if err == nil {
			if !fi.IsDir() {
				data, err := ioutil.ReadFile(abs)
				if err != nil {
					log.Println("[warn] failed to read file", abs, err)
					continue
				}
				content = append(content, string(data))
			}
		}
		if strings.HasPrefix(ij, "<script>") && strings.HasSuffix(ij, "</script>") {
			ij = strings.TrimPrefix(ij, "<script>")
			ij = strings.TrimSuffix(ij, "</script>")
			content = append(content, ij)
		}
		if strings.HasPrefix(ij, "<style>") && strings.HasSuffix(ij, "</style>") {
			ij = strings.TrimPrefix(ij, "<style>")
			ij = strings.TrimSuffix(ij, "</style>")
			content = append(content, ij)
		}
	}
	return content
}

func hasHttpPrefix(s string) bool {
	return strings.HasPrefix(s, "http") || strings.HasPrefix(s, "https")
}

func fetchRemote(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func newJavascriptFilter(scripts string) ScriptFilter {
	js := new(JavascriptFilter)
	js.script = scripts
	return js
}
