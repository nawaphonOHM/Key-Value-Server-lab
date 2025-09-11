package kv_server_with_stable_network

import (
	"encoding/json"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/anishathalye/porcupine"
)

type IKVClerk interface {
	Get(string) (string, Tversion, Err)
	Put(string, string, Tversion) Err
}

type IClerkMaker interface {
	MakeClerk() IKVClerk
	DeleteClerk(IKVClerk)
}

type OpLog struct {
	operations []porcupine.Operation
	sync.Mutex
}

type Test struct {
	*Config
	t          *testing.T
	oplog      *OpLog
	mck        IClerkMaker
	randomkeys bool
}

func (ts *Test) Cleanup() {
	ts.Config.End()
	ts.Config.Cleanup()
}

// spawn ncli clients
func (ts *Test) SpawnClientsAndWait(nclnt int, t time.Duration, fn Fclnt) []ClntRes {
	ca := make([]chan ClntRes, nclnt)
	done := make(chan struct{})
	for cli := 0; cli < nclnt; cli++ {
		ca[cli] = make(chan ClntRes)
		go ts.runClient(cli, ca[cli], done, ts.mck, fn)
	}
	time.Sleep(t)
	for i := 0; i < nclnt; i++ {
		done <- struct{}{}
	}
	rs := make([]ClntRes, nclnt)
	for cli := 0; cli < nclnt; cli++ {
		rs[cli] = <-ca[cli]
	}
	return rs
}

type ClntRes struct {
	Nok    int
	Nmaybe int
}

// One of perhaps many clients doing OnePut's until done signal.
func (ts *Test) OneClientPut(cli int, ck IKVClerk, ka []string, done chan struct{}) ClntRes {
	res := ClntRes{}
	verm := make(map[string]Tversion)
	for _, k := range ka {
		verm[k] = Tversion(0)
	}
	ok := false
	for true {
		select {
		case <-done:
			return res
		default:
			k := ka[0]
			if ts.randomkeys {
				k = ka[rand.Int()%len(ka)]
			}
			verm[k], ok = ts.OnePut(cli, ck, k, verm[k])
			if ok {
				res.Nok += 1
			} else {
				res.Nmaybe += 1
			}
		}
	}
	return res
}

func (ts *Test) CheckPutConcurrent(ck IKVClerk, key string, rs []ClntRes, res *ClntRes, reliable bool) {
	e := EntryV{}
	ver0 := ts.GetJson(ck, key, -1, &e)
	for _, r := range rs {
		res.Nok += r.Nok
		res.Nmaybe += r.Nmaybe
	}
	if reliable {
		if ver0 != Tversion(res.Nok) {
			ts.Fatalf("Reliable: Wrong number of puts: server %d clnts %v", ver0, res)
		}
	} else if ver0 > Tversion(res.Nok+res.Nmaybe) {
		ts.Fatalf("Unreliable: Wrong number of puts: server %d clnts %v", ver0, res)
	}
}

// a client runs the function f and then signals it is done
func (ts *Test) runClient(me int, ca chan ClntRes, done chan struct{}, mkc IClerkMaker, fn Fclnt) {
	ck := mkc.MakeClerk()
	v := fn(me, ck, done)
	ca <- v
	mkc.DeleteClerk(ck)
}

// Keep trying until we get one put succeeds while other clients
// tryint to put to the same key
func (ts *Test) OnePut(me int, ck IKVClerk, key string, ver Tversion) (Tversion, bool) {
	for true {
		err := ts.PutJson(ck, key, EntryV{me, ver}, ver, me)
		if !(err == OK || err == ErrVersion || err == ErrMaybe) {
			ts.Fatalf("Wrong error %v", err)
		}
		e := EntryV{}
		ver0 := ts.GetJson(ck, key, me, &e)
		if err == OK && ver0 == ver+1 { // my put?
			if e.Id != me && e.V != ver {
				ts.Fatalf("Wrong value %v", e)
			}
		}
		ver = ver0
		if err == OK || err == ErrMaybe {
			return ver, err == OK
		}
	}
	return 0, false
}

type EntryV struct {
	Id int
	V  Tversion
}

func (ts *Test) GetJson(ck IKVClerk, key string, me int, v any) Tversion {
	if val, ver, err := Get(ts.Config, ck, key, ts.oplog, me); err == OK {
		if err := json.Unmarshal([]byte(val), v); err != nil {
			ts.Fatalf("Unmarshal err %v", ver)
		}
		return ver
	} else {
		ts.Fatalf("%d: Get %q err %v", me, key, err)
		return 0
	}
}

type Fclnt func(int, IKVClerk, chan struct{}) ClntRes

func (ts *Test) PutJson(ck IKVClerk, key string, v any, ver Tversion, me int) Err {
	b, err := json.Marshal(v)
	if err != nil {
		ts.Fatalf("%d: marshal %v", me, err)
	}
	return Put(ts.Config, ck, key, string(b), ver, ts.oplog, me)
}

func MakeTest(t *testing.T, cfg *Config, randomkeys bool, mck IClerkMaker) *Test {
	ts := &Test{
		Config:     cfg,
		t:          t,
		mck:        mck,
		oplog:      &OpLog{},
		randomkeys: randomkeys,
	}
	return ts
}

type TestClerk struct {
	IKVClerk
	Clnt *Clnt
}

// n specifies the length of the string to be generated.
func RandValue(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}
