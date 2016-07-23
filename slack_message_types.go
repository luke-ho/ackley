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

type SlackChannel struct {
	Id          string  `json:"id"`
	Name        string  `json:"name"`
	Is_channel  bool    `json:"is_channel"`
	Created     float64 `json:"created"`
	Creator     string  `json:"creator"`
	Is_archived bool    `json:"is_archived"`
	Is_general  bool    `json:"is_general"`
	Has_pins    bool    `json:"has_pins"`
	Is_member   bool    `json:"is_member"`
}

type SlackPing struct {
	Id   int64  `json:"id"`
	Type string `json:"type"`
	Time int64  `json:"time"`
}

type SlackPong struct {
	Reply_to uint64  `json:"reply_to"`
	Type     string  `json:"type"`
	Time     float64 `json:"time"`
}

type SlackTypingEvent struct {
	Id      int64  `json:"id"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
}

type SlackResponse struct {
	Ok bool `json:"ok"`
}

type SlackUser struct {
	Id      string `json:"id"`
	Team_id string `json:"team_id"`
	Name    string `json:"name"`
}

type SlackMessage struct {
	Type       string                   `json:"type"`
	Channel    string                   `json:"channel"`
	User       string                   `json:"user"`
	Text       string                   `json:"text"`
	Ts         *float64                 `json:"ts,omitempty"`
	Markdown   *bool                    `json:"mrkdwn,omitempty"`
	Attachment []SlackMessageAttachment `json:"attachments,omitempty"` // TBD: Look at when slack supports attachements in RTM
}

type SlackMessageAttachmentField struct {
	Title    string   `json:"title"`
	Value    string   `json:"value"`
	Short    bool     `json:"short"`
	Markdown []string `json:"mrkdwn_in"`
}

type SlackMessageAttachment struct {
	Fallback    string                        `json:"fallback"`
	Color       string                        `json:"color"`
	Pretext     string                        `json:"pretext"`
	Author_name string                        `json:"author_name,omitempty"`
	Author_link string                        `json:"author_link,omitempty"`
	Author_icon string                        `json:"author_icon,omitempty"`
	Title       string                        `json:"title"`
	Title_link  string                        `json:"title_link"`
	Text        string                        `json:"text"`
	Fields      []SlackMessageAttachmentField `json:fields"`
	Image_url   string                        `json:"image_url,omitempty"`
	Thumb_url   string                        `json:"thumb_url,omitempty"`
	Footer      string                        `json:"footer,omitempty"`
	Footer_icon string                        `json:"footer_icon,omitempty"`
	Ts          float64                       `json:"ts,omitempty"`
}
