package kv_server_lock_mechanism_unstable_network

import "sync"

type ServerSrv struct {
	mu       sync.Mutex
	net      *Network
	saved    *Persister
	svcs     []IService // list of services exported by
	endNames []string
	clntEnds []*ClientEnd
}

func makeServer(net *Network, gid Tgid, nsrv int) *ServerSrv {
	srv := &ServerSrv{net: net}
	srv.endNames = make([]string, nsrv)
	srv.clntEnds = make([]*ClientEnd, nsrv)
	for j := 0; j < nsrv; j++ {
		// a fresh set of ClientEnds.
		srv.endNames[j] = Randstring(20)
		// a fresh set of ClientEnds.
		srv.clntEnds[j] = net.MakeEnd(srv.endNames[j])
		net.Connect(srv.endNames[j], ServerName(gid, j))
	}
	return srv
}

// If restart servers, first call ShutdownServer
func (s *ServerSrv) startServer(gid Tgid) *ServerSrv {
	srv := makeServer(s.net, gid, len(s.endNames))
	// a fresh persister, so old instance doesn't overwrite
	// new instance's persisted state.
	// give the fresh persister a copy of the old persister's
	// state, so that the spec is that we pass StartKVServer()
	// the last persisted state.
	if s.saved != nil {
		srv.saved = s.saved.Copy()
	} else {
		srv.saved = MakePersister()
	}
	return srv
}

// connect s to servers listed in to
func (s *ServerSrv) connect(sg *ServerGrp, to []int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for j := 0; j < len(to); j++ {
		if sg.IsConnected(to[j]) {
			//log.Printf("connect %d to %d (%v)", s.id, to[j], s.endNames[to[j]])
			endname := s.endNames[to[j]]
			s.net.Enable(endname, true)
		}
	}
}
