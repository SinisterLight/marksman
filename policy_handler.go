// Copyright 2015 CodeIgnition. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"net/http"
	"time"

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
		var replyErr error
		if err := natsEncConn.Request(p.AgentUID+"_policy", &p, &replyErr, 5*time.Second); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if replyErr != nil {
			http.Error(w, replyErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
}
