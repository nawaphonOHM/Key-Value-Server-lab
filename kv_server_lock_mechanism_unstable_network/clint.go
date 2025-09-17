package kv_server_lock_mechanism_unstable_network

import "sync"

type end struct {
	name string
	end  *ClientEnd
}

// Servers are named by ServerName() and clerks lazily make a
// per-clerk ClientEnd to a server.  Each clerk has a Clnt with a map
// of the allocated ends for this clerk.
type Clnt struct {
	mu   sync.Mutex
	net  *Network
	ends map[string]end

	// if nil client can connect to all servers
	// if len(srvs) = 0, client cannot connect to any servers
	srvs []string
}

func (clnt *Clnt) makeEnd(server string) end {
	clnt.mu.Lock()
	defer clnt.mu.Unlock()

	if end, ok := clnt.ends[server]; ok {
		return end
	}

	name := Randstring(20)
	//log.Printf("%p: makEnd %v %v allowed %t", clnt, name, server, clnt.allowedL(server))
	end := end{name: name, end: clnt.net.MakeEnd(name)}
	clnt.net.Connect(name, server)
	if clnt.allowedL(server) {
		clnt.net.Enable(name, true)
	} else {
		clnt.net.Enable(name, false)
	}
	clnt.ends[server] = end
	return end
}

func (clnt *Clnt) Call(server, method string, args interface{}, reply interface{}) bool {
	end := clnt.makeEnd(server)
	ok := end.end.Call(method, args, reply)
	// log.Printf("%p: Call done e %v m %v %v %v ok %v", clnt, end.name, method, args, reply, ok)
	return ok
}

// caller must acquire lock
func (clnt *Clnt) allowedL(server string) bool {
	if clnt.srvs == nil {
		return true
	}
	for _, n := range clnt.srvs {
		if n == server {
			return true
		}
	}
	return false
}

type Clnts struct {
	mu     sync.Mutex
	net    *Network
	clerks map[*Clnt]struct{}
}

func (clnt *Clnt) remove() {
	clnt.mu.Lock()
	defer clnt.mu.Unlock()

	for _, e := range clnt.ends {
		clnt.net.DeleteEnd(e.name)
	}
}

// Create a clnt for a clerk with specific server names, but allow
// only connections to connections to servers in to[].
func (clnts *Clnts) MakeClient() *Clnt {
	return clnts.MakeClientTo(nil)
}

func (clnts *Clnts) DeleteClient(clnt *Clnt) {
	clnts.mu.Lock()
	defer clnts.mu.Unlock()

	if _, ok := clnts.clerks[clnt]; ok {
		clnt.remove()
		delete(clnts.clerks, clnt)
	}
}

func (clnts *Clnts) MakeClientTo(srvs []string) *Clnt {
	clnts.mu.Lock()
	defer clnts.mu.Unlock()
	clnt := makeClntTo(clnts.net, srvs)
	clnts.clerks[clnt] = struct{}{}
	return clnt
}

func makeClntTo(net *Network, srvs []string) *Clnt {
	return &Clnt{ends: make(map[string]end), net: net, srvs: srvs}
}

func makeClnts(net *Network) *Clnts {
	clnts := &Clnts{net: net, clerks: make(map[*Clnt]struct{})}
	return clnts
}

func (clnts *Clnts) cleanup() {
	clnts.mu.Lock()
	defer clnts.mu.Unlock()

	for clnt, _ := range clnts.clerks {
		clnt.remove()
	}
	clnts.clerks = nil
}
