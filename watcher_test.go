// Copyright 2017 The casbin Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package etcdwatcher

import (
	"log"
	"testing"

	"github.com/casbin/casbin"
)

func updateCallback(rev string) {
	log.Println("New revision detected:", rev)
}

func TestWatcher(t *testing.T) {
	// updater represents the Casbin enforcer instance that changes the policy in DB.
	// Use the endpoints of etcd cluster as parameter.
	updater, _ := NewWatcher([]string{"http://127.0.0.1:2379"}, "/casbin")

	// listener represents any other Casbin enforcer instance that watches the change of policy in DB.
	listener, _ := NewWatcher([]string{"http://127.0.0.1:2379"}, "/casbin")
	// listener should set a callback that gets called when policy changes.
	listener.SetUpdateCallback(updateCallback)

	// updater changes the policy, and sends the notifications.
	err := updater.Update()
	if err != nil {
		panic(err)
	}

	// Now the listener's callback updateCallback() should be called,
	// because it receives the notification of policy update.
	// You should see "[New revision detected: X]" in the log.
}

func TestWithEnforcer(t *testing.T) {
	// Initialize the watcher.
	// Use the endpoints of etcd cluster as parameter.
	w, _ := NewWatcher([]string{"http://127.0.0.1:2379"}, "/casbin")

	// Initialize the enforcer.
	e, _ := casbin.NewEnforcer("examples/rbac_model.conf", "examples/rbac_policy.csv")

	// Set the watcher for the enforcer.
	e.SetWatcher(w)

	// By default, the watcher's callback is automatically set to the
	// enforcer's LoadPolicy() in the SetWatcher() call.
	// We can change it by explicitly setting a callback.
	w.SetUpdateCallback(updateCallback)

	// Update the policy to test the effect.
	// You should see "[New revision detected: X]" in the log.
	e.SavePolicy()
}
