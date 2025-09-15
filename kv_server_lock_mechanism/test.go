package kv_server_lock_mechanism

import "testing"

type TestKV struct {
	*Test
	t        *testing.T
	reliable bool
}

func MakeTestKV(t *testing.T, reliable bool) *TestKV {
	cfg := MakeConfig(t, 1, reliable, StartKVServer)
	ts := &TestKV{
		t:        t,
		reliable: reliable,
	}
	ts.Test = MakeTest(t, cfg, false, ts)
	return ts
}

func (ts *TestKV) MakeClerk() IKVClerk {
	clnt := ts.Config.MakeClient()
	ck := MakeClerk(clnt, ServerName(GRP0, 0))
	return &TestClerk{ck, clnt}
}

func (ts *TestKV) DeleteClerk(ck IKVClerk) {
	tck := ck.(*TestClerk)
	ts.DeleteClient(tck.Clnt)
}
