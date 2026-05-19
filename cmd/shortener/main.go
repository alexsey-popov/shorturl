package main

import (
	"net/http"

	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/handler"
)

func main() {
	h := http.HandlerFunc(handler.RouterFunc)

	err := http.ListenAndServe(config.ServerAddr, h)
	if err != nil {
		panic(err)
	}
}
