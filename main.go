// Copyright 2015 CodeIgnition. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/codeignition/recon"
	"github.com/nats-io/nats"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	metricsAPIPath = "/api/metrics"
	agentsAPIPath  = "/api/agents"
	agentAPIPath   = agentsAPIPath + "/"
)

// Agent represents a recon daemon running on
// a machine.
type Agent struct {
	UID          string    `json:"uid"`
	RegisteredAt time.Time `json:"registered_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

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

	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir("./public")))
	mux.HandleFunc(metricsAPIPath, metricsHandler)
	mux.HandleFunc(agentsAPIPath, agentsHandler)
	mux.HandleFunc(agentAPIPath, agentHandler)

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
	log.Fatal(http.ListenAndServe(*flagAddr, mux))
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		var metrics []recon.Metric
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
		var m recon.Metric
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&m); err != nil {
			http.Error(w, "unable to decode json", http.StatusBadRequest)
			return
		}
		err := metricsC.Insert(m)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var a Agent
		err = agentsC.Find(bson.M{"uid": m.AgentUID}).One(&a)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		a.UpdatedAt = time.Now()
		err = agentsC.Update(bson.M{"uid": m.AgentUID}, a)
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

func (a Agent) Status() string {
	if time.Since(a.UpdatedAt) > 10*time.Second {
		return "offline"
	}
	return "online"
}

func agentHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		uid := r.URL.Path[len(agentAPIPath):]
		var a Agent
		err := agentsC.Find(bson.M{"uid": uid}).One(&a)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		b := struct {
			Agent
			Status string `json:"status"`
		}{
			Agent:  a,
			Status: a.Status(),
		}

		enc := json.NewEncoder(w)
		if err := enc.Encode(b); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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
		var agents []Agent
		err := agentsC.Find(nil).All(&agents)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		type AgentOutput struct {
			Agent
			Status string `json:"status"`
		}
		b := make([]AgentOutput, len(agents))
		for i, a := range agents {
			b[i] = AgentOutput{a, a.Status()}
		}

		enc := json.NewEncoder(w)
		if err := enc.Encode(b); err != nil {
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

		a.RegisteredAt = time.Now()
		a.UpdatedAt = time.Now()

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
