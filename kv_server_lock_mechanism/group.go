package kv_server_lock_mechanism

import (
	"strconv"
	"sync"
)

type Tgid int

// A service must support Kill(); the tester will Kill()
// on service returned by FstartServer()
type IService interface {
	Kill()
}

// The tester may have many groups of servers (e.g., one per Raft group).
// Groups are named 0, 1, and so on.
type Groups struct {
	mu   sync.Mutex
	net  *Network
	grps map[Tgid]*ServerGrp
}

type ServerGrp struct {
	net         *Network
	srvs        []*ServerSrv
	servernames []string
	gid         Tgid
	connected   []bool // whether each server is on the net
	mks         FstartServer
	mu          sync.Mutex
}

// Start server and return the services to register with labrpc
type FstartServer func(ends []*ClientEnd, grp Tgid, srv int, persister *Persister) []IService

func (sg *ServerGrp) N() int {
	return len(sg.srvs)
}

func (gs *Groups) cleanup() {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	for _, sg := range gs.grps {
		sg.cleanup()
	}
}

func (gs *Groups) lookupGroup(gid Tgid) *ServerGrp {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	return gs.grps[gid]
}

func (sg *ServerGrp) cleanup() {
	for _, s := range sg.srvs {
		if s.svcs != nil {
			for _, svc := range s.svcs {
				svc.Kill()
			}
		}
	}
}

// Each server has a name: i'th server of group gid. If there is only a single
// server, it its gid = 0 and its i is 0.
func ServerName(gid Tgid, i int) string {
	return "server-" + strconv.Itoa(int(gid)) + "-" + strconv.Itoa(i)
}

func newGroups(net *Network) *Groups {
	return &Groups{net: net, grps: make(map[Tgid]*ServerGrp)}
}

func (gs *Groups) MakeGroup(gid Tgid, nsrv int, mks FstartServer) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gs.grps[gid] = makeSrvGrp(gs.net, gid, nsrv, mks)
}

// create a full set of KV servers.
func (sg *ServerGrp) StartServers() {
	sg.start()
	sg.ConnectAll()
}

func makeSrvGrp(net *Network, gid Tgid, n int, mks FstartServer) *ServerGrp {
	sg := &ServerGrp{
		net:       net,
		srvs:      make([]*ServerSrv, n),
		gid:       gid,
		connected: make([]bool, n),
		mks:       mks,
	}
	for i, _ := range sg.srvs {
		sg.srvs[i] = makeServer(net, gid, n)
	}
	sg.servernames = make([]string, n)
	for i := 0; i < n; i++ {
		sg.servernames[i] = ServerName(gid, i)
	}
	return sg
}

func (sg *ServerGrp) start() {
	for i, _ := range sg.srvs {
		sg.StartServer(i)
	}
}

func (sg *ServerGrp) ConnectAll() {
	for i, _ := range sg.srvs {
		sg.ConnectOne(i)
	}
}

// If restart servers, first call shutdownserver
func (sg *ServerGrp) StartServer(i int) {
	srv := sg.srvs[i].startServer(sg.gid)
	sg.srvs[i] = srv

	srv.svcs = sg.mks(srv.clntEnds, sg.gid, i, srv.saved)
	labsrv := MakeServerLabRPC()
	for _, svc := range srv.svcs {
		s := MakeService(svc)
		labsrv.AddService(s)
	}
	sg.net.AddServer(ServerName(sg.gid, i), labsrv)
}

func (sg *ServerGrp) ConnectOne(i int) {
	sg.connect(i, sg.all())
}

// attach server i to servers listed in to caller must hold cfg.mu.
func (sg *ServerGrp) connect(i int, to []int) {
	//log.Printf("connect peer %d to %v\n", i, to)

	sg.connected[i] = true

	// connect outgoing end points
	sg.srvs[i].connect(sg, to)

	// connect incoming end points to me
	for j := 0; j < len(to); j++ {
		if sg.IsConnected(to[j]) {
			//log.Printf("connect %d (%v) to %d", to[j], sg.srvs[to[j]].endNames[i], i)
			endname := sg.srvs[to[j]].endNames[i]
			sg.net.Enable(endname, true)
		}
	}
}

func (sg *ServerGrp) all() []int {
	all := make([]int, len(sg.srvs))
	for i, _ := range sg.srvs {
		all[i] = i
	}
	return all
}

func (sg *ServerGrp) IsConnected(i int) bool {
	defer sg.mu.Unlock()
	sg.mu.Lock()
	return sg.connected[i]
}
