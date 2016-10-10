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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"golang.org/x/net/websocket"
	"io/ioutil"
	"net/http"
	"sync/atomic"
	"syscall"
	"time"
)

func (ackley *Ackley) process_slack_event_classification() {
	for {
		select {
		case slack_event := <-ackley.slack_event_classification_channel:
			glog.Infof("Got slack event:%v\n", slack_event)
			slack_message_type, ok := slack_event["type"].(string)
			if ok == false {
				glog.Errorf("Error while trying to retrieve slack message type:%v\n", ok)
				ackley.flap_connection()
				continue
			}
			switch slack_message_type {
			case "message":
				ackley.slack_message_processing_channel <- slack_event
			case "pong":
				ackley.slack_pong_processing_channel <- slack_event
			case "team_migration_started":
				glog.Infof("Bringing down connection...\n")
				ackley.flap_connection()
			case "hello":
				glog.Infof("Got a hello request type from slack:%v\n", slack_message_type)
			default:
				glog.Infof("Unknown slack message type:%v\n", slack_message_type)
			}
		}
	}
}

func (ackley *Ackley) process_slack_message() {
	for {
		select {
		case slack_event := <-ackley.slack_message_processing_channel:
			glog.Infof("Got a message from slack:%v\n", slack_event)
			// Check the subtype
			if slack_message_subtype, ok := slack_event["subtype"].(string); ok {
				if slack_message_subtype == "message_deleted" {
					// Skip message_deleted subtype
					glog.Infof("Skipping message deleted subtype\n")
					continue
				}
				if slack_message_subtype == "message_changed" {
					// Skip message_changed subtype
					glog.Infof("Skipping message changed subtype\n")
					continue
				}
				if slack_message_subtype == "bot_message" {
					// Skip bot_message subtype
					// TBD: Support attachments
					glog.Infof("Skipping bot_message subtype\n")
					continue
				}
			}

			message_text, ok := slack_event["text"].(string)
			if ok == false {
				glog.Errorf("Error: Unable to get text from slack event\n")
				ackley.flap_connection()
				continue
			}
			slack_response_channel, ok := slack_event["channel"].(string)
			if ok == false {
				glog.Errorf("Error: Unable to respond to channel who sent message:\n")
				ackley.flap_connection()
				continue
			}
			if ackley.ackley_regexp.MatchString(message_text) == false {
				// Check the channel to see if it's a direct message
				if ackley.direct_message_regexp.MatchString(slack_response_channel) == false {
					// Not a direct message, continue
					glog.Infof("Ignoring this message as it does not contain the bot name or ackley's user id, and was not a direct message:%v\n", message_text)
					continue
				}
			}
			responder_user_id, ok := slack_event["user"].(string)
			if ok == false {
				glog.Errorf("Error: Unable to respond to user who sent message:\n")
				ackley.flap_connection()
				continue
			}
			if responder_user_id == ackley.slack_botid {
				glog.Info("Not responding to myself\n")
				continue
			}

			var ackley_user_id string
			for _, user := range ackley.slack_users {
				if user.Name == "ackley" {
					ackley_user_id = user.Id
				}
			}
			if user, ok := ackley.slack_user_id_to_user[responder_user_id]; ok {
				ackley.send_typing_event(slack_response_channel)
				current_ts := float64(time.Now().UTC().Unix())
				slack_message_response := &SlackMessage{Type: "message", Channel: slack_response_channel, User: ackley_user_id, Ts: &current_ts}
				slack_message_info := &SlackMessageInfo{User: user, UserMessage: message_text, Msg: slack_message_response}
				var err error
				var slack_message_response_bytes []byte
				if ackley.message_response_handler != nil {
					slack_message_response_bytes, err = ackley.message_response_handler(slack_message_info)
				} else {
					slack_message_response_bytes, err = ackley.make_default_response(slack_message_info)
				}

				if err != nil {
					glog.Errorf("Error in callback for slack message response:%v\n", err)
					ackley.interrupt_channel <- syscall.SIGINT
					return
				}
				if slack_message_response_bytes == nil || len(slack_message_response_bytes) < 1 {
					glog.Errorf("Callback returned nil response, ignoring...\n")
					continue
				}
				if atomic.LoadInt32(&ackley.cleaning) == 0 {
					_, err = ackley.slack_web_socket.Write(slack_message_response_bytes)
				} else {
					err = fmt.Errorf("ackley is still cleaning")
				}
				if err != nil || ackley.test_mode == true {
					if err != nil {
						glog.Errorf("Error while trying to write slack message response:%v.\n", err)
					}
					if ackley.message_retransmission_enabled == true {
						glog.Infof("Begin retransmission...\n")
						go func() {
							ackley.message_retransmission_channel <- AckleySlackRetransmission{Retrans_time: ackley.message_retransmission_factor, Message: slack_message_response_bytes}
						}()
					}
				}
			}
		}
	}
}

func (ackley *Ackley) read_from_slack_websocket() {
	for {
		select {
		default:
			var msg []byte
			var err error
			var num_bytes_read int
			slack_websocket_nil := false
			msg = make([]byte, MAX_SLACK_MESSAGE_SIZE)
			if atomic.LoadInt32(&ackley.cleaning) == 0 {
				num_bytes_read, err = ackley.slack_web_socket.Read(msg)
			} else {
				err = fmt.Errorf("ackley is still cleaning up")
				slack_websocket_nil = true
			}
			if err != nil {
				if err.Error() != "EOF" {
					if slack_websocket_nil == false {
						glog.Errorf("Error while trying to read from websocket: %v\n", err)
						ackley.flap_connection()
					}
				}
				continue
			}
			// Remove extraneous bytes from msg
			if num_bytes_read+1 < len(msg) {
				msg = msg[:num_bytes_read]
			}
			glog.Infof("Received(%v):%s\n", num_bytes_read, string(msg))

			var slack_event map[string]interface{}
			err = json.Unmarshal(msg, &slack_event)
			if err != nil {
				glog.Errorf("Error while trying to unmarshal slack event:%v\n", err)
				ackley.flap_connection()
			}

			response, ok := slack_event["ok"].(bool)
			if ok == true && response == true {
				glog.Infof("Received ok response, continuing\n")
				continue
			} else if !(ok == false && response == false) {
				if ok == true && response == false {
					glog.Errorf("Something bad is happening. Exiting. response:%v, ok:%v\n", response, ok)
					ackley.interrupt_channel <- syscall.SIGINT
					return
				} else {
					glog.Errorf("Received ok response, but something's wrong.  Ignoring for now.  response:%v, ok:%v\n", response, ok)
					continue
				}
			}
			ackley.slack_event_classification_channel <- slack_event
		}
	}
}
func (ackley *Ackley) establish_connection() {
	// Issue a POST request to slack rtm
	rtm_start_post_values := fmt.Sprintf("%s=%s", "token", ackley.slack_auth_token)
	resp, err := http.Post("https://slack.com/api/rtm.start", "application/x-www-form-urlencoded", bytes.NewReader([]byte(rtm_start_post_values)))
	if err != nil {
		glog.Errorf("Error while trying to get rtm start:%v\n", err)
		ackley.flap_connection()
		return
	}

	buf, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		glog.Errorf("Error while trying to read body of rtm request:%v\n", err)
		ackley.flap_connection()
		return
	}
	if resp.StatusCode != http.StatusOK {
		glog.Errorf("Resp status code != OK: Status Code: %v:Body:%v\n", resp.StatusCode, string(buf))
		ackley.flap_connection()
		return
	}

	var rtm_start_json map[string]interface{}
	rtm_start_json = make(map[string]interface{})

	err = json.Unmarshal(buf, &rtm_start_json)
	if err != nil {
		glog.Errorf("Unable to unmarshal response: %v\n", err)
		ackley.flap_connection()
		return
	}
	glog.Infof("Buffer:%v\n", string(buf))
	if ok_resp, ok := rtm_start_json["ok"].(bool); !ok || ok_resp != true {
		glog.Errorf("Error while starting rtm.Start: ok resp: %v\n", ok_resp)
		ackley.flap_connection()
		return
	}
	ackley.store_slack_channel_info(rtm_start_json)
	ackley.store_slack_users(rtm_start_json)
	ackley.store_slack_ims_info(rtm_start_json)

	// Get connection url
	if connection_url, success := rtm_start_json["url"].(string); success {
		// Connect via websocket
		if ackley.slack_web_socket, err = websocket.Dial(connection_url, "", ackley.websocket_origin); err == nil {
			glog.Infof("Connected!\n")
			atomic.StoreInt32(&ackley.cleaning, 0)
			ackley.update_bot_presence()
		} else {
			glog.Errorf("Error while trying to connect to websocket:%v\n", err)
			ackley.flap_connection()
			return
		}
	} else {
		glog.Errorf("Error while trying to get connection_url\n")
		ackley.flap_connection()
		return
	}
}
