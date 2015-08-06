package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/nats-io/nats"

	"gopkg.in/mgo.v2"
)

const (
	metricsAPIPath = "/api/metrics"
	agentsAPIPath  = "/api/agents"
)

var (
	// agents collection
	agentsC *mgo.Collection
)

var (
	flagAddr    = flag.String("addr", ":8080", "serve HTTP on `address`")
	flagNatsURL = flag.String("nats", nats.DefaultURL, "nats URL")
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("marksman: ")
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
	agentsC = session.DB("recon-dev").C("agents")

	log.Println("Server started: http://localhost" + *flagAddr)
	log.Fatal(http.ListenAndServe(*flagAddr, nil))
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "TODO")
}

func agentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		var R []Agent
		err := agentsC.Find(nil).All(&R)
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
		err := agentsC.Insert(a)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		nurl := struct {
			NatsURL string `json:"nats_url"`
		}{
			NatsURL: *flagNatsURL,
		}
		enc := json.NewEncoder(w)
		if err := enc.Encode(nurl); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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
