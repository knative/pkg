/*
 * Copyright 2020 The Knative Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package upgrade

import (
	"go.uber.org/zap"
)

func (s *Suite) Execute(c Configuration) {
	c.T.Error("not yet implemented")
}

// NewOperation creates a new upgrade operation or test
func NewOperation(name string, handler func(c Context)) Operation {
	return &operationHolder{name: name, handler: handler}
}

// NewStoppableOperation creates a new upgrade operation or test that can be
// notified to stop operating
func NewStoppableOperation(name string, handler func(c StoppableContext)) StoppableOperation {
	return &stoppableOperationHolder{name: name, handler: handler}
}

func (c Configuration) logger() *zap.SugaredLogger {
	return c.Log.Sugar()
}

type operationHolder struct {
	name    string
	handler func(c Context)
}

type stoppableOperationHolder struct {
	name    string
	handler func(c StoppableContext)
}

func (h *operationHolder) Name() string {
	return h.name
}

func (h *operationHolder) Handler() func(c Context) {
	return h.handler
}

func (s *stoppableOperationHolder) Name() string {
	return s.name
}

func (s *stoppableOperationHolder) Handler() func(c StoppableContext) {
	return s.handler
}
