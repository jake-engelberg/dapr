/*
Copyright 2021 The Dapr Authors
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

package actors

import (
	"errors"
	"sync"
	"sync/atomic"
)

var ErrMaxStackDepthExceeded = errors.New("maximum stack depth exceeded")

type ActorLock struct {
	requestLock   sync.Mutex
	activeRequest *string
	stackDepth    atomic.Int32
	maxStackDepth int32
	// lockChan is used instead of a sync.Mutex to enforce FIFO ordering of method execution.
	// We use a buffered channel to ensure that requests are processed
	// in the order they arrive, which sync.Mutex does not guarantee.
	lockChan chan struct{}
}

func NewActorLock(maxStackDepth int32) *ActorLock {
	return &ActorLock{
		lockChan:      make(chan struct{}, 1),
		maxStackDepth: maxStackDepth,
	}
}

func (a *ActorLock) Lock(requestID *string) error {
	currentRequest := a.getCurrentID()

	if a.stackDepth.Load() == a.maxStackDepth {
		return ErrMaxStackDepthExceeded
	}

	if currentRequest == nil || *currentRequest != *requestID {
		a.lockChan <- struct{}{}
		a.setCurrentID(requestID)
		a.stackDepth.Add(1)
	} else {
		a.stackDepth.Add(1)
	}

	return nil
}

func (a *ActorLock) Unlock() {
	a.stackDepth.Add(-1)
	if a.stackDepth.Load() == 0 {
		a.clearCurrentID()
		<-a.lockChan
	}
}

func (a *ActorLock) getCurrentID() *string {
	a.requestLock.Lock()
	defer a.requestLock.Unlock()

	return a.activeRequest
}

func (a *ActorLock) setCurrentID(id *string) {
	a.requestLock.Lock()
	defer a.requestLock.Unlock()

	a.activeRequest = id
}

func (a *ActorLock) clearCurrentID() {
	a.requestLock.Lock()
	defer a.requestLock.Unlock()

	a.activeRequest = nil
}
