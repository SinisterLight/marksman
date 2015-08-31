// Copyright 2015 CodeIgnition. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"net/http"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/codeignition/recon/policy"
)

func policyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "POST":
		var p policy.Policy
		defer r.Body.Close()
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&p); err != nil {
			http.Error(w, "unable to decode json", http.StatusBadRequest)
			return
		}
		if p.AgentUID == "" {
			http.Error(w, "UID can't be empty", http.StatusBadRequest)
			return
		}
		var a Agent
		err := agentsC.Find(bson.M{"uid": p.AgentUID}).One(&a)
		if err != nil {
			http.Error(w, "recond agent unknown: check the agent UID and try again", http.StatusInternalServerError)
			return
		}
		if a.Status() == "offline" {
			http.Error(w, "recond agent offline: restart the agent and try again", http.StatusInternalServerError)
			return
		}
		var reply string
		if err := natsEncConn.Request(p.AgentUID+"_policy", &p, &reply, 5*time.Second); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if reply != "policy ack" {
			http.Error(w, reply, http.StatusInternalServerError)
			return
		}
		// TODO: if the policy is successfully sent to the agent, store the agent's PolicyConfig in the master server's DB as well.
		return
	case "PATCH":
		// TODO: implement
	case "DELETE":
		// TODO: implement
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
}
