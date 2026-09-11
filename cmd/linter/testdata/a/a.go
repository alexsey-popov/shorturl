package main

import (
	"log"
	"os"
)

func main() {
	panic("test") // want "найден ручной вызов паники"
	log.Fatal("test")
	os.Exit(1)
}

func test() {
	panic("test")     // want "найден ручной вызов паники"
	log.Fatal("test") // want "найден вызов запрещённого метода log.Fatal"
	os.Exit(1)        // want "найден вызов запрещённого метода os.Exit"
}
