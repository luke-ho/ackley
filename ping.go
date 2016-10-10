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
	"github.com/golang/glog"
	"math"
	"sync/atomic"
	"time"
)

func (ackley *Ackley) process_slack_pong() {
	for {
		select {
		case slack_pong := <-ackley.slack_pong_processing_channel:
			glog.Infof("Got a pong request from slack:%v\n", slack_pong)
			reply_to, ok := slack_pong["reply_to"].(float64)
			if ok == false {
				glog.Errorf("Unable to retrieve reply_to value from slack pong:%v\n", slack_pong)
			}
			// Check to see if this one is in the map
			if _, ok := ackley.slack_ping_id_ack[int64(reply_to)]; ok {
				ackley.slack_ping_id_ack_mutex.Lock()
				// Remove ping from the map
				delete(ackley.slack_ping_id_ack, int64(reply_to))
				ackley.slack_ping_id_ack_mutex.Unlock()
			} else {
				// Mismatch, flap the connection
				glog.Errorf("Mismatch in pong processing:%v, flapping connection\n", reply_to)
				ackley.flap_connection()
				continue
			}
		}
	}
}

func (ackley *Ackley) process_pong_misses() {
	for {
		select {
		case <-time.After(time.Second * time.Duration(2)):
			// Now check the size of the map
			ackley.slack_ping_id_ack_mutex.Lock()
			pings_remaining := len(ackley.slack_ping_id_ack)
			ackley.slack_ping_id_ack_mutex.Unlock()

			if pings_remaining > MAX_SLACK_PINGS_UNANSWERED {
				glog.Errorf("Flapping connection. Haven't heard back for %v pongs\n", MAX_SLACK_PINGS_UNANSWERED)
				ackley.flap_connection()
				continue
			}
		}
	}
}

func (ackley *Ackley) ping_slack_websocket() {
	for {
		time.Sleep(time.Second * 1)
		slack_ping := &SlackPing{Id: atomic.LoadInt64(&ackley.slack_ping_id), Type: "ping", Time: time.Now().UTC().Unix()}
		slack_ping_bytes, err := json.Marshal(slack_ping)
		if err != nil {
			glog.Errorf("Error while trying to marshal slack_ping: %v\n", err)
			ackley.flap_connection()
			continue
		}
		ackley.slack_ping_id_ack_mutex.Lock()
		ackley.slack_ping_id_ack[atomic.LoadInt64(&ackley.slack_ping_id)] = slack_ping.Time
		ackley.slack_ping_id_ack_mutex.Unlock()
		glog.Infof("Writing ping:%v\n", slack_ping)
		if ackley.slack_web_socket != nil {
			ackley.slack_web_socket.Write(slack_ping_bytes)
			atomic.AddInt64(&ackley.slack_ping_id, 1)
		} else {
			glog.Infof("Skipping write as slack web socket is nil\n")
		}
		if atomic.LoadInt64(&ackley.slack_ping_id) >= math.MaxInt64 {
			glog.Infof("Resetting slack_ping_id...\n")
			atomic.StoreInt64(&ackley.slack_ping_id, 0)
		}
	}
}

func (ackley *Ackley) reset_ping_processing() {
	ackley.slack_ping_id_ack_mutex.Lock()
	ackley.slack_ping_id_ack = make(map[int64]int64)
	ackley.slack_ping_id_ack_mutex.Unlock()
	atomic.StoreInt64(&ackley.slack_ping_id, 1)
}
