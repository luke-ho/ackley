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
	"math/rand"
	"os"
	"regexp"
	"time"
)

func (ackley *Ackley) Init(ai *AckleyInit) {
	ackley.slack_auth_token = ai.Slack_auth_token
	ackley.slack_botname = ai.Slack_botname
	ackley.slack_botid = ai.Slack_botid
	ackley.web_server_address = ai.Web_server_address
	ackley.start_web_server = ai.Start_web_server
	ackley.test_mode = ai.Test_mode
	ackley.websocket_origin = ai.Websocket_origin
	ackley.message_retransmission_enabled = ai.Message_retransmission_enabled
	ackley.message_retransmission_factor = ai.Message_retransmission_factor
	ackley.message_retransmission_max_duration = ai.Message_retransmission_max_duration

	ackley.slack_channels = make([]SlackChannel, 0)
	ackley.slack_users = make([]SlackUser, 0)
	ackley.slack_user_id_to_user = make(map[string]*SlackUser)
	ackley.slack_user_id_to_im = make(map[string]string)
	ackley.slack_ping_id_ack = make(map[int64]int64)
	ackley.slack_ping_id = 1

	bot_regexp := fmt.Sprintf("(?i)%v|%v", ai.Slack_botname, ai.Slack_botid)
	ackley.ackley_regexp = regexp.MustCompile(bot_regexp)
	ackley.direct_message_regexp = regexp.MustCompile(`^D`)

	ackley.interrupt_channel = make(chan os.Signal, 1)
	ackley.interrupt_channel_resp = ai.Interrupt_channel_resp

	ackley.slack_event_classification_channel = make(chan map[string]interface{}, MAX_BUFFERED_SLACK_EVENTS)
	ackley.slack_message_processing_channel = make(chan map[string]interface{}, MAX_BUFFERED_SLACK_EVENTS)
	ackley.slack_pong_processing_channel = make(chan map[string]interface{}, MAX_BUFFERED_SLACK_EVENTS)
	ackley.slack_flap_connection_channel = make(chan bool, 5)

	ackley.process_slack_event_classification_return_channel = make(chan bool, 100)
	ackley.process_slack_message_return_channel = make(chan bool, 100)
	ackley.process_slack_pong_return_channel = make(chan bool, 100)
	ackley.process_slack_pong_misses_return_channel = make(chan bool, 100)
	ackley.read_from_slack_websocket_return_channel = make(chan bool, 100)
	ackley.ping_slack_websocket_return_channel = make(chan bool, 100)

	ackley.slack_flapping_connection = false
	ackley.slack_num_conn_flapped = 0

	if ackley.message_retransmission_enabled == true {
		ackley.message_retransmission_channel = make(chan AckleySlackRetransmission, MAX_BUFFERED_RETRANSMISSIONS)
	}

	if ackley.test_mode == true {
		ackley.message_retransmission_test_channel = make(chan bool)
	}

	ackley.message_response_handler = ai.Message_response_handler

	rand.Seed(time.Now().UTC().Unix())
}

func (ackley *Ackley) Start() {
	go ackley.listen_for_interrupts()
	go ackley.process_flap_connections()
	go ackley.connection_flap_cleanup()
	if ackley.start_web_server == true {
		go ackley.spawn_web_server()
	}
	go ackley.slack_ping_ack_cleanup()
	if ackley.message_retransmission_enabled {
		go ackley.process_message_retransmissions()
	}

	ackley.establish_connection()
}

func (ackley *Ackley) Interrupt(interrupt os.Signal) bool {
	ackley.interrupt_channel <- interrupt
	return <-ackley.interrupt_channel_resp
}
