/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package slack

import (
	"os"
	"testing"
	"time"

	"knative.dev/pkg/test/slackutil/fakeslackutil"
)

var mh MessageHandler

func TestMain(m *testing.M) {
	client := fakeslackutil.NewFakeSlackClient()
	mh = MessageHandler{
		readClient:  client,
		writeClient: client,
		config: repoConfig{
			repo: "test_repo",
			channels: []channel{
				{name: "test_channel1", identity: "fsfdsf"},
				{name: "test_channel2", identity: "fdsfhfdh"},
			},
		},
		dryrun: false,
	}
	os.Exit(m.Run())
}

func TestMessaging(t *testing.T) {
	firstMsg := "first message"
	if err := mh.SendAlert(firstMsg); err != nil {
		t.Fatalf("expected to send the message, but failed: %v", err)
	}
	for _, channel := range mh.config.channels {
		history, err := mh.readClient.MessageHistory(channel.identity, time.Now().Add(-1*time.Hour))
		if err != nil {
			t.Fatalf("expected to get the message history, but failed: %v", err)
		}
		if len(history) != 1 {
			t.Fatalf("the message is expected to be successfully sent, but failed: %v", err)
		}
	}

	secondMsg := "second message"
	if err := mh.SendAlert(secondMsg); err != nil {
		t.Fatalf("expected to send the message, but failed: %v", err)
	}
	for _, channel := range mh.config.channels {
		history, err := mh.readClient.MessageHistory(channel.identity, time.Now().Add(-1*time.Hour))
		if err != nil {
			t.Fatalf("expected to get the message history, but failed: %v", err)
		}
		if len(history) != 1 {
			t.Fatalf("the message history is expected to still be 1, but now it's: %d", len(history))
		}
	}
}
