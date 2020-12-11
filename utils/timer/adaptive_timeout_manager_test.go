// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package timer

import (
	"sync"
	"testing"
	"time"

	"github.com/liraxapp/avalanchego/ids"
	"github.com/liraxapp/avalanchego/utils/constants"
	"github.com/prometheus/client_golang/prometheus"
)

func TestAdaptiveTimeoutManager(t *testing.T) {
	tm := AdaptiveTimeoutManager{}
	err := tm.Initialize(&AdaptiveTimeoutConfig{
		InitialTimeout: time.Millisecond,
		MinimumTimeout: time.Millisecond,
		MaximumTimeout: time.Hour,
		TimeoutInc:     2 * time.Millisecond,
		TimeoutDec:     time.Microsecond,
		Namespace:      constants.PlatformName,
		Registerer:     prometheus.NewRegistry(),
	})
	if err != nil {
		t.Fatal(err)
	}
	go tm.Dispatch()

	var lock sync.Mutex

	numSuccessful := 5

	wg := sync.WaitGroup{}
	wg.Add(numSuccessful)

	callback := new(func())
	*callback = func() {
		lock.Lock()
		defer lock.Unlock()

		numSuccessful--
		if numSuccessful > 0 {
			tm.Put(ids.ID{byte(numSuccessful)}, *callback)
		}
		if numSuccessful >= 0 {
			wg.Done()
		}
		if numSuccessful%2 == 0 {
			tm.Remove(ids.ID{byte(numSuccessful)})
			tm.Put(ids.ID{byte(numSuccessful)}, *callback)
		}
	}
	(*callback)()
	(*callback)()

	wg.Wait()
}
