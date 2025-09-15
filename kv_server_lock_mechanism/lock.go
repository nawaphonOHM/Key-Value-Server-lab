package kv_server_lock_mechanism

import (
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

func (lk *Lock) Acquire() {
	// Your code here

	log.Printf("[LockId %v]: I will try to acquire lock", lk.lockId)

	_, _, err := lk.ck.Get(lk.lockStateKey)

	if err == ErrNoKey {
		lk.ck.Put(lk.lockStateKey, "noUseLock", 0)
	}

	for {
		value, version, _ := lk.ck.Get(lk.lockStateKey)

		if value == "useLock" {
			log.Printf("[LockId %v]: There are the others are using... Retry next 1 second", lk.lockId)

			time.Sleep(time.Second)

			continue
		}

		errAcquireLock := lk.ck.Put(lk.lockStateKey, "useLock", version)

		if errAcquireLock != OK {
			log.Printf("[LockId %v]: There are the others are using... Retry next 1 second", lk.lockId)

			time.Sleep(time.Second)

			continue
		}

		log.Printf("[LockId %v]: accquire lock is done", lk.lockId)

		break

	}

}

func (lk *Lock) Release() {
	// Your code here

	log.Printf("[LockId %v]: I will try to release lock", lk.lockId)

	value, version, err := lk.ck.Get(lk.lockStateKey)

	if err != OK {
		log.Fatalf("[lockId: %v]: It should not has any errors!!!", lk.lockId)
	}

	if value != "useLock" {
		log.Fatalf(`[lockId: %v]: It should get value as "useLock"`, lk.lockId)
	}

	lk.ck.Put(lk.lockStateKey, "noUseLock", version)

}
