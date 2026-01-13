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
	"context"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/casbin/casbin/v2/persist"
	client "go.etcd.io/etcd/client/v3"
)

type Watcher struct {
	// lock for callback
	lock        sync.RWMutex
	endpoints   []string
	client      *client.Client
	running     bool
	callback    func(string)
	keyName     string
	password    string
	lastSentRev int64
	conf        *WatcherConfig
}

type WatcherConfig struct {
	Hosts                []string
	Key                  string `json:",default=casbin_watcher"`
	User                 string
	Pass                 string
	DialKeepAliveTimeout time.Duration `json:",default=10"`
	DialTimeout          time.Duration `json:",default=30"`
}

// finalizer is the destructor for Watcher.
func finalizer(w *Watcher) {
	w.running = false
}

// NewWatcher is the constructor for Watcher.
// endpoints is the endpoints for etcd clusters.
func NewWatcher(endpoints []string, keyName string, password ...string) (persist.Watcher, error) {
	w := &Watcher{}
	w.endpoints = endpoints
	w.running = true
	w.callback = nil
	w.keyName = keyName
	if len(password) > 0 {
		w.password = password[0]
	}

	// Create the client.
	err := w.createClient()
	if err != nil {
		return nil, err
	}

	// Call the destructor when the object is released.
	runtime.SetFinalizer(w, finalizer)

	go func() {
		_ = w.startWatch()
	}()

	return w, nil
}

// NewWatcherWithConfig is a configurable Watcher constructor
func NewWatcherWithConfig(config WatcherConfig) (persist.Watcher, error) {
	w := &Watcher{}
	w.running = true
	w.callback = nil
	w.keyName = config.Key
	w.conf = &config

	// Create the client.
	err := w.createClient()
	if err != nil {
		return nil, err
	}

	// Call the destructor when the object is released.
	runtime.SetFinalizer(w, finalizer)

	go func() {
		_ = w.startWatch()
	}()

	return w, nil
}

// Close closes the Watcher.
func (w *Watcher) Close() {
	finalizer(w)
}

func (w *Watcher) createClient() error {
	cfg := client.Config{
		Endpoints: w.endpoints,
		// set timeout per request to fail fast when the target endpoints is unavailable
		DialKeepAliveTimeout: time.Second * 10,
		DialTimeout:          time.Second * 30,
		Password:             w.password,
	}

	if w.conf != nil {
		cfg = client.Config{
			Endpoints: w.conf.Hosts,
			// set timeout per request to fail fast when the target endpoints is unavailable
			DialTimeout:          time.Second * w.conf.DialTimeout,
			DialKeepAliveTimeout: time.Second * w.conf.DialKeepAliveTimeout,
			Username:             w.conf.User,
			Password:             w.conf.Pass,
		}
	}

	c, err := client.New(cfg)
	if err != nil {
		return err
	}
	w.client = c
	return nil
}

// SetUpdateCallback sets the callback function that the watcher will call
// when the policy in DB has been changed by other instances.
// A classic callback is Enforcer.LoadPolicy().
func (w *Watcher) SetUpdateCallback(callback func(string)) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.callback = callback
	return nil
}

// Update calls the update callback of other instances to synchronize their policy.
// It is usually called after changing the policy in DB, like Enforcer.SavePolicy(),
// Enforcer.AddPolicy(), Enforcer.RemovePolicy(), etc.
func (w *Watcher) Update() error {
	w.lock.Lock()
	defer w.lock.Unlock()
	resp, err := w.client.Put(context.TODO(), w.keyName, "")
	if err == nil {
		w.lastSentRev = resp.Header.GetRevision()
	}
	return err
}

// startWatch is a goroutine that watches the policy change.
func (w *Watcher) startWatch() error {
	watcher := w.client.Watch(context.Background(), w.keyName)
	for res := range watcher {
		// Skip progress notifications
		if res.IsProgressNotify() {
			continue
		}
		// Skip empty events
		if len(res.Events) == 0 {
			continue
		}
		t := res.Events[0]
		if t.IsCreate() || t.IsModify() {
			w.lock.RLock()
			//ignore self update
			if rev := t.Kv.ModRevision; rev > w.lastSentRev && w.callback != nil {
				w.callback(strconv.FormatInt(rev, 10))
			}
			w.lock.RUnlock()
		}
	}
	return nil
}
