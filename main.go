package main

import (
	"flag"
	"log"
	"net/http"
)

const (
	statsPath = "/stats"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("marksman: ")
	var (
		addr = flag.String("addr", ":8080", "serve HTTP on `address`")
	)
	flag.Parse()

	http.HandleFunc(statsPath, statsHandler)
	http.Handle("/", http.FileServer(http.Dir("./public")))
	log.Println("Server started: http://localhost" + *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	panic("TODO")
}
