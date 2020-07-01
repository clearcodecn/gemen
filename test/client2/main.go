package main

import (
	"bufio"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"sync"
	"time"
)

func main() {
	wg := sync.WaitGroup{}
	go func() {
		defer wg.Done()
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
		http.ListenAndServe(":1122", nil)
	}()

	wg.Add(2)
	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Second)
		f, err := os.OpenFile("/tmp/1.big", os.O_RDONLY, 0777)
		if err != nil {
			log.Println(err)
			return
		}
		r := bufio.NewReader(f)
		defer f.Close()

		multipart.NewWriter()

		req, err := http.NewRequest(http.MethodPost, "http://localhost:1122/", r)
		if err != nil {
			log.Println(err)
			return
		}
		_, err = http.DefaultClient.Do(req)
		if err != nil {
			log.Println(err)
		}
	}()

	wg.Wait()
}
