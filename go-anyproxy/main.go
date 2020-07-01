package main

import (
	"github.com/clearcodecn/ca-gen/go-anyproxy/proxy"
	"log"
)

func main() {
	log.SetFlags(log.Lshortfile)
	p := proxy.New("./request.js", "./cert.pem", "./key.pem")
	p.Run(":1111")
}
