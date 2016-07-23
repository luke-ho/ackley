// Copyright 2016 Luke Ho All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ackley

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"net/http"
	"syscall"
)

func (ackley *Ackley) userHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, fmt.Errorf("Must be GET request").Error(), http.StatusBadRequest)
		return
	}
	slack_users_json_bytes, err := json.Marshal(ackley.slack_users)
	if err != nil {
		http.Error(w, fmt.Errorf("Error while trying to populate data:%v", err.Error()).Error(), http.StatusInternalServerError)
		glog.Errorf(err.Error())
		return
	}
	w.Write(slack_users_json_bytes)
}
func (ackley *Ackley) channelHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, fmt.Errorf("Must be GET request").Error(), http.StatusBadRequest)
		return
	}
	slack_channels_json_bytes, err := json.Marshal(ackley.slack_channels)
	if err != nil {
		http.Error(w, fmt.Errorf("Error while trying to populate data:%v", err.Error()).Error(), http.StatusInternalServerError)
		glog.Errorf(err.Error())
		return
	}
	w.Write(slack_channels_json_bytes)
}
func (ackley *Ackley) imsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, fmt.Errorf("Must be a GET request").Error(), http.StatusBadRequest)
		return
	}
	slack_user_id_to_im_json_bytes, err := json.Marshal(&ackley.slack_user_id_to_im)
	if err != nil {
		http.Error(w, fmt.Errorf("Error while trying to populate data:%v", err.Error()).Error(), http.StatusInternalServerError)
		glog.Errorf(err.Error())
		return
	}
	w.Write(slack_user_id_to_im_json_bytes)
}
func (ackley *Ackley) spawn_web_server() {
	http.HandleFunc("/api/v1/users", ackley.userHandler)
	http.HandleFunc("/api/v1/channels", ackley.channelHandler)
	http.HandleFunc("/api/v1/ims", ackley.imsHandler)

	err := http.ListenAndServe(ackley.web_server_address, nil)
	if err != nil {
		glog.Errorf("Unable to start webserver:%v\n", err)
		ackley.interrupt_channel <- syscall.SIGINT
	}
}
