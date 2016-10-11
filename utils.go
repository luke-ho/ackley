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
	"io/ioutil"
	"math/rand"
	"net/http"
	"reflect"
	"sync/atomic"
	"time"
)

func (ackley *Ackley) update_bot_presence() {
	// Update the presence of the bot itself
	presence_url := fmt.Sprintf("https://slack.com/api/users.setPresence?%v=%v&%v=%v", "token", ackley.slack_auth_token, "presence", "auto")
	resp, err := http.Get(presence_url)
	if err != nil {
		glog.Errorf("Error while trying to set presence:%v\n", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		glog.Errorf("Error while trying to set presence, status code not OK:%v\n", resp.StatusCode)
		return
	}
	pres_resp_buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("Error while trying to read from presence response body:%v\n", err)
		return
	}
	resp.Body.Close()
	slack_response := &SlackResponse{}
	err = json.Unmarshal(pres_resp_buf, slack_response)
	if err != nil {
		glog.Errorf("Error while trying to unmarshal slack response for presence:%v\n", err)
		return
	}
	if slack_response.Ok == false {
		glog.Errorf("Error while trying to set presence: slack response not ok\n")
		return
	}
}

func (ackley *Ackley) store_slack_channel_info(rtm_start_json map[string]interface{}) {
	glog.Infof("Type of channels:%v\n", reflect.TypeOf(rtm_start_json["channels"]))
	if cv, ok := rtm_start_json["channels"].([]interface{}); ok {
		glog.Infof("channels value:%v\n", cv)
		for _, entry := range cv {
			cur_slack_channel := &SlackChannel{}
			glog.Infof("Entry type: %v\n", reflect.TypeOf(entry))
			cur_map, err := entry.(map[string]interface{})
			if err == false {
				glog.Errorf("Cur_map error:%v\n", err)
				return
			}
			for k, v := range cur_map {
				switch k {
				case "id":
					cur_slack_channel.Id = v.(string)
				case "name":
					cur_slack_channel.Name = v.(string)
				case "is_channel":
					cur_slack_channel.Is_channel = v.(bool)
				case "created":
					cur_slack_channel.Created = v.(float64)
				case "creator":
					cur_slack_channel.Creator = v.(string)
				case "is_archived":
					cur_slack_channel.Is_archived = v.(bool)
				case "is_general":
					cur_slack_channel.Is_general = v.(bool)
				case "has_pins":
					cur_slack_channel.Has_pins = v.(bool)
				case "is_member":
					cur_slack_channel.Is_member = v.(bool)
				default:
					glog.Infof("Unknown attribute: %v = %v\n", k, v)
				}
			}
			ackley.slack_channels = append(ackley.slack_channels, *cur_slack_channel)
		}
	} else {
		glog.Errorf("Error while getting list of slack channels from rtm.Start: ok resp: %v\n", ok)
		return
	}
}

func (ackley *Ackley) store_slack_users(rtm_start_json map[string]interface{}) {
	// Populate users
	if cv, ok := rtm_start_json["users"].([]interface{}); ok {
		glog.Infof("users:%v\n", cv)
		for _, entry := range cv {
			cur_slack_user := &SlackUser{}
			glog.Infof("Entry type: %v\n", reflect.TypeOf(entry))
			cur_map, err := entry.(map[string]interface{})
			if err == false {
				glog.Errorf("Cur_map error:%v\n", err)
				return
			}
			for k, v := range cur_map {
				switch k {
				case "id":
					cur_slack_user.Id = v.(string)
				case "team_id":
					cur_slack_user.Team_id = v.(string)
				case "name":
					cur_slack_user.Name = v.(string)
				default:
					glog.Infof("Ignoring attribute: %v = %v\n", k, v)
				}
			}
			ackley.slack_users = append(ackley.slack_users, *cur_slack_user)
			ackley.slack_user_id_to_user[cur_slack_user.Id] = &ackley.slack_users[len(ackley.slack_users)-1]
		}
	} else {
		glog.Errorf("Error while getting list of slack channels from rtm.Start: ok resp: %v\n", ok)
		return
	}
}

func (ackley *Ackley) store_slack_ims_info(rtm_start_json map[string]interface{}) {
	// Populate users
	if cv, ok := rtm_start_json["ims"].([]interface{}); ok {
		glog.Infof("ims:%v\n", cv)
		for _, entry := range cv {
			cur_slack_im := &SlackIm{}
			glog.Infof("Entry type: %v\n", reflect.TypeOf(entry))
			cur_map, err := entry.(map[string]interface{})
			if err == false {
				glog.Errorf("Cur_map error:%v\n", err)
				return
			}
			for k, v := range cur_map {
				switch k {
				case "id":
					cur_slack_im.Id = v.(string)
				case "user":
					cur_slack_im.User_id = v.(string)
				default:
					glog.Infof("Ignoring attribute: %v = %v\n", k, v)
				}
			}
			ackley.slack_user_id_to_im[cur_slack_im.User_id] = cur_slack_im.Id
		}
	} else {
		glog.Errorf("Error while getting list of slack ims from rtm.Start: ok resp: %v\n", ok)
		return
	}
}

func (ackley *Ackley) listen_for_interrupts() {
	for {
		select {
		case <-ackley.interrupt_channel:
			ackley.cleanup()
			ackley.interrupt_channel_resp <- true
		}
	}
}

// TBD: Remove this
func (ackley *Ackley) randomly_disconnect() {
	rand_seconds := rand.Int31n(5)
	glog.Errorf("Randomly disconnect: About to sleep for %v seconds\n", rand_seconds)
	time.Sleep(time.Second * time.Duration(rand_seconds))
	glog.Errorf("Randomly disconnect: Begin flap\n")
	ackley.flap_connection()
}

func (ackley *Ackley) make_default_response(info *SlackMessageInfo) ([]byte, error) {
	text_response := fmt.Sprintf("Hi, %v!\n", info.User.Name)

	info.Msg.Text = text_response
	response_bytes, err := json.Marshal(info.Msg)
	if err != nil {
		return nil, err
	}
	return response_bytes, nil
}
func (ackley *Ackley) send_typing_event(channel string) {
	ste := SlackTypingEvent{Id: 1, Type: "typing", Channel: channel}
	ste_bytes, err := json.Marshal(&ste)
	if err != nil {
		glog.Errorf("Error while trying to Marshal typing event (%v):%v\n", ste, err.Error())
	}
	if atomic.LoadInt32(&ackley.cleaning) == 0 {
		_, err = ackley.slack_web_socket.Write(ste_bytes)
	} else {
		err = fmt.Errorf("Unable to send typing event as ackley is cleaning up.")
	}
	if err != nil {
		glog.Errorf("Error while writing to web socket(%v):%v\n", string(ste_bytes), err.Error())
	}
}
