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
	"fmt"
	"strings"
	"time"

	vegeta "github.com/tsenart/vegeta/lib"
)

// combinedPacer is a Pacer that combines multiple Pacers and runs them sequentially when being used for attack.
type combinedPacer struct {
	// pacers is a list of pacers that will be used sequentially for attack, must be more than 1 pacer.
	pacers []vegeta.Pacer
	// durations is the list of durations for the given Pacers, must have the same length as Pacers.
	durations []time.Duration

	// totalDuration is sum of the given Durations.
	totalDuration uint64
	// stepDurations is the accumulative duration of each step calculated from the given Durations.
	stepDurations []uint64

	curtPacerIndex  uint
	prevElapsedHits uint64
	prevElapsedTime uint64
}

// NewCombinedPacer returns a new CombinedPacer with the given config.
func NewCombinedPacer(pacers []vegeta.Pacer, durations []time.Duration) vegeta.Pacer {
	if len(pacers) == 0 || len(durations) == 0 || len(pacers) != len(durations) || len(pacers) == 1 {
		panic("Configuration for this CombinedPacer is invalid!")
	}

	var totalDuration uint64
	stepDurations := make([]uint64, len(pacers))
	for i, duration := range durations {
		totalDuration += uint64(duration)
		if i == 0 {
			stepDurations[i] = uint64(duration)
		} else {
			stepDurations[i] = stepDurations[i-1] + uint64(duration)
		}
	}
	pacer := &combinedPacer{
		pacers:    pacers,
		durations: durations,

		totalDuration: totalDuration,
		stepDurations: stepDurations,

		curtPacerIndex:  0,
		prevElapsedHits: 0,
		prevElapsedTime: 0,
	}
	return pacer
}

// combinedPacer satisfies the Pacer interface.
var _ vegeta.Pacer = &combinedPacer{}

// String returns a pretty-printed description of the combinedPacer's behaviour.
func (cp *combinedPacer) String() string {
	var sb strings.Builder
	for i := range cp.pacers {
		pacerStr := fmt.Sprintf("Pacer: %s, Duration: %s\n", cp.pacers[i], cp.durations[i])
		sb.WriteString(pacerStr)
	}
	return sb.String()
}

func (cp *combinedPacer) Pace(elapsedTime time.Duration, elapsedHits uint64) (time.Duration, bool) {
	pacerTimeOffset := uint64(elapsedTime) % cp.totalDuration
	pacerIndex := cp.getPacerIndex(pacerTimeOffset)
	// The pacer must be the same or the next neighbor of the current pacer, otherwise stop the attack.
	if pacerIndex != cp.curtPacerIndex &&
		pacerIndex != cp.curtPacerIndex+1 &&
		!(pacerIndex == 0 && cp.curtPacerIndex == uint(len(cp.pacers)-1)) {
		return 0, true
	}

	// If it needs to switch to the next pacer, update prevElapsedTime, prevElapsedHits and curtPacerIndex.
	if pacerIndex != cp.curtPacerIndex {
		cp.prevElapsedTime = uint64(elapsedTime)
		cp.prevElapsedHits = elapsedHits
		cp.curtPacerIndex = pacerIndex
	}

	// Use the adjusted elapsedTime and elapsedHits to get the time to wait for the next hit.
	curtPacer := cp.pacers[cp.curtPacerIndex]
	curtElapsedTime := time.Duration(uint64(elapsedTime) - cp.prevElapsedTime)
	curtElapsedHits := elapsedHits - cp.prevElapsedHits
	return curtPacer.Pace(curtElapsedTime, curtElapsedHits)
}

// getPacerIndex returns the index of pacer that pacerTimeOffset falls into
func (cp *combinedPacer) getPacerIndex(pacerTimeOffset uint64) uint {
	for i, stepDuration := range cp.stepDurations {
		if pacerTimeOffset < stepDuration {
			return uint(i)
		}
	}
	return 0
}
