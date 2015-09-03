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

	"github.com/codeignition/recon/policy"
	"github.com/nats-io/nats"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// TODO: If possible, extract the path constants into a separate package
// which can be used both by marksman and recond.

// Different URL paths.
// They are not hardcoded so that changing them is easier.
const (
	eventsAPIPath = "/api/events"
	agentsAPIPath = "/api/agents"
	agentAPIPath  = agentsAPIPath + "/"
	policyAPIPath = "/api/policy"
)

// Agent represents a recon daemon running on
// a machine.
type Agent struct {
	UID          string    `json:"uid"`
	HostName     string    `json:"host_name"`
	RegisteredAt time.Time `json:"registered_at"` // time at which the agent was registered
	UpdatedAt    time.Time `json:"updated_at"`    // time at which the agent metrics were last received / updated
}

// agentOutput is the structure that's present in the JSON API.
// It specifies a contract with the clients (e.g. frontend).
// So, be careful while changing it and update the clients accordingly.
type agentOutput struct {
	Agent
	Status string `json:"status"`
}

// TODO: Instead of using a global for each collection,
// abstract this into an interface, which makes it
// easier for testing.
var (
	// agents collection
	agentsC *mgo.Collection

	// events collection
	eventsC *mgo.Collection
)

// NATS variables
var (
	natsEncConn *nats.EncodedConn
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
	mux.HandleFunc(eventsAPIPath, eventsHandler)
	mux.HandleFunc(agentsAPIPath, agentsHandler)
	mux.HandleFunc(agentAPIPath, agentHandler)
	mux.HandleFunc(policyAPIPath, policyHandler)

	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)
	agentsC = session.DB("recon-dev").C("agents")
	eventsC = session.DB("recon-dev").C("events")

	nc, err := nats.Connect(*flagNatsURL)
	if err != nil {
		log.Fatalln(err)
	}
	natsEncConn, err = nats.NewEncodedConn(nc, "json")
	if err != nil {
		log.Fatalln(err)
	}
	defer natsEncConn.Close()

	natsEncConn.Subscribe("policy_events", func(e *policy.Event) {
		uid := e.Policy.AgentUID
		var a Agent
		err := agentsC.Find(bson.M{"uid": uid}).One(&a)
		if err != nil {
			log.Printf("failed to find the agent %s: %s", uid, err)
			return
		}

		a.UpdatedAt = time.Now()
		err = agentsC.Update(bson.M{"uid": uid}, a)
		if err != nil {
			log.Printf("failed to update the agent %s: %s", uid, err)
		}

		err = eventsC.Insert(e)
		if err != nil {
			log.Printf("failed to insert event: %s", err)
		}

	})

	log.Println("Server started: http://localhost" + *flagAddr)
	log.Fatal(http.ListenAndServe(*flagAddr, mux))
}

// Status returns the status of the agent.
// It is calculated on-demand. If the agent was registered
// or its metrics were updated in the last 10 seconds, it
// returns "online", else "offline"
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

		b := agentOutput{
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

		b := make([]agentOutput, len(agents))
		for i, a := range agents {
			b[i] = agentOutput{
				Agent:  a,
				Status: a.Status(),
			}
		}

		enc := json.NewEncoder(w)
		if err := enc.Encode(b); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	case "POST":
		var a Agent
		defer r.Body.Close()
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&a); err != nil {
			http.Error(w, "unable to decode json", http.StatusBadRequest)
			return
		}
		if a.UID == "" {
			http.Error(w, "UID can't be empty", http.StatusBadRequest)
			return
		}
		// check whether the agent is already registered.
		// Insert it only if its not registered else just
		// update the UpdatedAt field for the agent in the DB.
		err := agentsC.Find(bson.M{"uid": a.UID}).One(&a)
		if err != nil {
			if err == mgo.ErrNotFound {
				a.RegisteredAt = time.Now()
				a.UpdatedAt = time.Now()
				if err := agentsC.Insert(a); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			a.UpdatedAt = time.Now()
			err = agentsC.Update(bson.M{"uid": a.UID}, a)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
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
