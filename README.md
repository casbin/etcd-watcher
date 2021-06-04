Etcd Watcher [![Go](https://github.com/casbin/etcd-watcher/actions/workflows/test.yml/badge.svg)](https://github.com/casbin/etcd-watcher/actions/workflows/test.yml) [![Coverage Status](https://coveralls.io/repos/github/casbin/etcd-watcher/badge.svg?branch=master)](https://coveralls.io/github/casbin/etcd-watcher?branch=master) [![Godoc](https://godoc.org/github.com/casbin/etcd-watcher?status.svg)](https://godoc.org/github.com/casbin/etcd-watcher)
====

Etcd Watcher is the [Etcd](https://github.com/coreos/etcd) watcher for [Casbin](https://github.com/casbin/casbin). With this library, Casbin can synchronize the policy with the database in multiple enforcer instances.

## Installation

    go get github.com/casbin/etcd-watcher

## Simple Example

```go
package main

import (
    "log"

    "github.com/casbin/casbin"
    "github.com/casbin/etcd-watcher"
)

func updateCallback(rev string) {
    log.Println("New revision detected:", rev)
}

func main() {
    // Initialize the watcher.
    // Use the endpoint of etcd cluster as parameter.
    w, _ := etcdwatcher.NewWatcher("http://127.0.0.1:2379")
    
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
```

## Getting Help

- [Casbin](https://github.com/casbin/casbin)

## License

This project is under Apache 2.0 License. See the [LICENSE](LICENSE) file for the full license text.
