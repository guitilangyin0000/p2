package storageserver

import (
	"container/list"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
    "sort"
	"strconv"
	"time"

	"github.com/cmu440/tribbler/events"
	"github.com/cmu440/tribbler/rpc/storagerpc"
)

type nodeList []storagerpc.Node

type storageServer struct {
	// TODO: implement this!
	isMaster bool
	nodes    nodeList
	nodeMap  map[uint32]bool // 用来记录nodeID是否存在
	numNodes int
	nodeID   uint32
	hostport string

	msgChan chan interface{}

	keyValueMap map[string]string
	keyListMap  map[string][]string
	leaseMap    map[string]map[string]time.Time // map from key to map of hostport to lease start time

	pendingPut     map[string]*list.List // key -> list of pending put request
	pendingListOps map[string]*list.List // key -> list of pending list ops request(Append/Remove)

	pendingPutRevoke  map[string]int // key -> number of pending put revoke reply
	pendingListRevoke map[string]int // key -> number of pending list revoke reply

	clientCache map[string]*rpc.Client // connection cache to trib server
}

func (list nodeList) Len() int           { return len(list) }
func (list nodeList) Less(i, j int) bool { return list[i].NodeID < list[j].NodeID }
func (list nodeList) Swap(i, j int)      { list[i], list[j] = list[j], list[i] }

var (
	LOGE = log.New(os.Stderr, "Error: ", log.Lmicroseconds|log.Lshortfile)
	LOGV = log.New(os.Stdout, "Verbose: ", log.Lmicroseconds|log.Lshortfile)
)

// NewStorageServer creates and starts a new StorageServer. masterServerHostPort
// is the master storage server's host:port address. If empty, then this server
// is the master; otherwise, this server is a slave. numNodes is the total number of
// servers in the ring. port is the port number that this server should listen on.
// nodeID is a random, unsigned 32-bit ID identifying this server.
//
// This function should return only once all storage servers have joined the ring,
// and should return a non-nil error if the storage server could not be started.
func NewStorageServer(masterServerHostPort string, numNodes, port int, nodeID uint32) (StorageServer, error) {
	server := &storageServer{
		isMaster:          false,
		nodes:             make([]storagerpc.Node, 0, numNodes),
		nodeMap:           make(map[uint32]bool),
		numNodes:          numNodes,
		nodeID:            nodeID,
		hostport:          "localhost" + strconv.Itoa(port),
		msgChan:           make(chan interface{}, 1000),
		keyValueMap:       make(map[string]string),
		keyListMap:        make(map[string][]string),
		leaseMap:          make(map[string]map[string]time.Time),
		pendingPut:        make(map[string]*list.List),
		pendingListOps:    make(map[string]*list.List),
		pendingPutRevoke:  make(map[string]int),
		pendingListRevoke: make(map[string]int),
		clientCache:       make(map[string]*rpc.Client),
	}

	node := storagerpc.Node{
		HostPort: server.hostport,
		NodeID:   server.nodeID,
	}

	if masterServerHostPort == "" {
		args := storagerpc.RegisterArgs{
			ServerInfo: node,
		}
		reply := new(storagerpc.RegisterReply)
		for {
			client, err := rpc.DialHTTP("tcp", masterServerHostPort)
			if err != nil {
				LOGE.Println("rpc DialHTTP wrong")
				continue
			}
			LOGV.Println("server %s try to connect to %d\n", port, masterServerHostPort)
			err = client.Call("StorageServer.RegisterServer", &args, reply)
			if err != nil {
				LOGV.Printf("RPC failed\n")
				continue
			}

			if reply.Status == storagerpc.OK {
				LOGV.Printf("server %s register self to %s success\n", strconv.Itoa(port), masterServerHostPort)
				break
			} else {
				time.Sleep(time.Duration(time.Microsecond * 10))
			}

		}
	} else {
		server.isMaster = true
		server.nodes = append(server.nodes, node)
		server.nodeMap[node.NodeID] = true
	}

	for {
		err := rpc.Register(storagerpc.Wrap(server))
		LOGV.Println("rpc register its methods failed")
		if err == nil {
			LOGV.Println("rpc register its methods success")
			break
		}
	}

	rpc.HandleHTTP()

	l, err := net.Listen("tcp", strconv.Itoa(port))
	if err != nil {
		LOGE.Fatal("net listen error", err)
	}

	go http.Serve(l, nil)

	go server.masterHandler()

	return server, nil
}

func (ss *storageServer) masterHandler() {
	var request interface{}
	for {
		select {
		case request = <-ss.msgChan:
			switch request := request.(type) {
			case events.SSRegisterServer:
				ss.doRegisterServer(&request)
			case events.SSGetServers:
				ss.doGetServers(&request)
			case events.SSGet:
				ss.doGet(&request)
			case events.SSGetList:
				ss.doGetList(&request)
			case events.SSPut:
				ss.doPut(&request)
			case events.SSAppendList:
				ss.doAppendList(&request)
			case events.SSRemoveList:
				ss.doRemoveList(&request)
            case events.SSPutRevokeReply:
                ss.doPutRevokeReply(&request)
            case events.SSListRevokeReply:
                ss.doListRevokeReply(&request)
			}
		case <-time.After(time.Duration(time.Microsecond)):
			ss.gc()
		}

	}
}
func (ss *storageServer) RegisterServer(args *storagerpc.RegisterArgs, reply *storagerpc.RegisterReply) error {
	request := events.SSRegisterServer{
		Arg:   args,
		Reply: make(chan *storagerpc.RegisterReply),
	}
	ss.msgChan <- request
	r := <-request.Reply
	reply.Servers = r.Servers
	reply.Status = r.Status
	return nil
}
func (ss *storageServer) doRegisterServer(request *events.SSRegisterServer) {
	args := request.Arg
	reply := new(storagerpc.RegisterReply)
    if _, present := ss.nodeMap[args.ServerInfo.NodeID]; !present {
        ss.nodeMap[args.ServerInfo.NodeID] = true
        ss.nodes = append(ss.nodes, args.ServerInfo)
    }

    if ss.nodes.Len() == ss.numNodes {
        sort.Sort(ss.nodes)
        reply.Servers = ss.nodes
        reply.Status = storagerpc.OK
        request.Reply <- reply
    } else {
        reply.Status = storagerpc.NotReady
        request.Reply <- reply
    }

}

func (ss *storageServer) GetServers(args *storagerpc.GetServersArgs, reply *storagerpc.GetServersReply) error {
	return nil
}
func (ss *storageServer) doGetServers(request *events.SSGetServers) {

}

func (ss *storageServer) Get(args *storagerpc.GetArgs, reply *storagerpc.GetReply) error {
	return nil
}
func (ss *storageServer) doGet(request *events.SSGet) {

}

func (ss *storageServer) GetList(args *storagerpc.GetArgs, reply *storagerpc.GetListReply) error {
	return nil
}
func (ss *storageServer) doGetList(request *events.SSGetList) {

}

func (ss *storageServer) Put(args *storagerpc.PutArgs, reply *storagerpc.PutReply) error {
    request := events.SSPut{args, make(chan *storagerpc.PutReply)}
    ss.msgChan <- request
    r := <-request.Reply
    reply.Status = r.Status
    return nil
}
// 需要考虑已经被client 缓存的key，需要先撤销并且把本地缓存的leaseMap清除,再把对应的keyValueMap更新
// 需要注意的是libstore端的lease需要重新申请才行，并不能有storageServer将新的更新推送过去
func (ss *storageServer) doPut(request *events.SSPut) {
    Key := request.Arg.Key
    if ss.hasPendingPut(Key) {     // 如果因为某些原因导致put不能立即执行，则需要将此缓存起来
        ss.pendingPut[Key].PushBack(request)
    } else {
        // 可以立即执行，则执行然后返回
        if _, present := ss.keyValueMap[Key]; !present {
            ss.keyValueMap[Key] = request.Arg.Value
            request.Reply <- &storagerpc.PutReply{Status:storagerpc.OK}
            return
        }

        // 判断是否在libstore存在缓存
        if _, present := ss.leaseMap[Key]; present {
            var counter int = 0     // 有几处缓存
            for hostport, t := range ss.leaseMap[Key] {
                if isValidLease(t) {
                    counter += 1
                    args := &storagerpc.RevokeLeaseArgs {Key : request.Arg.Key}
                    reply := new(storagerpc.RevokeLeaseReply)
                    rpcClient := ss.getRPCClient(hostport)
                    go func() {
                        rpcClient.Call("LeaseCallbacks.RevokeLease", args, reply)
                        ss.msgChan<-events.SSPutRevokeReply{Key, hostport}
                    }()
                }
            }

            if (counter > 0) {
                // 这里赋值和doPutRevokeReply不会冲突，因为它们是在一个线程里,这里执行完了才会执行ss.pendingPutRevoke[key]-=1
                ss.pendingPutRevoke[Key] = counter
                ss.pendingPut[Key] = list.New()
                ss.pendingPut[Key].PushBack(request)
            } else {
                ss.keyValueMap[Key] = request.Arg.Value
                request.Reply <- &storagerpc.PutReply{Status:storagerpc.OK}
                return
            }

        } else {
            ss.keyValueMap[Key] = request.Arg.Value
            request.Reply <- &storagerpc.PutReply{Status:storagerpc.OK}
            return
        }

    }

}

// 当所有libstore的lease都撤销了之后，处理put
func (ss *storageServer) doPutRevokeReply(request *events.SSPutRevokeReply) {
    key := request.Key
    hostport := request.HostPort

    if !ss.hasPendingPut(key) {
        return
    }

    ss.pendingPutRevoke[key] -= 1
    delete(ss.leaseMap[key], hostport)
    if ss.pendingPutRevoke[key] == 0 {
        // 表示所有的lease都已经撤销
        if pendingList, present := ss.pendingPut[key]; present && pendingList.Len() > 0 {
            ss.keyValueMap[key] = pendingList.Back().Value.(events.SSPut).Arg.Value

            for r := pendingList.Front(); r != nil; r = r.Next() {
                r.Value.(*events.SSPut).Reply <- &storagerpc.PutReply{storagerpc.OK}
            }
        }
        delete(ss.leaseMap, key)
        delete(ss.pendingPut, key)
        delete(ss.pendingPutRevoke, key)
    }
}
func (ss *storageServer) getRPCClient(hostport string) *rpc.Client {
    client, err := rpc.DialHTTP("tcp", hostport)
    if err != nil {
        LOGV.Println("rpc dial error")
    }
    return client
}
func (ss *storageServer) AppendToList(args *storagerpc.PutArgs, reply *storagerpc.PutReply) error {
    request := events.SSAppendList{
        Arg : args,
        Reply : make(chan *storagerpc.PutReply),
    }
    ss.msgChan <- request
    r := <-request.Reply
    reply.Status = r.Status
	return nil
}
func (ss *storageServer) doAppendList(request *events.SSAppendList) {
    key := request.Arg.Key
    if ss.hasPendingListOps(key) {
        ss.pendingListOps[key].PushBack(request);
    } else {
        if lease, present := ss.leaseMap[key]; present {
            var counter int = 0
            for hostport, timestamp := range lease {
                if isValidLease(timestamp) {
                    counter += 1
                    rpcClient := ss.getRPCClient(hostport)
                    arg := &storagerpc.RevokeLeaseArgs{Key : key}
                    reply := new(storagerpc.PutReply)
                    go func() {
                        rpcClient.Call("LeaseCallbacks.RevokeLease", arg, reply)
                        LOGV.Printf("rpcClient call")
                        ss.msgChan <- &events.SSListRevokeReply{Key : key, HostPort : hostport}
                    }()
                }
            }

            if counter > 0 {
                ss.pendingListRevoke[key] = counter
                ss.pendingListOps[key] = list.New()
                ss.pendingListOps[key].PushBack(request)
            } else {
                ss.appendList(request)
            }
        } else {
            ss.appendList(request)
        }
    }
}

func (ss *storageServer) appendList(request *events.SSAppendList) {
    arg := request.Arg
    existKeyMapList := ss.keyListMap[arg.Key]
    for _, value := range existKeyMapList {
        if (value == arg.Value) {
            request.Reply<-&storagerpc.PutReply{storagerpc.ItemExists}
            return
        }
    }
    ss.keyListMap[arg.Key] = append(existKeyMapList, arg.Value)
    request.Reply<-&storagerpc.PutReply{storagerpc.OK}
}

func (ss *storageServer) doListRevokeReply(request *events.SSListRevokeReply) {
    key := request.Key
    hostport := request.HostPort
    if !ss.hasPendingListOps(key) {
        return
    }

    ss.pendingListRevoke[key] -= 1

    delete(ss.leaseMap[key], hostport)
    if (ss.pendingListRevoke[key] == 0) {   // 说明全都撤销了
        if  opsList, present := ss.pendingListOps[key]; present && opsList.Len() > 0 {
            for r := opsList.Front(); r != nil; r = r.Next() {
                switch r.Value.(type){
                case events.SSAppendList:
                    ss.appendList(r.Value.(*events.SSAppendList))
                case events.SSRemoveList:
                    ss.removeKeyValueFromList(r.Value.(*events.SSRemoveList))
                }
            }

        }
        delete(ss.pendingListOps, key)
        delete(ss.pendingListRevoke, key)
        delete(ss.leaseMap, key)
    }
}
func (ss *storageServer) RemoveFromList(args *storagerpc.PutArgs, reply *storagerpc.PutReply) error {
    request := events.SSRemoveList {
        Arg : args,
        Reply : make(chan *storagerpc.PutReply),
    }
    ss.msgChan<-&request
    r := <-request.Reply
    reply.Status = r.Status
    return nil
}

func (ss *storageServer) doRemoveList(request *events.SSRemoveList) {
    key := request.Arg.Key
    if ss.hasPendingListOps(key) {
        ss.pendingListOps[key].PushBack(request)
    } else {
        if lease, present := ss.leaseMap[key]; present {
            var counter int = 0
            for hostport, timestamp := range lease {
                if isValidLease(timestamp) {
                    counter += 1
                    arg := storagerpc.RevokeLeaseArgs{Key : key}
                    reply := new(storagerpc.PutReply)
                    rpcClient := ss.getRPCClient(hostport)
                    go func() {
                        rpcClient.Call("LeaseCallbacks.RevokeLease", &arg, reply)
                        LOGV.Printf("doRemoveList rpc call\n")
                        ss.msgChan <- &events.SSListRevokeReply{Key:key, HostPort:hostport}
                    }()
                }
            }
            if counter > 0 {
                ss.pendingListRevoke[key] = counter
                ss.pendingListOps[key] = list.New()
                ss.pendingListOps[key].PushBack(request)
            } else {
                ss.removeKeyValueFromList(request)
            }
        } else {
            ss.removeKeyValueFromList(request)
        }
    }

}

func (ss *storageServer) removeKeyValueFromList(request *events.SSRemoveList) {
    arg := request.Arg
    existKeyMapList, present := ss.keyListMap[arg.Key]
    if !present {
        request.Reply <- &storagerpc.PutReply{Status:storagerpc.KeyNotFound}
        return
    }
    for i, value := range existKeyMapList {
        if value == request.Arg.Value {
            existKeyMapList[i] = existKeyMapList[len(existKeyMapList)-1]
            ss.keyListMap[arg.Key] = existKeyMapList[0:len(existKeyMapList)-1]
            request.Reply <- &storagerpc.PutReply{storagerpc.OK}
            return
        }
    }

    request.Reply <- &storagerpc.PutReply{storagerpc.ItemNotFound}

}
func (ss *storageServer) gc() {

}

func (ss *storageServer) hasPendingListOps(key string) bool {
    if list, present := ss.pendingListOps[key]; present {
        return list.Len() > 0
    }
    return false
}
func (ss *storageServer) hasPendingPut(key string) bool {
    if list, present := ss.pendingPut[key]; present {
        return list.Len() > 0
    }
    return false
}

func isValidLease(t time.Time) bool {
    return time.Since(t).Seconds() <= storagerpc.LeaseSeconds + storagerpc.LeaseGuardSeconds-1
}
