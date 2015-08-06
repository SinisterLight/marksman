package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

const (
	metricsPath = "/metrics"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("marksman: ")

	var (
		addr = flag.String("addr", ":8080", "serve HTTP on `address`")
	)

	flag.Parse()

	http.HandleFunc(metricsPath, metricsHandler)
	http.Handle("/", http.FileServer(http.Dir("./public")))

	log.Println("Server started: http://localhost" + *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "TODO")
}
