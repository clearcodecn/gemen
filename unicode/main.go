package main

import (
	"fmt"
	"github.com/go-xorm/xorm"
	_ "github.com/mattn/go-oci8"
	"log"
)

func main() {

	e, err := xorm.NewEngine("oci8", "scott/Oracle1")
	if err != nil {
		log.Fatalln(err)
	}
	sess := e.NewSession()
	defer sess.Close()
	var info []*Info
	err = sess.Find(&info)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(info)
}

// 你的struct
type Info struct {
}

func (Info) TableName() string {
	return "你的表名"
}