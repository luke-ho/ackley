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
	"flag"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

const (
	TEST_RETRANS_FACTOR            = 2
	TEST_RETRANS_MAX_DURATION      = 4
	VERSION                        = "0.1"
	DEFAULT_SLACK_RESPONSE_CHANNEL = "D1ESSDZU6"
)

var (
	ack                                   *Ackley
	done_channel                          chan bool
	slack_retransmission_response_channel string
)

func Test1Retransmission(t *testing.T) {
	if ack.message_retransmission_enabled == true {
		// TBD: Hardcoded
		slack_response_channel := "D1ESSDZU6"
		var ackley_user_id string
		for _, user := range ack.slack_users {
			if user.Name == "ackley" {
				ackley_user_id = user.Id
			}
		}

		text_response := "This is a retransmission test. Please ignore"
		slack_message_response := &SlackMessage{Type: "message", Channel: slack_response_channel, User: ackley_user_id, Text: text_response, Ts: string(time.Now().UTC().Unix())}
		slack_message_response_bytes, err := json.Marshal(slack_message_response)
		if err != nil {
			t.Error("Error while trying to marshal json for slack message response:%v\n", err)
			return
		}

		ack.message_retransmission_channel <- AckleySlackRetransmission{Retrans_time: 1, Message: slack_message_response_bytes}
		<-ack.message_retransmission_test_channel
	}
}

func Test2FlappingConnection(t *testing.T) {
	for i := 0; i < 10; i++ {
		ack.flap_connection()
	}
}

func Test3UserListing(t *testing.T) {
	if ack.start_web_server == true {
		resp, err := http.Get("http://" + ack.web_server_address + "/api/v1/users")
		if err != nil {
			t.Error("Unable to get users: %v\n", err.Error())
			return
		}
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Error("Unable to read body: %v\n", err.Error())
			return
		}
		if len(buf) <= 0 {
			t.Error("Size of Users is < 0")
			return
		}
		if resp.StatusCode != http.StatusOK {
			t.Error("resp statusCode != OK: %v\n", resp.StatusCode)
			return
		}
	}
}

func Test4ChannelsAPIEndpoint(t *testing.T) {
	if ack.start_web_server == true {
		resp, err := http.Get("http://" + ack.web_server_address + "/api/v1/channels")
		if err != nil {
			t.Error("Unable to get channels: %v\n", err.Error())
			return
		}
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Error("Unable to read body: %v\n", err.Error())
			return
		}
		if len(buf) <= 0 {
			t.Error("Size of channels is < 0")
			return
		}
		if resp.StatusCode != http.StatusOK {
			t.Error("resp statusCode != OK: %v\n", resp.StatusCode)
			return
		}
	}
}

func Test5IMsAPIEndpoint(t *testing.T) {
	if ack.start_web_server == true {
		resp, err := http.Get("http://" + ack.web_server_address + "/api/v1/ims")
		if err != nil {
			t.Error("Unable to get ims: %v\n", err.Error())
			return
		}
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Error("Unable to read body: %v\n", err.Error())
			return
		}
		if len(buf) <= 0 {
			t.Error("Size of channels is < 0")
			return
		}
		if resp.StatusCode != http.StatusOK {
			t.Error("resp statusCode != OK: %v\n", resp.StatusCode)
			return
		}
	}
}

func Test6SigInt(t *testing.T) {
	init_complete_channel := make(chan bool, 1)
	interrupt_channel := make(chan os.Signal, 1)
	signal.Notify(interrupt_channel, os.Interrupt)
	go func() {
		for {
			init_complete_channel <- true
			select {
			case v := <-interrupt_channel:
				fmt.Printf("Got an interrupt on the interrupt_channel:%v\n", v)
				ack.Interrupt(v)
			case v := <-done_channel:
				fmt.Printf("Got an interrupt from the done_channel:%v\n", v)
			}
		}
	}()
	<-init_complete_channel
	interrupt_channel <- syscall.SIGINT
	<-done_channel
}

func testInit() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Ackley: Slack bot version %v\n", VERSION)
		flag.PrintDefaults()
	}
	ack_init := &AckleyInit{}
	flag.StringVar(&ack_init.Web_server_address, "ack_web_server_address", DEFAULT_ACKLEY_WEB_SERVER_ADDRESS, "URL Host/IP Web server should run on")
	flag.BoolVar(&ack_init.Start_web_server, "start_ack_web_server", true, "If true, starts up the web server")
	// Note: test_mode is set by default to be true
	flag.BoolVar(&ack_init.Test_mode, "test_mode", true, "If true, enables test mode.")
	flag.BoolVar(&ack_init.Message_retransmission_enabled, "message_retransmission_enabled", true, "If true, ackley will queue up and retransmit messages on message send failure")
	flag.IntVar(&ack_init.Message_retransmission_factor, "message_retransmission_factor", TEST_RETRANS_FACTOR, "How much time increases per retransmission")
	flag.IntVar(&ack_init.Message_retransmission_max_duration, "message_retransmission_max_duration", TEST_RETRANS_MAX_DURATION, "Maximum duration for retransmission before message is discarded")

	flag.StringVar(&slack_retransmission_response_channel, "slack_retransmission_response_channel", DEFAULT_SLACK_RESPONSE_CHANNEL, "Channel ackley will respond to on slack for retransmission test")
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

	ack = &Ackley{}
	ack.Init(ack_init)
}

func TestMain(m *testing.M) {
	testInit()
	init_complete_channel := make(chan bool, 1)
	go func() {
		wait_forever := make(chan bool, 1)
		ack.Start()
		init_complete_channel <- true
		<-wait_forever
	}()
	<-init_complete_channel

	os.Exit(m.Run())
}
