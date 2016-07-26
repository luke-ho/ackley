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
	"github.com/golang/glog"
	"sync/atomic"
	"syscall"
	"time"
)

func (ackley *Ackley) cleanup() {
	ackley.cleanup_mutex.Lock()
	glog.Errorf("Cleaning up...\n")
	if ackley.slack_web_socket != nil {
		err := ackley.slack_web_socket.Close()
		if err != nil {
			glog.Errorf("Error while trying to close websocket:%v\n", err)
		}
	}
	ackley.update_bot_presence()
	glog.Flush()
	ackley.cleanup_mutex.Unlock()
}

func (ackley *Ackley) flap_connection() {
	ackley.flap_connection_mutex.Lock()
	if ackley.slack_flapping_connection == false {
		glog.Infof("Begin sending flapping connection:\n")
		// Start flapping the connection
		ackley.slack_flapping_connection = true
		// Notify all return channels
		ackley.process_slack_pong_misses_return_channel <- true
		ackley.read_from_slack_websocket_return_channel <- true
		ackley.ping_slack_websocket_return_channel <- true

		ackley.slack_flap_connection_channel <- true
		glog.Infof("End sending flapping connection:\n")
	}
	ackley.flap_connection_mutex.Unlock()
}

func (ackley *Ackley) process_flap_connections() {
	// Listen for flapped connections
	for {
		select {
		case <-ackley.slack_flap_connection_channel:
			sleep_duration := time.Millisecond * time.Duration(250+((atomic.LoadUint32(&ackley.slack_num_conn_flapped)+1)*5))
			glog.Infof("Processing request to flap connection.  Sleeping for: %v\n", sleep_duration)
			time.Sleep(sleep_duration)
			atomic.AddUint32(&ackley.slack_num_conn_flapped, 1)
			if atomic.LoadUint32(&ackley.slack_num_conn_flapped) >= MAX_SLACK_CONN_FLAP {
				// Flapped too many times, it's time to exit
				glog.Errorf("Flapped too many times, exiting...\n")
				ackley.interrupt_channel <- syscall.SIGINT
				return
			}

			// reset ping processing
			ackley.reset_ping_processing()

			// Flap connection
			ackley.cleanup()

			// Empty the return channels
			for i := 0; i < len(ackley.process_slack_pong_misses_return_channel); {
				<-ackley.process_slack_pong_misses_return_channel
			}
			for i := 0; i < len(ackley.read_from_slack_websocket_return_channel); {
				<-ackley.read_from_slack_websocket_return_channel
			}
			for i := 0; i < len(ackley.ping_slack_websocket_return_channel); {
				<-ackley.ping_slack_websocket_return_channel
			}

			// reset slack structures
			ackley.slack_channels = make([]SlackChannel, 0)
			ackley.slack_users = make([]SlackUser, 0)
			ackley.slack_user_id_to_user = make(map[string]*SlackUser)
			ackley.slack_user_id_to_im = make(map[string]string)

			ackley.slack_flapping_connection = false
			ackley.establish_connection()
			// Done flapping connection

		}
	}
}

func (ackley *Ackley) connection_flap_cleanup() {
	for {
		time.Sleep(time.Second * 1)
		if atomic.LoadUint32(&ackley.slack_num_conn_flapped) > 0 {
			// Decrement slack_num_conn_flapped
			atomic.AddUint32(&ackley.slack_num_conn_flapped, ^uint32(0))
		}
	}
}

func (ackley *Ackley) slack_ping_ack_cleanup() {
	for {
		time.Sleep(time.Minute * 1)
		expiry_time := time.Now().UTC().Unix() - 60
		ackley.slack_ping_id_ack_mutex.Lock()
		for k, v := range ackley.slack_ping_id_ack {
			if v < expiry_time {
				delete(ackley.slack_ping_id_ack, k)
			}
		}
		ackley.slack_ping_id_ack_mutex.Unlock()
	}
}
