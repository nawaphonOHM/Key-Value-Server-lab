package kv_server_lock_mechanism

import (
	"log"
	"sync"
)

const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

type Key string

type Value struct {
	value   string
	version Tversion
}

type KVServer struct {
	mu sync.Mutex

	// Your definitions here.
	data map[Key]*Value
}

func MakeKVServer() *KVServer {
	kv := &KVServer{}
	// Your code here.

	kv.data = make(map[Key]*Value)

	return kv
}

// Get returns the value and version for args.Key, if args.Key
// exists. Otherwise, Get returns ErrNoKey.
func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.

	kv.mu.Lock()
	defer kv.mu.Unlock()

	value, found := kv.data[Key(args.Key)]

	if !found {
		reply.Err = ErrNoKey
		return
	}

	reply.Value = value.value
	reply.Version = value.version
	reply.Err = OK

	return
}

// Update the value for a key if args.Version matches the version of
// the key on the server. If versions don't match, return ErrVersion.
// If the key doesn't exist, Put installs the value if the
// args.Version is 0, and returns ErrNoKey otherwise.
func (kv *KVServer) Put(args *PutArgs, reply *PutReply) {
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()

	key := Key(args.Key)

	value, found := kv.data[key]

	if found && args.Version == 0 {
		reply.Err = ErrVersion
		return
	}

	if !found && args.Version != 0 {
		reply.Err = ErrNoKey
		return
	}

	if found && value.version != args.Version {
		reply.Err = ErrVersion
		return
	}

	if !found && args.Version == 0 {
		kv.data[key] = &Value{
			value:   args.Value,
			version: 1,
		}

		reply.Err = OK
		return
	}

	if found && value.version == args.Version {
		value.value = args.Value
		value.version += 1

		reply.Err = OK
		return
	}

	log.Fatalf("[Server->Put]: should not reach here")

}

// You can ignore all arguments; they are for replicated KVservers
func StartKVServer(ends []*ClientEnd, gid Tgid, srv int, persister *Persister) []IService {
	kv := MakeKVServer()
	return []IService{kv}
}

// You can ignore Kill() for this lab
func (kv *KVServer) Kill() {
}
