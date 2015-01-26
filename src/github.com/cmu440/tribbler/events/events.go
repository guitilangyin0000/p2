package events

import (
	"github.com/cmu440/tribbler/rpc/storagerpc"
	"github.com/cmu440/tribbler/rpc/tribrpc"
)

/******************************
To save some typing, there are 3 types of message:
1. LibStore, start with "LS"
2. Trib Server, start with "TS"
3. Storage, start with "SS"
******************************/

/******************************
 * Libstore
 ******************************/
type LSRevokeLeaseReply struct {
	E      error
	Status storagerpc.Status
}

type LSRevokeLease struct {
	Arg   *storagerpc.RevokeLeaseArgs
	Reply chan *LSRevokeLeaseReply
}

type LSGetReply struct {
	E     error
	Reply string
}

type LSGet struct {
	Key   string
	Reply chan *LSGetReply
}

type LSPutReply struct {
	E error
}

type LSPut struct {
	Key, Value string
	Reply      chan *LSPutReply
}

type LSGetListReply struct {
	Reply []string
	E     error
}

type LSGetList struct {
	Key   string
	Reply chan *LSGetListReply
}

type LSRemoveListReply struct {
	E error
}

type LSRemoveList struct {
	Key, Remove string
	Reply       chan *LSRemoveListReply
}

type LSAppendListReply struct {
	E error
}

type LSAppendList struct {
	Key, Add string
	Reply    chan *LSAppendListReply
}

/******************************
 * Storage Server
 ******************************/
type SSRegister struct {
	Arg   *storagerpc.RegisterArgs
	Reply chan *storagerpc.RegisterReply
}

type SSGetServer struct {
	Reply chan *storagerpc.GetServersReply
}

type SSGet struct {
	Arg   *storagerpc.GetArgs
	Reply chan *storagerpc.GetReply
}

type SSGetList struct {
	Arg   *storagerpc.GetArgs
	Reply chan *storagerpc.GetListReply
}

type SSPut struct {
	Arg   *storagerpc.PutArgs
	Reply chan *storagerpc.PutReply
}

type SSAppendList struct {
	Arg   *storagerpc.PutArgs
	Reply chan *storagerpc.PutReply
}

type SSRemoveList struct {
	Arg   *storagerpc.PutArgs
	Reply chan *storagerpc.PutReply
}

type SSPutRevokeReply struct {
	Key      string
	HostPort string
}

type SSListRevokeReply struct {
	Key      string
	HostPort string
}

/******************************
 * Trib Server
 ******************************/
type TSCreateUser struct {
	Arg   *tribrpc.CreateUserArgs
	Reply chan *tribrpc.CreateUserReply
}

type TSAddSub struct {
	Arg   *tribrpc.SubscriptionArgs
	Reply chan *tribrpc.SubscriptionReply
}

type TSRemoveSub struct {
	Arg   *tribrpc.SubscriptionArgs
	Reply chan *tribrpc.SubscriptionReply
}

type TSGetSub struct {
	Arg   *tribrpc.GetSubscriptionsArgs
	Reply chan *tribrpc.GetSubscriptionsReply
}

type TSPostTrib struct {
	Arg   *tribrpc.PostTribbleArgs
	Reply chan *tribrpc.PostTribbleReply
}

type TSGetTrib struct {
	Arg   *tribrpc.GetTribblesArgs
	Reply chan *tribrpc.GetTribblesReply
}

type TSGetTribBySub struct {
	Arg   *tribrpc.GetTribblesArgs
	Reply chan *tribrpc.GetTribblesReply
}
