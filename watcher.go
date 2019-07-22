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
	"time"

	"github.com/casbin/casbin/persist"
	"github.com/casbin/casbin/util"
	"github.com/coreos/etcd/client"
)

type Watcher struct {
	endpoint string
	client client.Client
	kapi client.KeysAPI
	running bool
	callback func(string)
}

// finalizer is the destructor for Watcher.
func finalizer(w *Watcher) {
	w.running = false
}

// NewWatcher is the constructor for Watcher.
// endpoint is the endpoint for etcd clusters.
func NewWatcher(endpoint string) persist.Watcher {
	w := &Watcher{}
	w.endpoint = endpoint
	w.running = true
	w.callback = nil

	// Create the client.
	w.createClient()

	// Call the destructor when the object is released.
	runtime.SetFinalizer(w, finalizer)

	go w.startWatch()

	return w
}

func (w *Watcher) createClient() error {
	cfg := client.Config{
		Endpoints:               []string{w.endpoint},
		Transport:               client.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
	}

	c, err := client.New(cfg)
	if err != nil {
		return err
	}

	kapi := client.NewKeysAPI(c)
	w.client = c
	w.kapi = kapi
	return nil
}

// SetUpdateCallback sets the callback function that the watcher will call
// when the policy in DB has been changed by other instances.
// A classic callback is Enforcer.LoadPolicy().
func (w *Watcher) SetUpdateCallback(callback func(string)) error {
	w.callback = callback
	return nil
}

// Update calls the update callback of other instances to synchronize their policy.
// It is usually called after changing the policy in DB, like Enforcer.SavePolicy(),
// Enforcer.AddPolicy(), Enforcer.RemovePolicy(), etc.
func (w *Watcher) Update() error {
	rev := 0

	// Get "/casbin" key's value.
	resp, err := w.kapi.Get(context.Background(), "/casbin", nil)
	if err != nil {
		if err.(client.Error).Code != client.ErrorCodeKeyNotFound {
			return err
		}
	} else {
		rev, err = strconv.Atoi(resp.Node.Value)
		if err != nil {
			return err
		}
		util.LogPrint("Get revision: ", rev)
		rev += 1
	}

	newRev := strconv.Itoa(rev)

	// Set "/casbin" key with new revision value.
	util.LogPrint("Set revision: ", newRev)
	resp, err = w.kapi.Set(context.Background(), "/casbin", newRev, nil)
	return err
}

// startWatch is a goroutine that watches the policy change.
func (w *Watcher) startWatch() error {
	watcher := w.kapi.Watcher("/casbin", &client.WatcherOptions{Recursive: false})
	for {
		if !w.running {
			return nil
		}

		res, err := watcher.Next(context.Background())
		if err != nil {
			return err
			break
		}

		if res.Action == "set" || res.Action == "update" {
			if w.callback != nil {
				w.callback(res.Node.Value)
			}
		}
	}

	return nil
}
