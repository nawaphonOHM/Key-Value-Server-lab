package kv_server_lock_mechanism

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

const (
	NACQUIRE = 10
	NCLNT    = 10
	NSEC     = 2
)

func TestOneClientReliable(t *testing.T) {
	runClients(t, 1, true)
}

func TestManyClientsReliable(t *testing.T) {
	runClients(t, NCLNT, true)
}

// Run test clients
func runClients(t *testing.T, nclnt int, reliable bool) {
	ts := MakeTestKV(t, reliable)
	defer ts.Cleanup()

	ts.Begin(fmt.Sprintf("Test: %d lock clients", nclnt))

	ts.SpawnClientsAndWait(nclnt, NSEC*time.Second, func(me int, myck IKVClerk, done chan struct{}) ClntRes {
		return oneClient(t, me, myck, done)
	})
}

func oneClient(t *testing.T, me int, ck IKVClerk, done chan struct{}) ClntRes {
	lk := MakeLock(ck, "l")
	ck.Put("l0", "", 0)
	for i := 1; true; i++ {
		select {
		case <-done:
			return ClntRes{i, 0}
		default:
			lk.Acquire()

			// log.Printf("%d: acquired lock", me)

			b := strconv.Itoa(me)
			val, ver, err := ck.Get("l0")
			if err == OK {
				if val != "" {
					t.Fatalf("%d: two clients acquired lock %v", me, val)
				}
			} else {
				t.Fatalf("%d: get failed %v", me, err)
			}

			err = ck.Put("l0", string(b), ver)
			if !(err == OK || err == ErrMaybe) {
				t.Fatalf("%d: put failed %v", me, err)
			}

			time.Sleep(10 * time.Millisecond)

			err = ck.Put("l0", "", ver+1)
			if !(err == OK || err == ErrMaybe) {
				t.Fatalf("%d: put failed %v", me, err)
			}

			// log.Printf("%d: release lock", me)

			lk.Release()
		}
	}
	return ClntRes{}
}
