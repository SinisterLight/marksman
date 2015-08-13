// Copyright 2015 CodeIgnition. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/codeignition/recon"
	"gopkg.in/mgo.v2/bson"
)

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
