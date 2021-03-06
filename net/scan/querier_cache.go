// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a GPL
// license that can be found in the LICENSE file.

// Package scan is the scan service
package scan

import (
	"errors"
	"github.com/rafaeljusto/shelter/model"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Maximum number of timeouts in a host before we start setting every query from this
	// host as timeout without checking it
	maxTimeoutsPerHost = 500
)

var (
	// Global variable used by all queriers (go routines) to access the cache
	querierCache QuerierCache

	// Maximum number of queries per second that a host will receive
	MaxQPSPerHost = uint64(500)

	// Error to identify a nameserver that had too many timeouts and is probably down
	ErrHostTimeout = errors.New("Nameserver down after too many timeouts detected")

	// Error to alert about too many queries only for one host. If we didn't have this, the
	// server could be added to a rate limit algorithm that could timeout all other queries
	ErrHostQPSExceeded = errors.New("Maximum number of queries per second for this host")
)

func init() {
	querierCache = QuerierCache{
		hosts: make(map[string]*hostCache),
	}
}

// NameserverCache was created to store beyond the addresses, a counter of how many times
// this host got timeout. For hosts with many timeouts we assume that their are down and
// avoid making queries whitout necessity. We also control the number of queries per
// second to avoid rate limit algorithms
type hostCache struct {
	addresses        []net.IP // nameserver's addresses
	lastEpoch        int64    // last query epoch
	queriesPerSecond uint64   // number of queries per second (epoch)
	timeouts         uint64   // counter that detects if this nameserver is down
}

// Method to detect if the number of timeouts in a host was exceeded
func (h hostCache) timeoutsPerHostExceeded() bool {
	return h.timeouts > maxTimeoutsPerHost
}

// Mehtod to check if the number of queries per second on this host was exceeded
func (h hostCache) queriesPerSecondExceeded() bool {
	// If the parameter that indicates if the host has to many requests is zero, we assume that the
	// user wants to turn off this feature
	if MaxQPSPerHost == 0 {
		return false
	}

	return time.Now().Unix() == h.lastEpoch && h.queriesPerSecond > MaxQPSPerHost
}

// QuerierCache was created to make the name resolution faster. Many domains use ISP the
// same host, so if we cache the hosts addresses we are speeding up many domains scans
type QuerierCache struct {
	hosts      map[string]*hostCache // key-value structure that store nameserver data
	hostsMutex sync.RWMutex          // Lock to allow concurrent access
}

// Method used to retrieve addresses of a given nameserver, if the address does not exist
// in the local cache the system will lookup for the domain and will store the result
func (q *QuerierCache) Get(nameserver model.Nameserver, fqdn string) ([]net.IP, error) {
	q.hostsMutex.RLock()
	host, found := q.hosts[nameserver.Host]
	q.hostsMutex.RUnlock()

	if found {
		if host.timeoutsPerHostExceeded() {
			return nil, ErrHostTimeout

		} else if host.queriesPerSecondExceeded() {
			return nil, ErrHostQPSExceeded

		} else {
			return host.addresses, nil
		}
	}

	// Not found in cache, lets discover the address of this name sending DNS requests or
	// retrieving from the namserver object (glue record)
	var addresses []net.IP

	if nameserver.NeedsGlue(fqdn) {
		if nameserver.IPv4 != nil {
			addresses = append(addresses, nameserver.IPv4)
		}

		if nameserver.IPv6 != nil {
			addresses = append(addresses, nameserver.IPv6)
		}
	}

	// In case that the nameserver doesn't have a glue record we try to resolve the hostname
	if len(addresses) == 0 {
		var err error
		addresses, err = net.LookupIP(nameserver.Host)
		if err != nil {
			return nil, err
		}
	}

	q.hostsMutex.Lock()
	q.hosts[nameserver.Host] = &hostCache{
		addresses:        addresses,
		lastEpoch:        0,
		queriesPerSecond: 0,
		timeouts:         0,
	}
	q.hostsMutex.Unlock()

	return addresses, nil
}

// Method used to notify when a host got timeout for a query, after a special number of
// timeouts we assume that every nameserver that use this host will get timeout status
func (q *QuerierCache) Timeout(name string) {
	q.hostsMutex.RLock()
	host, found := q.hosts[name]
	q.hostsMutex.RUnlock()

	if found {
		atomic.AddUint64(&host.timeouts, 1)
	}
}

// Method used to notify when a new query was made to a host. This is used to control the
// maximum number of queries sent to a host, avoiding rate limit startegies
func (q *QuerierCache) Query(name string) {
	q.hostsMutex.RLock()
	host, found := q.hosts[name]
	q.hostsMutex.RUnlock()

	if !found {
		return
	}

	now := time.Now().Unix()

	if now == host.lastEpoch {
		atomic.AddUint64(&host.queriesPerSecond, 1)

	} else if now > host.lastEpoch {
		if atomic.CompareAndSwapInt64(&host.lastEpoch, host.lastEpoch, now) {
			atomic.StoreUint64(&host.queriesPerSecond, 1)
		}
	}
}

// Clear cache. This method is for now used in integration test scenarios to get more
// realistic results in performance reports
func (q *QuerierCache) Clear() {
	q.hostsMutex.Lock()
	q.hosts = make(map[string]*hostCache)
	q.hostsMutex.Unlock()
}
