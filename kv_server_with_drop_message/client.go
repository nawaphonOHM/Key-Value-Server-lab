package kv_server_with_stable_network

import (
	"time"
)

type Clerk struct {
	clnt   *Clnt
	server string
}

func MakeClerk(clnt *Clnt, server string) IKVClerk {
	ck := &Clerk{clnt: clnt, server: server}
	// You may add code here.
	return ck
}

// Get fetches the current value and version for a key.  It returns
// ErrNoKey if the key does not exist. It keeps trying forever in the
// face of all other errors.
//
// You can send an RPC with code like this:
// ok := ck.clnt.Call(ck.server, "KVServer.Get", &args, &reply)
//
// The types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. Additionally, reply must be passed as a pointer.
func (ck *Clerk) Get(key string) (string, Tversion, Err) {
	// You will have to modify this function.

	args := &GetArgs{Key: key}
	reply := &GetReply{}

	for {
		ok := ck.clnt.Call(ck.server, "KVServer.Get", args, reply)

		if ok {
			break
		}

		time.Sleep(100 * time.Millisecond)

	}

	return reply.Value, reply.Version, reply.Err
}

// Put updates key with value only if the version in the
// request matches the version of the key at the server.  If the
// versions numbers don't match, the server should return
// ErrVersion.  If Put receives an ErrVersion on its first RPC, Put
// should return ErrVersion, since the Put was definitely not
// performed at the server. If the server returns ErrVersion on a
// resend RPC, then Put must return ErrMaybe to the application, since
// its earlier RPC might have been processed by the server successfully
// but the response was lost, and the Clerk doesn't know if
// the Put was performed or not.
//
// You can send an RPC with code like this:
// ok := ck.clnt.Call(ck.server, "KVServer.Put", &args, &reply)
//
// The types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. Additionally, reply must be passed as a pointer.
func (ck *Clerk) Put(key, value string, version Tversion) Err {
	// You will have to modify this function.

	reply := &PutReply{}
	arg := &PutArgs{Key: key, Value: value, Version: version}

	hasFailed := false

	for {
		ok := ck.clnt.Call(ck.server, "KVServer.Put", arg, reply)

		if ok {
			break
		}

		hasFailed = true
		time.Sleep(100 * time.Millisecond)
	}

	if hasFailed {
		return ErrMaybe
	} else {
		return reply.Err
	}

}
