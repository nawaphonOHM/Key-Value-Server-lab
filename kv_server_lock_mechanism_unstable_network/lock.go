package kv_server_lock_mechanism_unstable_network

import (
	"fmt"
	"log"
	"time"
)

type Lock struct {
	// IKVClerk is a go interface for k/v clerks: the interface hides
	// the specific Clerk type of ck but promises that ck supports
	// Put and Get.  The tester passes the clerk in when calling
	// MakeLock().
	ck IKVClerk
	// You may add code here
	lockId string

	lockStateKey string
}

// The tester calls MakeLock() and passes in a k/v clerk; your code can
// perform a Put or Get by calling lk.ck.Put() or lk.ck.Get().
//
// Use l as the key to store the "lock state" (you would have to decide
// precisely what the lock state is).
func MakeLock(ck IKVClerk, l string) *Lock {
	lk := &Lock{ck: ck}
	// You may add code here

	lk.lockId = RandValue(8)

	lk.lockStateKey = l

	return lk
}

func (lk *Lock) tryGetVersion() Tversion {

	for {
		value, version, err := lk.ck.Get(lk.lockStateKey)

		if err != OK {
			log.Printf("tryGetVersion: err != rpc.OK")

			time.Sleep(time.Second)
			continue
		}

		if value == fmt.Sprintf("useLock(%v)", lk.lockId) {
			return version
		}

		if value != "noUseLock" {
			log.Printf(`[id: %v]: tryGetVersion: value != "noUseLock" but got value = %v`, lk.lockId, value)

			time.Sleep(time.Second)
			continue
		}

		return version

	}

}

func (lk *Lock) tryGetWhoIsUsingLock() (string, bool) {

	for {
		value, _, err := lk.ck.Get(lk.lockStateKey)

		if err == ErrMaybe {

			time.Sleep(time.Second)
			continue
		}

		if value == "noUseLock" {
			return "", true
		} else {
			return value, false
		}

	}

}

func (lk *Lock) validationLock() bool {

	for {

		whoIsUsingLock, reset := lk.tryGetWhoIsUsingLock()

		if reset {
			return false
		}

		if whoIsUsingLock != fmt.Sprintf("useLock(%v)", lk.lockId) {

			time.Sleep(time.Second)
			continue
		} else {
			return true
		}

	}

}

func (lk *Lock) tryLock() bool {

	for {
		version := lk.tryGetVersion()

		err := lk.ck.Put(lk.lockStateKey, fmt.Sprintf("useLock(%v)", lk.lockId), version)

		if err == ErrMaybe {

			time.Sleep(time.Second)
			continue
		}

		ok := lk.validationLock()

		if !ok {

			time.Sleep(time.Second)
			continue
		}

		return true

	}

}

func (lk *Lock) Acquire() {
	// Your code here

	_, _, err := lk.ck.Get(lk.lockStateKey)

	if err == ErrNoKey {
		lk.ck.Put(lk.lockStateKey, "noUseLock", 0)
	}

	lk.tryLock()

}

func (lk *Lock) Release() {
	// Your code here

	value, version, err := lk.ck.Get(lk.lockStateKey)

	if err != OK {
		log.Fatalf("[lockId: %v]: It should not has any errors!!!", lk.lockId)
	}

	if value != fmt.Sprintf("useLock(%v)", lk.lockId) {
		log.Fatalf(`[lockId: %v]: It should get value as "useLock(%v)"`, lk.lockId, lk.lockId)
	}

	lk.ck.Put(lk.lockStateKey, "noUseLock", version)

}
