package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"github.com/codeignition/recon"
	"github.com/nats-io/nats"

	"gopkg.in/mgo.v2"
)

const (
	metricsAPIPath = "/api/metrics"
	agentsAPIPath  = "/api/agents"
)

// TODO: Instead of using a global for each collection,
// abstract this into an interface, which makes it
// easier for testing.
var (
	// agents collection
	agentsC *mgo.Collection

	// metrics collection
	metricsC *mgo.Collection
)

// Command line flags
// prepend flag to variable names to not pollute the global namespace.
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
	metricsC = session.DB("recon-dev").C("metrics")

	log.Println("Server started: http://localhost" + *flagAddr)
	log.Fatal(http.ListenAndServe(*flagAddr, nil))
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		var metrics []map[string]interface{}
		err := metricsC.Find(nil).All(&metrics)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		enc := json.NewEncoder(w)
		if err := enc.Encode(metrics); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	case "POST":
		var data map[string]interface{}
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&data); err != nil {
			http.Error(w, "unable to decode json", http.StatusBadRequest)
			return
		}
		err := metricsC.Insert(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

func agentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		var agents []recon.Agent
		err := agentsC.Find(nil).All(&agents)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		enc := json.NewEncoder(w)
		if err := enc.Encode(agents); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	case "POST":
		var a recon.Agent
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
