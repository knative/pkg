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
	"fmt"
	"sync"
	"time"

	"knative.dev/pkg/test/mako/alerter/shared"
	"knative.dev/pkg/test/slackutil"
)

const (
	// do not send alert on the same channel within 24 hours
	minInterval     = 24 * time.Hour
	messageTemplate = `
As of %s, there is a new performance regression detected from automation test:
%s`
)

// MessageHandler handles methods for slack messages
type MessageHandler struct {
	readClient  slackutil.ReadOperations
	writeClient slackutil.WriteOperations
	config      repoConfig
	dryrun      bool
}

// Setup creates the necessary setup to make calls to work with slack
func Setup(userName, readTokenPath, writeTokenPath, repo string, dryrun bool) (*MessageHandler, error) {
	readClient, err := slackutil.NewReadClient(userName, readTokenPath)
	if err != nil {
		return nil, fmt.Errorf("cannot authenticate to slack read client: %v", err)
	}
	writeClient, err := slackutil.NewWriteClient(userName, writeTokenPath)
	if err != nil {
		return nil, fmt.Errorf("cannot authenticate to slack write client: %v", err)
	}
	var config *repoConfig
	for _, repoConfig := range repoConfigs {
		if repoConfig.repo == repo {
			config = &repoConfig
			break
		}
	}
	if config == nil {
		return nil, fmt.Errorf("no channel configuration found for repo %v", repo)
	}
	return &MessageHandler{
		readClient:  readClient,
		writeClient: writeClient,
		config:      *config,
		dryrun:      dryrun,
	}, nil
}

// SendAlert will send the alert text to the slack channel(s)
func (smh *MessageHandler) SendAlert(text string) error {
	errs := make([]error, 0)
	channels := smh.config.channels
	mux := &sync.Mutex{}
	var wg sync.WaitGroup
	for i := range channels {
		channel := channels[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			startTime := time.Now().Add(-1 * minInterval)
			messageHistory, err := smh.readClient.MessageHistory(channel.identity, startTime)
			// do not send message again if messages were sent on the same channel a while ago
			if err == nil && messageHistory != nil && len(messageHistory) != 0 {
				return
			}

			message := fmt.Sprintf(messageTemplate, time.Now(), text)
			if err := smh.writeClient.Post(message, channel.identity); err != nil {
				mux.Lock()
				errs = append(errs, fmt.Errorf("failed to send message to channel %v", channel))
				mux.Unlock()
			}
		}()
	}
	wg.Wait()

	return shared.CombineErrors(errs)
}
