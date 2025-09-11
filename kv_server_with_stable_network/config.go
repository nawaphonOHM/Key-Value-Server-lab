package kv_server_with_stable_network

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	randmath "math/rand"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const GRP0 = 0

var ncpu_once sync.Once

func Randstring(n int) string {
	b := make([]byte, 2*n)
	rand.Read(b)
	s := base64.URLEncoding.EncodeToString(b)
	return s[0:n]
}

type Config struct {
	*Clnts  // The clnts in the test
	*Groups // The server groups in the test

	t   *testing.T
	net *Network // The network shared by clnts and servers

	start time.Time // time at which make_config() was called
	// begin()/end() statistics
	t0    time.Time // time at which test_test.go called cfg.begin()
	rpcs0 int       // rpcTotal() at start of test
	ops   int32     // number of clerk get/put/append method calls
}

// start a Test.
// print the Test message.
// e.g. cfg.begin("Test (2B): RPC counts aren't too high")
func (cfg *Config) Begin(description string) {
	rel := "reliable"
	if !cfg.net.IsReliable() {
		rel = "unreliable"
	}
	fmt.Printf("%s (%s network)...\n", description, rel)
	cfg.t0 = time.Now()
	cfg.rpcs0 = cfg.RpcTotal()
	atomic.StoreInt32(&cfg.ops, 0)
}

func (cfg *Config) IsReliable() bool {
	return cfg.net.IsReliable()
}

func (cfg *Config) RpcTotal() int {
	return cfg.net.GetTotalCount()
}

// end a Test -- the fact that we got here means there
// was no failure.
// print the Passed message,
// and some performance numbers.
func (cfg *Config) End() {
	cfg.CheckTimeout()
	if cfg.t.Failed() == false {
		t := time.Since(cfg.t0).Seconds()  // real time
		npeers := cfg.Group(GRP0).N()      // number of Raft peers
		nrpc := cfg.RpcTotal() - cfg.rpcs0 // number of RPC sends
		ops := atomic.LoadInt32(&cfg.ops)  //  number of clerk get/put/append calls

		fmt.Printf("  ... Passed --")
		fmt.Printf("  time %4.1fs #peers %d #RPCs %5d #Ops %4d\n", t, npeers, nrpc, ops)
	}
}

func (cfg *Config) Cleanup() {
	cfg.Clnts.cleanup()
	cfg.Groups.cleanup()
	cfg.net.Cleanup()
	if cfg.t.Failed() {
		annotation.cleanup(true, "test failed")
	} else {
		annotation.cleanup(false, "test passed")
	}
	cfg.CheckTimeout()
}

func (cfg *Config) Fatalf(format string, args ...any) {
	const maxStackLen = 50
	fmt.Printf("Fatal: ")
	fmt.Printf(format, args...)
	fmt.Println("")
	var pc [maxStackLen]uintptr
	// Skip two extra frames to account for this function
	// and runtime.Callers itself.
	n := runtime.Callers(2, pc[:])
	if n == 0 {
		panic("testing: zero callers found")
	}
	frames := runtime.CallersFrames(pc[:n])
	var frame runtime.Frame
	for more := true; more; {
		frame, more = frames.Next()
		// Print only frames in our test files
		if strings.Contains(frame.File, "test.go") {
			fmt.Printf("        %v:%d\n", frame.File, frame.Line)
		}
	}
	cfg.t.FailNow()
}

func (cfg *Config) Op() {
	atomic.AddInt32(&cfg.ops, 1)
}

func (cfg *Config) CheckTimeout() {
	// enforce a two minute real-time limit on each test
	if !cfg.t.Failed() && time.Since(cfg.start) > 120*time.Second {
		cfg.t.Fatal("test took longer than 120 seconds")
	}
}

func (cfg *Config) Group(gid Tgid) *ServerGrp {
	return cfg.lookupGroup(gid)
}

func MakeConfig(t *testing.T, n int, reliable bool, mks FstartServer) *Config {
	ncpu_once.Do(func() {
		if runtime.NumCPU() < 2 {
			fmt.Printf("warning: only one CPU, which may conceal locking bugs\n")
		}
		randmath.Seed(makeSeed())
	})
	runtime.GOMAXPROCS(4)
	cfg := &Config{}
	cfg.t = t
	cfg.net = MakeNetwork()
	cfg.Groups = newGroups(cfg.net)
	cfg.MakeGroupStart(GRP0, n, mks)
	cfg.Clnts = makeClnts(cfg.net)
	cfg.start = time.Now()

	cfg.net.Reliable(reliable)

	return cfg
}

func makeSeed() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func (cfg *Config) MakeGroupStart(gid Tgid, nsrv int, mks FstartServer) {
	cfg.MakeGroup(gid, nsrv, mks)
	cfg.Group(gid).StartServers()
}
