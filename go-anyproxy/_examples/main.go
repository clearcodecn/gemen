package main

import (
	"fmt"
	"github.com/robertkrimen/otto"
)

func main() {
	ot := otto.New()

	val, _ := ot.Run(`if("aaa".lastIndexOf("a") != -1) {console.log(222)}else{console.log(2)};`)
	fmt.Println(val)
}
