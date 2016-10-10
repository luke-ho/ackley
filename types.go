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
	"golang.org/x/net/websocket"
	"os"
	"regexp"
	"sync"
)

type MessageResponseHandler func(*SlackMessageInfo) ([]byte, error)

const (
	DEFAULT_ACKLEY_WEB_SERVER_ADDRESS = ":8080"
	DEFAULT_PERIODIC_QUOTES_DURATION  = 3000
	MAX_BUFFERED_SLACK_EVENTS         = 1000000
	MAX_SLACK_PINGS_UNANSWERED        = 5
	MAX_SLACK_MESSAGE_SIZE            = 16384 + 2
	MAX_SLACK_CONN_FLAP               = 30
)

type Ackley struct {
	slack_auth_token       string
	slack_botname          string
	slack_botid            string
	slack_web_socket       *websocket.Conn
	slack_channels         []SlackChannel
	slack_users            []SlackUser
	slack_ping_id          int64
	interrupt_channel      chan os.Signal
	interrupt_channel_resp chan bool
	websocket_origin       string

	slack_user_id_to_user map[string]*SlackUser
	slack_user_id_to_im   map[string]string
	slack_ping_id_ack     map[int64]int64

	ackley_regexp           *regexp.Regexp
	direct_message_regexp   *regexp.Regexp
	cleaning                int32
	cleanup_mutex           sync.Mutex
	slack_ping_id_ack_mutex sync.Mutex
	flap_connection_mutex   sync.Mutex
	start_web_server        bool
	web_server_address      string

	slack_event_classification_channel chan map[string]interface{}

	slack_message_processing_channel chan map[string]interface{}
	slack_pong_processing_channel    chan map[string]interface{}

	slack_flap_connection_channel chan bool
	slack_flapping_connection     bool
	slack_num_conn_flapped        uint32
	test_mode                     bool

	message_retransmission_enabled      bool
	message_retransmission_factor       int
	message_retransmission_max_duration int
	message_retransmission_channel      chan AckleySlackRetransmission
	message_retransmission_test_channel chan bool

	message_response_handler MessageResponseHandler
}

type AckleyInit struct {
	Slack_auth_token                    string
	Slack_botname                       string
	Slack_botid                         string
	Websocket_origin                    string
	Message_response_handler            MessageResponseHandler
	Web_server_address                  string
	Start_web_server                    bool
	Message_retransmission_enabled      bool
	Message_retransmission_factor       int
	Message_retransmission_max_duration int
	Test_mode                           bool
	Interrupt_channel_resp              chan bool
}

type SlackIm struct {
	Id      string
	User_id string
}

type AckleySlackRetransmission struct {
	Retrans_time int
	Message      []byte
}

type SlackMessageInfo struct {
	User        *SlackUser
	UserMessage string
	Msg         *SlackMessage
}
