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

package pacers

import (
	"testing"
	"time"

	vegeta "github.com/tsenart/vegeta/lib"
)

func TestCombinedPacer(t *testing.T) {
	pacer1 := vegeta.Rate{Freq: 1, Per: time.Second}
	pacer2 := vegeta.Rate{Freq: 5, Per: time.Second}
	pacer := NewCombinedPacer([]vegeta.Pacer{pacer1, pacer2}, []time.Duration{5 * time.Second, 5 * time.Second})

	for _, tt := range []struct {
		name            string
		elapsedTime     time.Duration
		elapsedHits     uint64
		expectedNextHit time.Duration
		expectedStop    bool
	}{{
		name:            "test the first hit",
		elapsedTime:     0 * time.Second,
		elapsedHits:     0,
		expectedNextHit: 1 * time.Second,
	}, {
		name:            "test the switch pacer hit",
		elapsedTime:     5 * time.Second,
		elapsedHits:     5,
		expectedNextHit: 200 * time.Millisecond,
	}, {
		name:            "test the loop back pacer hit",
		elapsedTime:     10 * time.Second,
		elapsedHits:     30,
		expectedNextHit: 1 * time.Second,
	}, {
		name:            "test the catch up hit",
		elapsedTime:     11 * time.Second,
		elapsedHits:     30,
		expectedNextHit: 0,
	}} {
		t.Run(tt.name, func(t *testing.T) {
			nextHit, _ := pacer.Pace(tt.elapsedTime, tt.elapsedHits)
			if nextHit != tt.expectedNextHit {
				t.Errorf("expected next hit %v, got %v", tt.expectedNextHit, nextHit)
			}
		})
	}
}
