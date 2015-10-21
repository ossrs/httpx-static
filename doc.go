/*
The go-srs project is a go implementation of http://github.com/simple-rtmp-server/srs.
*/
package main

import "fmt"

func main() {
    fmt.Println("Please use the following components:")
    fmt.Println("   1. srs: the srs server.")
    fmt.Println("For example:")
    fmt.Println("   go build -o objs/srs github.com/simple-rtmp-server/go-srs/srs && ./objs/srs -c conf/console.json")
}