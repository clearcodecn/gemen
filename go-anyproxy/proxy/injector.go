package proxy

type ScriptFilter interface {
	FilterRequest(r *Request) bool
	FilterResponse(request *Request, response *Response)
}

//type JavascriptFilter struct {
//	Scripts []string
//}
//
//const (
//	callInjectRequest  = "injectRequest(request);"
//	callInjectResponse = "injectResponse(request,response);"
//)
//
//func getScriptRequest(src []byte, r *http.Request) []byte {
//	var header string
//	for key := range r.Header {
//		header += fmt.Sprintf(`"%s": %s,`, key, strconv.Quote(r.Header.Get(key)))
//	}
//	obj := fmt.Sprintf(`;var request = {
//			headers: {%s},
//			url: "%s",
//		};%s`, header, r.URL.String(), callInjectRequest)
//	return append(src, obj...)
//}
//
//func getScriptRequest2(src []byte, r *http.Request) []byte {
//	var header string
//	for key := range r.Header {
//		header += fmt.Sprintf(`"%s": %s,`, key, strconv.Quote(r.Header.Get(key)))
//	}
//	obj := fmt.Sprintf(`;var request = {
//			headers: {%s},
//			url: "%s",
//		}`, header, r.URL.String())
//	return append(src, obj...)
//}
//
//func getScriptResponse(src []byte, response *http.Response) ([]byte, error) {
//	var headers string
//	for key := range response.Header {
//		headers += fmt.Sprintf(`"%s": %s,`, key, strconv.Quote(response.Header.Get(key)))
//	}
//	b, err := ioutil.ReadAll(response.Body)
//	if err != nil {
//		return nil, err
//	}
//	obj := fmt.Sprintf(`;var response = {
//			statusCode: %d,
//			header: {%s},
//			body: "%s",
//		};`, response.StatusCode, headers, b)
//	return append(src, obj...), nil
//}
//
//func (js *JavascriptFilter) FilterRequest(r *http.Request) {
//	for _, script := range js.Scripts {
//		ot := otto.New()
//		data, err := ioutil.ReadFile(script)
//		if err != nil {
//			log.Println("[warn] failed to read script: ", script)
//			return
//		}
//		data = getScriptRequest(data, r)
//
//		result, err := ot.Run(data)
//		if err != nil {
//			log.Printf("[warn] failed to execute script %s: %v\n", script, err)
//			return
//		}
//		headers, err := result.Object().Get("headers")
//		if err != nil {
//			return
//		}
//		keys := headers.Object().Keys()
//		for _, k := range keys {
//			val, err := headers.Object().Get(k)
//			if err != nil {
//				log.Printf("[warn] failed to merge headers in script %s: %v\n", script, err)
//				return
//			}
//			r.Header.Set(k, val.String())
//		}
//		// write body.
//		body, err := result.Object().Get("body")
//		if err != nil {
//			return
//		}
//		if body.String() != "undefined" {
//			requestBody := []byte(body.String())
//			nop := ioutil.NopCloser(bytes.NewBuffer(requestBody))
//			r.Body = nop
//		}
//	}
//}
//
//func (js *JavascriptFilter) FilterResponse(request *http.Request, response *http.Response) {
//	for _, script := range js.Scripts {
//		ot := otto.New()
//		data, err := ioutil.ReadFile(script)
//		if err != nil {
//			log.Println("[warn] failed to read script: ", script)
//			continue
//		}
//		data, err = getScriptResponse(data, response)
//		if err != nil {
//			log.Println("[warn] failed to filter response", request.URL.String(), err)
//			return
//		}
//		data = getScriptRequest(data, request)
//		data = append(data, callInjectResponse...)
//
//		result, err := ot.Run(data)
//		if err != nil {
//			log.Printf("[warn] failed to execute script url: %s, %s: %s, %v\n", request.URL.String(), script, data, err)
//			return
//		}
//
//		headers, err := result.Object().Get("header")
//		if err != nil {
//			return
//		}
//		keys := headers.Object().Keys()
//		for _, k := range keys {
//			val, err := headers.Object().Get(k)
//			if err != nil {
//				return
//			}
//
//			response.Header.Set(k, val.String())
//		}
//		body, err := result.Object().Get("body")
//		if err != nil {
//			return
//		}
//		bodyLength := len(body.String())
//		if bodyLength != 0 {
//			response.ContentLength = int64(bodyLength)
//			response.Body = ioutil.NopCloser(bytes.NewBufferString(body.String()))
//		}
//	}
//}
//
//func newJavascriptFilter(scripts []string) ScriptFilter {
//	js := new(JavascriptFilter)
//	js.Scripts = scripts
//	return js
//}
//
//var availableHeaderList = []string{
//	"text/html",
//	"text/plain",
//	"application/json",
//}
//
//func availableHeader(header string) bool {
//	for _, v := range availableHeaderList {
//		if strings.Contains(header, v) {
//			return true
//		}
//	}
//	return false
//}
//
//type Request struct {
//	URL    url.URL     `json:"url"`
//	Header http.Header `json:"header"`
//
//	Body string `json:"body"`
//}
//
//type Response struct {
//	Header http.Header `json:"header"`
//	Body   string      `json:"body"`
//
//	Code int `json:"code"`
//}
//
//type RequestHandler func(request *Request)
//
//func javascriptRequestHandler(vm *otto.Object, req *http.Request) RequestHandler {
//	return func(request *Request) {
//		vm.Set("getRequest", func(call otto.FunctionCall) otto.Value {
//			newRequest := new(Request)
//			newRequest.URL = *req.URL
//			newRequest.Header = make(http.Header)
//			for k := range req.Header {
//				newRequest.Header.Set(k, req.Header.Get(k))
//			}
//			if req.Body != nil {
//				body, err := ioutil.ReadAll(req.Body)
//				if err != nil {
//					return otto.Value{}
//				}
//				newRequest.Body = string(body)
//				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
//			}
//			val, err := otto.ToValue(newRequest)
//			if err != nil {
//				return otto.Value{}
//			}
//			return val
//		})
//	}
//}
