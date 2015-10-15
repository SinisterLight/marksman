// Copyright 2015 CodeIgnition. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/codeignition/recon/policy"
	"gopkg.in/mgo.v2/bson"
)

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		uid := r.FormValue("uid")
		if uid == "" {
			http.Error(w, "uid query string required", http.StatusInternalServerError)
			return
		}
		policyName := r.FormValue("policy_name")
		if policyName == "" {
			http.Error(w, "policy_name query string required", http.StatusInternalServerError)
			return
		}
		t := r.FormValue("t")
		if t == "" {
			t = "5m"
		}
		d, err := time.ParseDuration(t)
		if err != nil {
			http.Error(w, "malformed time duration", http.StatusInternalServerError)
			return
		}
		var events []policy.Event
		err = eventsC.Find(bson.M{
			"policy.agentuid": uid,
			"policy.name":     policyName,
			"time":            bson.M{"$gt": time.Now().Add(-d)},
		}).All(&events)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		enc := json.NewEncoder(w)
		if err := enc.Encode(events); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
}
