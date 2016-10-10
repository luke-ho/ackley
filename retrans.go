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
	"fmt"
	"github.com/golang/glog"
	"time"
)

const (
	DEFAULT_RETRANS_FACTOR       = 2
	DEFAULT_RETRANS_MAX_DURATION = 32
	MAX_BUFFERED_RETRANSMISSIONS = 1000000
)

func (ackley *Ackley) process_message_retransmissions() {
	for {
		select {
		case msg := <-ackley.message_retransmission_channel:
			go func(AckleySlackRetransmission) {
				var err error
				glog.Infof("Sleeping for %v...\n", msg.Retrans_time)
				time.Sleep(time.Second * time.Duration(msg.Retrans_time))
				if ackley.slack_web_socket != nil {
					_, err = ackley.slack_web_socket.Write(msg.Message)
				} else {
					err = fmt.Errorf("slack web socket is nil, retry")
				}
				if err != nil || ackley.test_mode == true {
					if err != nil {
						glog.Errorf("Error while trying to write slack message response:%v, Message:%v.\n", err, string(msg.Message))
					}
					// Add more time to the retransmission
					msg.Retrans_time *= ackley.message_retransmission_factor
					glog.Infof("Retrans_time is now: %v...\n", msg.Retrans_time)
					if msg.Retrans_time > ackley.message_retransmission_max_duration {
						glog.Errorf("Discarding message: %v\n", string(msg.Message))
						if ackley.test_mode == true {
							ackley.message_retransmission_test_channel <- true
						}
					} else {
						ackley.message_retransmission_channel <- msg
					}
				}
			}(msg)
		}
	}
}
