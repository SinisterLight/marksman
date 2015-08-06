package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"gopkg.in/mgo.v2"
)

const (
	metricsAPIPath = "/api/metrics"
	agentsAPIPath  = "/api/agents"
)

var (
	ca *mgo.Collection
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("marksman: ")

	var (
		addr = flag.String("addr", ":8080", "serve HTTP on `address`")
	)

	flag.Parse()

	http.HandleFunc(metricsAPIPath, metricsHandler)
	http.HandleFunc(agentsAPIPath, agentsHandler)
	http.Handle("/", http.FileServer(http.Dir("./public")))

	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)
	ca = session.DB("recon-dev").C("agents")

	log.Println("Server started: http://localhost" + *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "TODO")
}

func agentsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		var R []Agent
		err := ca.Find(nil).All(&R)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		enc := json.NewEncoder(w)
		if err := enc.Encode(R); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	case "POST":
		var a Agent
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&a); err != nil {
			http.Error(w, "unable to decode json", http.StatusBadRequest)
			return
		}
		if a.UID == "" {
			http.Error(w, "UID can't be empty", http.StatusBadRequest)
			return
		}
		err := ca.Insert(a)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(a.UID))
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// Agent represents a recon agent running on
// a machine.
type Agent struct {
	UID string
}
