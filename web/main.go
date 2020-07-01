package main

import (
	"fmt"
	"strconv"
)

func main() {
	var str = `http is the best way to for web""`

	s := strconv.Quote(str)
	fmt.Println(s)
}
