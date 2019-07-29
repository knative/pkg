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

// pacers contains self-defined pacers that can be used in Attack of Vegeta (for performance testing)

package vegeta

import (
	"fmt"
	"math"
	"strings"
	"time"

	vegeta "github.com/tsenart/vegeta/lib"
)

// steadyUpPacer is a Pacer that describes attack request rates that increases in the beginning then becomes steady.
//  Max  |     ,----------------
//       |    /
//       |   /
//       |  /
//       | /
//  Min -+------------------------------> t
//       |<-Up->|
type steadyUpPacer struct {
	// upDuration is the duration that attack request rates increase from Min to Max, must be larger than 0.
	upDuration time.Duration
	// min is the attack request rates from the beginning, must be larger than 0.
	min vegeta.Rate
	// max is the maximum and final steady attack request rates, must be larger than Min.
	max vegeta.Rate

	slope        float64
	minHitsPerNs float64
	maxHitsPerNs float64
}

// NewSteadyUpPacer returns a new SteadyUpPacer with the given config.
func NewSteadyUpPacer(min vegeta.Rate, max vegeta.Rate, upDuration time.Duration) vegeta.Pacer {
	if upDuration <= 0 || min.Freq <= 0 || min.Per <= 0 || max.Freq <= 0 || max.Per <= 0 {
		panic("Configuration for this steadyUpPacer is invalid!")
	}
	minHitsPerNs := hitsPerNs(min)
	maxHitsPerNs := hitsPerNs(max)
	if minHitsPerNs >= maxHitsPerNs {
		panic("min rate must be smaller than max rate!")
	}

	pacer := &steadyUpPacer{
		min:          min,
		max:          max,
		upDuration:   upDuration,
		slope:        (maxHitsPerNs - minHitsPerNs) / float64(upDuration),
		minHitsPerNs: minHitsPerNs,
		maxHitsPerNs: maxHitsPerNs,
	}
	return pacer
}

// steadyUpPacer satisfies the Pacer interface.
var _ vegeta.Pacer = &steadyUpPacer{}

// String returns a pretty-printed description of the steadyUpPacer's behaviour.
func (sup *steadyUpPacer) String() string {
	return fmt.Sprintf("Up{%s + %s / %s}, then Steady{%s}", sup.min, sup.max, sup.upDuration, sup.max)
}

// Pace determines the length of time to sleep until the next hit is sent.
func (sup *steadyUpPacer) Pace(elapsedTime time.Duration, elapsedHits uint64) (time.Duration, bool) {
	expectedHits := sup.hits(elapsedTime)
	if elapsedHits < uint64(expectedHits) {
		// Running behind, send next hit immediately.
		return 0, false
	}

	// Re-arranging our hits equation to provide a duration given the number of
	// requests sent is non-trivial, so we must solve for the duration numerically.
	// math.Round() added here because we have to coerce to int64 nanoseconds
	// at some point and it corrects a bunch of off-by-one problems.
	nsPerHit := 1 / sup.hitsPerNs(elapsedTime)
	hitsToWait := float64(elapsedHits+1) - expectedHits
	nextHitIn := time.Duration(nsPerHit * hitsToWait)

	// If we can't converge to an error of <1e-3 within 10 iterations, bail.
	// This rarely even loops for any large Period if hitsToWait is small.
	for i := 0; i < 10; i++ {
		hitsAtGuess := sup.hits(elapsedTime + nextHitIn)
		err := float64(elapsedHits+1) - hitsAtGuess
		if math.Abs(err) < 1e-3 {
			return nextHitIn, false
		}
		nextHitIn = time.Duration(float64(nextHitIn) / (hitsAtGuess - float64(elapsedHits)))
	}

	return nextHitIn, false
}

// hits returns the number of expected hits for this pacer during the given time.
func (sup *steadyUpPacer) hits(t time.Duration) float64 {
	// If t is smaller than the upDuration, calculate the hits as a trapezoid.
	if t <= sup.upDuration {
		curtHitsPerNs := sup.hitsPerNs(t)
		return (curtHitsPerNs + sup.minHitsPerNs) / 2.0 * float64(t)
	}

	// If t is larger than the upDuration, calculate the hits as a trapezoid + a rectangle.
	upHits := (sup.maxHitsPerNs + sup.minHitsPerNs) / 2.0 * float64(sup.upDuration)
	steadyHits := sup.maxHitsPerNs * float64(t-sup.upDuration)
	return upHits + steadyHits
}

// hitsPerNs returns the attack rate for this pacer at a given time.
func (sup *steadyUpPacer) hitsPerNs(t time.Duration) float64 {
	if t <= sup.upDuration {
		return sup.minHitsPerNs + float64(t)*sup.slope
	}

	return sup.maxHitsPerNs
}

// hitsPerNs returns the attack rate this ConstantPacer represents, in
// fractional hits per nanosecond.
func hitsPerNs(cp vegeta.ConstantPacer) float64 {
	return float64(cp.Freq) / float64(cp.Per)
}

// combinedPacer is a pacer that combines multiple pacers and runs them sequentially when being used for attack.
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
		pacerStr := fmt.Sprintf("Pacer: %s, Duration: %s", cp.pacers[i], cp.durations[i])
		sb.WriteString(pacerStr)
	}
	return sb.String()
}

func (cp *combinedPacer) Pace(elapsedTime time.Duration, elapsedHits uint64) (time.Duration, bool) {
	pacerTimeOffset := uint64(elapsedTime) % cp.totalDuration
	numRounds := uint64(elapsedTime) / cp.totalDuration

	pacerIndex := cp.getPacerIndex(pacerTimeOffset)
	// The pacer must be the same or the next neighbor of the current pacer, otherwise stop the attack.
	if pacerIndex != cp.curtPacerIndex &&
		pacerIndex != cp.curtPacerIndex+1 &&
		!(pacerIndex == 0 && cp.curtPacerIndex == uint(len(cp.pacers)-1)) {
		return 0, true
	}

	// If it needs to switch to the next pacer, update prevElapsedTime, prevElapsedHits and curtPacerIndex.
	if pacerIndex != cp.curtPacerIndex {
		cp.prevElapsedTime = numRounds*cp.totalDuration + cp.stepDurations[cp.curtPacerIndex]
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
