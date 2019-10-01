/*
Copyright 2018 The Kubernetes Authors.

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

package client

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	gerrit "github.com/andygrunwald/go-gerrit"
)

type fgc struct {
	instance string
	changes  map[string][]gerrit.ChangeInfo
}

func (f *fgc) QueryChanges(opt *gerrit.QueryChangeOptions) (*[]gerrit.ChangeInfo, *gerrit.Response, error) {
	changes := []gerrit.ChangeInfo{}

	changeInfos, ok := f.changes[f.instance]
	if !ok {
		return &changes, nil, nil
	}

	project := ""
	for _, query := range opt.Query {
		for _, q := range strings.Split(query, "+") {
			if strings.HasPrefix(q, "project:") {
				project = q[8:]
			}
		}
	}

	for idx, change := range changeInfos {
		if idx >= opt.Start && len(changes) <= opt.Limit {
			if project == change.Project {
				changes = append(changes, change)
			}
		}
	}

	return &changes, nil, nil
}

func (f *fgc) SetReview(changeID, revisionID string, input *gerrit.ReviewInput) (*gerrit.ReviewResult, *gerrit.Response, error) {
	return nil, nil, nil
}

func makeStamp(t time.Time) gerrit.Timestamp {
	return gerrit.Timestamp{Time: t}
}

func newStamp(t time.Time) *gerrit.Timestamp {
	gt := makeStamp(t)
	return &gt
}

func TestQueryChange(t *testing.T) {
	now := time.Now().UTC()

	var testcases = []struct {
		name       string
		lastUpdate time.Time
		changes    map[string][]gerrit.ChangeInfo
		revisions  map[string][]string
	}{
		{
			name:       "no changes",
			lastUpdate: now,
			revisions:  map[string][]string{},
		},
		{
			name:       "one outdated change",
			lastUpdate: now.Add(-time.Minute),
			changes: map[string][]gerrit.ChangeInfo{
				"foo": {
					{
						Project:         "bar",
						ID:              "1",
						CurrentRevision: "1-1",
						Updated:         makeStamp(now.Add(-time.Hour)),
						Revisions: map[string]gerrit.RevisionInfo{
							"1-1": {
								Created: makeStamp(now.Add(-time.Hour)),
							},
						},
						Status: "NEW",
					},
				},
			},
			revisions: map[string][]string{},
		},
		{
			name:       "one outdated change, but there's a new message",
			lastUpdate: now.Add(-time.Minute),
			changes: map[string][]gerrit.ChangeInfo{
				"foo": {
					{
						Project:         "bar",
						ID:              "100",
						CurrentRevision: "1-1",
						Updated:         makeStamp(now),
						Revisions: map[string]gerrit.RevisionInfo{
							"1-1": {
								Created: makeStamp(now.Add(-time.Hour)),
								Number:  1,
							},
						},
						Status: "NEW",
						Messages: []gerrit.ChangeMessageInfo{
							{
								Date:           makeStamp(now),
								Message:        "some message",
								RevisionNumber: 1,
							},
						},
					},
				},
			},
			revisions: map[string][]string{"foo": {"1-1"}},
		},
		{
			name:       "one up-to-date change",
			lastUpdate: now.Add(-time.Minute),
			changes: map[string][]gerrit.ChangeInfo{
				"foo": {
					{
						Project:         "bar",
						ID:              "1",
						CurrentRevision: "1-1",
						Updated:         makeStamp(now),
						Revisions: map[string]gerrit.RevisionInfo{
							"1-1": {
								Created: makeStamp(now),
							},
						},
						Status: "NEW",
					},
				},
			},
			revisions: map[string][]string{
				"foo": {"1-1"},
			},
		},
		{
			name:       "one up-to-date change, same timestamp",
			lastUpdate: now.Truncate(time.Second),
			changes: map[string][]gerrit.ChangeInfo{
				"foo": {
					{
						Project:         "bar",
						ID:              "1",
						CurrentRevision: "1-1",
						Updated:         makeStamp(now),
						Revisions: map[string]gerrit.RevisionInfo{
							"1-1": {
								Created: makeStamp(now),
							},
						},
						Status: "NEW",
					},
				},
			},
			revisions: map[string][]string{
				"foo": {"1-1"},
			},
		},
		{
			name:       "one up-to-date change but stale commit",
			lastUpdate: now.Add(-time.Minute),
			changes: map[string][]gerrit.ChangeInfo{
				"foo": {
					{
						Project:         "bar",
						ID:              "1",
						CurrentRevision: "1-1",
						Updated:         makeStamp(now),
						Revisions: map[string]gerrit.RevisionInfo{
							"1-1": {
								Created: makeStamp(now.Add(-time.Hour)),
							},
						},
						Status: "NEW",
					},
				},
			},
			revisions: map[string][]string{},
		},
		{
			name:       "one up-to-date change, wrong instance",
			lastUpdate: now.Add(-time.Minute),
			changes: map[string][]gerrit.ChangeInfo{
				"evil": {
					{
						Project:         "bar",
						ID:              "1",
						CurrentRevision: "1-1",
						Updated:         makeStamp(now),
						Revisions: map[string]gerrit.RevisionInfo{
							"1-1": {
								Created: makeStamp(now),
							},
						},
						Status: "NEW",
					},
				},
			},
			revisions: map[string][]string{},
		},
		{
			name:       "one up-to-date change, wrong project",
			lastUpdate: now.Add(-time.Minute),
			changes: map[string][]gerrit.ChangeInfo{
				"foo": {
					{
						Project:         "evil",
						ID:              "1",
						CurrentRevision: "1-1",
						Updated:         makeStamp(now),
						Revisions: map[string]gerrit.RevisionInfo{
							"1-1": {
								Created: makeStamp(now),
							},
						},
						Status: "NEW",
					},
				},
			},
			revisions: map[string][]string{},
		},
		{
			name:       "two up-to-date changes, two projects",
			lastUpdate: now.Add(-time.Minute),
			changes: map[string][]gerrit.ChangeInfo{
				"foo": {
					{
						Project:         "bar",
						ID:              "1",
						CurrentRevision: "1-1",
						Updated:         makeStamp(now),
						Revisions: map[string]gerrit.RevisionInfo{
							"1-1": {
								Created: makeStamp(now),
							},
						},
						Status: "NEW",
					},
					{
						Project:         "bar",
						ID:              "2",
						CurrentRevision: "2-1",
						Updated:         makeStamp(now),
						Revisions: map[string]gerrit.RevisionInfo{
							"2-1": {
								Created: makeStamp(now),
							},
						},
						Status: "NEW",
					},
				},
			},
			revisions: map[string][]string{
				"foo": {"1-1", "2-1"},
			},
		},
		{
			name:       "one good one bad",
			lastUpdate: now.Add(-time.Minute),
			changes: map[string][]gerrit.ChangeInfo{
				"foo": {
					{
						Project:         "bar",
						ID:              "1",
						CurrentRevision: "1-1",
						Updated:         makeStamp(now),
						Revisions: map[string]gerrit.RevisionInfo{
							"1-1": {
								Created: makeStamp(now),
							},
						},
						Status: "NEW",
					},
					{
						Project:         "bar",
						ID:              "2",
						CurrentRevision: "2-1",
						Updated:         makeStamp(now.Add(-time.Hour)),
						Revisions: map[string]gerrit.RevisionInfo{
							"2-1": {
								Created: makeStamp(now.Add(-time.Hour)),
							},
						},
						Status: "NEW",
					},
				},
			},
			revisions: map[string][]string{
				"foo": {"1-1"},
			},
		},
		{
			name:       "multiple up-to-date changes",
			lastUpdate: now.Add(-time.Minute),
			changes: map[string][]gerrit.ChangeInfo{
				"foo": {
					{
						Project:         "bar",
						ID:              "1",
						CurrentRevision: "1-1",
						Updated:         makeStamp(now),
						Revisions: map[string]gerrit.RevisionInfo{
							"1-1": {
								Created: makeStamp(now),
							},
						},
						Status: "NEW",
					},
					{
						Project:         "bar",
						ID:              "2",
						CurrentRevision: "2-1",
						Updated:         makeStamp(now),
						Revisions: map[string]gerrit.RevisionInfo{
							"2-1": {
								Created: makeStamp(now),
							},
						},
						Status: "NEW",
					},
				},
				"baz": {
					{
						Project:         "boo",
						ID:              "3",
						CurrentRevision: "3-2",
						Updated:         makeStamp(now),
						Revisions: map[string]gerrit.RevisionInfo{
							"3-2": {
								Created: makeStamp(now),
							},
							"3-1": {
								Created: makeStamp(now),
							},
						},
						Status: "NEW",
					},
					{
						Project:         "evil",
						ID:              "4",
						CurrentRevision: "4-1",
						Updated:         makeStamp(now.Add(-time.Hour)),
						Revisions: map[string]gerrit.RevisionInfo{
							"4-1": {
								Created: makeStamp(now.Add(-time.Hour)),
							},
						},
						Status: "NEW",
					},
				},
			},
			revisions: map[string][]string{
				"foo": {"1-1", "2-1"},
				"baz": {"3-2"},
			},
		},
		{
			name:       "one up-to-date merged change",
			lastUpdate: now.Add(-time.Minute),
			changes: map[string][]gerrit.ChangeInfo{
				"foo": {
					{
						Project:         "bar",
						ID:              "1",
						CurrentRevision: "1-1",
						Updated:         makeStamp(now),
						Submitted:       newStamp(now),
						Status:          "MERGED",
					},
				},
			},
			revisions: map[string][]string{
				"foo": {"1-1"},
			},
		},
		{
			name:       "one up-to-date abandoned change",
			lastUpdate: now.Add(-time.Minute),
			changes: map[string][]gerrit.ChangeInfo{
				"foo": {
					{
						Project:         "bar",
						ID:              "1",
						CurrentRevision: "1-1",
						Updated:         makeStamp(now),
						Submitted:       newStamp(now),
						Status:          "ABANDONED",
					},
				},
			},
			revisions: map[string][]string{},
		},
		{
			name:       "merged change recently updated but submitted before last update",
			lastUpdate: now.Add(-time.Minute),
			changes: map[string][]gerrit.ChangeInfo{
				"foo": {
					{
						Project:         "bar",
						ID:              "1",
						CurrentRevision: "1-1",
						Updated:         makeStamp(now),
						Submitted:       newStamp(now.Add(-2 * time.Minute)),
						Status:          "MERGED",
					},
				},
			},
			revisions: map[string][]string{},
		},
		{
			name:       "one abandoned, one merged",
			lastUpdate: now.Add(-time.Minute),
			changes: map[string][]gerrit.ChangeInfo{
				"foo": {
					{
						Project:         "bar",
						ID:              "1",
						CurrentRevision: "1-1",
						Updated:         makeStamp(now),
						Status:          "ABANDONED",
					},
					{
						Project:         "bar",
						ID:              "2",
						CurrentRevision: "2-1",
						Updated:         makeStamp(now),
						Submitted:       newStamp(now),
						Status:          "MERGED",
					},
				},
			},
			revisions: map[string][]string{
				"foo": {"2-1"},
			},
		},
		{
			name:       "merged change with new message, should ignore",
			lastUpdate: now.Add(-time.Minute),
			changes: map[string][]gerrit.ChangeInfo{
				"foo": {
					{
						Project:         "bar",
						ID:              "2",
						CurrentRevision: "2-1",
						Updated:         makeStamp(now),
						Submitted:       newStamp(now.Add(-time.Hour)),
						Status:          "MERGED",
						Messages: []gerrit.ChangeMessageInfo{
							{
								Date:           makeStamp(now),
								Message:        "some message",
								RevisionNumber: 1,
							},
						},
					},
				},
			},
			revisions: map[string][]string{},
		},
	}

	for _, tc := range testcases {
		client := &Client{
			handlers: map[string]*gerritInstanceHandler{
				"foo": {
					instance: "foo",
					projects: []string{"bar"},
					changeService: &fgc{
						changes:  tc.changes,
						instance: "foo",
					},
				},
				"baz": {
					instance: "baz",
					projects: []string{"boo"},
					changeService: &fgc{
						changes:  tc.changes,
						instance: "baz",
					},
				},
			},
		}

		changes := client.QueryChanges(tc.lastUpdate, 5)

		revisions := map[string][]string{}
		for instance, changes := range changes {
			revisions[instance] = []string{}
			for _, change := range changes {
				revisions[instance] = append(revisions[instance], change.CurrentRevision)
			}
			sort.Strings(revisions[instance])
		}

		if !reflect.DeepEqual(revisions, tc.revisions) {
			t.Errorf("tc %s - wrong revisions: got %#v, expect %#v", tc.name, revisions, tc.revisions)
		}
	}
}
