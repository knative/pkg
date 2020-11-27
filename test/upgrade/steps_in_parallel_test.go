/*
Copyright 2020 The Knative Authors

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

package upgrade_test

import (
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"
	"knative.dev/pkg/test/upgrade"
)

func TestStepsCanRunInParallel(t *testing.T) {
	st := &state{
		numbers: nil,
		ready:   0,
	}
	s := upgrade.Suite{
		Tests: upgrade.Tests{
			PreUpgrade: []upgrade.Operation{
				op(1, st),
				op(2, st),
				op(3, st),
			},
		},
	}
	log, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal(err)
	}
	s.Execute(upgrade.Configuration{
		T:   t,
		Log: log,
	})

	largest := 0
	for _, number := range st.numbers {
		if number > largest {
			largest = number
			continue
		}
		if number < largest {
			// Run in parallel, numbers arent in order
			return
		}
	}
	t.Errorf("st.numbers are in order: %#v", st.numbers)
}

func op(num int, s *state) upgrade.Operation {
	return upgrade.NewOperation(fmt.Sprint("op", num), func(c upgrade.Context) {
		c.T.Parallel()
		s.ready++
		step := time.Millisecond * 2
		loop := func() {
			s.add(num)
			time.Sleep(step)
		}
		for s.ready < 3 {
			loop()
		}
		cooldown := time.Second
		noloops := int(cooldown.Milliseconds() / step.Milliseconds())
		for i := 0; i < noloops; i++ {
			loop()
		}
	})
}

type state struct {
	numbers []int
	ready   int
}

func (s *state) add(num int) {
	s.numbers = append(s.numbers, num)
}
