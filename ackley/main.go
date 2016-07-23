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

package main

import (
	"ackley"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"os"
	"os/signal"
)

const (
	VERSION = "0.1"
)

var (
	ack               *ackley.Ackley
	interrupt_channel chan os.Signal
	done_channel      chan bool
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Ackley: Slack bot version %v\n", VERSION)
		flag.PrintDefaults()
	}
	ack_init := &ackley.AckleyInit{}
	flag.StringVar(&ack_init.Web_server_address, "ack_web_server_address", ackley.DEFAULT_ACKLEY_WEB_SERVER_ADDRESS, "URL Host/IP Web server should run on")
	flag.BoolVar(&ack_init.Start_web_server, "start_ack_web_server", false, "If true, starts up the web server")
	flag.BoolVar(&ack_init.Test_mode, "test_mode", false, "If true, enables test mode for unit tests")
	flag.BoolVar(&ack_init.Message_retransmission_enabled, "message_retransmission_enabled", true, "If true, ackley will queue up and retransmit messages on message send failure")
	flag.IntVar(&ack_init.Message_retransmission_factor, "message_retransmission_factor", ackley.DEFAULT_RETRANS_FACTOR, "How much time increases per retransmission")
	flag.IntVar(&ack_init.Message_retransmission_max_duration, "message_retransmission_max_duration", ackley.DEFAULT_RETRANS_MAX_DURATION, "Maximum duration for retransmission before message is discarded")
	flag.Parse()

	ack_init.Slack_os_auth_token = os.Getenv("ACKLEY_SLACK_API_TOKEN")
	if len(ack_init.Slack_os_auth_token) < 1 {
		glog.Errorf("Error: please set the ACKLEY_SLACK_API_TOKEN environment variable\n")
		os.Exit(1)
	}

	ack_init.Slack_botname = os.Getenv("ACKLEY_BOTNAME")
	if len(ack_init.Slack_botname) < 1 {
		glog.Errorf("Error: please set the ACKLEY_BOTNAME environment variable\n")
		os.Exit(1)
	}

	ack_init.Slack_botid = os.Getenv("ACKLEY_BOTID")
	if len(ack_init.Slack_botid) < 1 {
		glog.Errorf("Error: please set the ACKLEY_BOTID environment variable\n")
		os.Exit(1)
	}

	ack_init.Websocket_origin = os.Getenv("ACKLEY_WEBSOCKET_ORIGIN")
	if len(ack_init.Websocket_origin) < 1 {
		glog.Errorf("Error: please set the ACKLEY_WEBSOCKET_ORIGIN environment variable\n")
		os.Exit(1)
	}

	done_channel = make(chan bool, 1)
	ack_init.Interrupt_channel_resp = done_channel

	QuotesInit(ack_init)
	ack_init.Message_response_handler = QuoteResponseHandler

	ack = &ackley.Ackley{}
	ack.Init(ack_init)
}

func QuoteResponseHandler(info *ackley.SlackMessageInfo) ([]byte, error) {
	text_response := fmt.Sprintf("Hi, %v!\n", info.User)

	iqr := InspirationalQuotesRequest{Type: "GET", Val: ""}
	inspirational_quotes_req_channel <- iqr
	iqr_response := <-inspirational_quotes_resp_channel
	if iqr_response.Type == "OK" {
		text_response = iqr_response.Val
	} else {
		glog.Errorf("Got a non-OK response (%v) from inspirational_quotes_resp_channel: %v\n", iqr_response.Type, iqr_response.Val)
	}

	info.Msg.Text = text_response

	response_bytes, err := json.Marshal(info.Msg)
	if err != nil {
		return nil, err
	}
	return response_bytes, nil
}

func main() {
	init_complete_channel := make(chan bool, 1)
	wait_forever := make(chan bool, 1)
	interrupt_channel = make(chan os.Signal, 1)
	signal.Notify(interrupt_channel, os.Interrupt)
	go func() {
		init_complete_channel <- true
		for {
			select {
			case v := <-interrupt_channel:
				fmt.Printf("Got an signal on the interrupt_channel:%v\n", v)
				ack.Interrupt(v)
				os.Exit(0)
			case v := <-done_channel:
				fmt.Printf("Got a signal on the done_channel, exiting:%v\n", v)
				os.Exit(0)
			}
		}
	}()
	<-init_complete_channel
	ack.Start()
	<-wait_forever
}
