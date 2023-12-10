package shardctrler

//
// Shardctrler clerk.
//

import "6.824/labrpc"
import "time"
import "crypto/rand"
import "math/big"
import "sync/atomic"
import "6.824/raft"

type Clerk struct {
	servers []*labrpc.ClientEnd
	// Your data here.
	clientId int64
	seqId int64
	mylog *raft.Mylog
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers []*labrpc.ClientEnd, mylog *raft.Mylog) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	// Your code here.
	ck.clientId = nrand()
	ck.seqId = 0
	ck.mylog = mylog

	ck.mylog.DFprintf("**MakeClerk: clientId: %v, seqId: %v\n", ck.clientId, ck.seqId)

	return ck
}

func (ck *Clerk) Query(num int) Config {
	args := &QueryArgs{}
	// Your code here.
	args.Num = num

	args.ClientId = ck.clientId
	args.SeqId = atomic.AddInt64(&ck.seqId, 1)
	ck.mylog.DFprintf("**ck.Query: args: %+v\n", args)
	for {
		// try each known server.
		for _, srv := range ck.servers {
			var reply QueryReply
			ok := srv.Call("ShardCtrler.Query", args, &reply)
			if ok && reply.WrongLeader == false {
				ck.mylog.DFprintf("**ck.Query: finish args: %+v\n", args)

				return reply.Config
			}
		}
		ck.mylog.DFprintf("**ck.Query: resend args: %+v, \n", args)

		time.Sleep(100 * time.Millisecond)
	}
}

func (ck *Clerk) Join(servers map[int][]string) {
	args := &JoinArgs{}
	// Your code here.
	args.Servers = servers

	args.ClientId = ck.clientId
	args.SeqId = atomic.AddInt64(&ck.seqId, 1)
	ck.mylog.DFprintf("**ck.Join: args: %+v\n", args)

	for {
		// try each known server.
		for _, srv := range ck.servers {
			var reply JoinReply
			ok := srv.Call("ShardCtrler.Join", args, &reply)
			if ok && reply.WrongLeader == false {
				ck.mylog.DFprintf("**ck.Join: finish args: %+v\n", args)

				return
			}
		}

		ck.mylog.DFprintf("**ck.Join: resend args: %+v, \n", args)
		time.Sleep(100 * time.Millisecond)
	}
}

func (ck *Clerk) Leave(gids []int) {
	args := &LeaveArgs{}
	// Your code here.
	args.GIDs = gids

	args.ClientId = ck.clientId
	args.SeqId = atomic.AddInt64(&ck.seqId, 1)
	ck.mylog.DFprintf("**ck.Leave: args: %+v\n", args)

	for {
		// try each known server.

		for _, srv := range ck.servers {
			var reply LeaveReply
			ok := srv.Call("ShardCtrler.Leave", args, &reply)
			if ok && reply.WrongLeader == false {
				ck.mylog.DFprintf("**ck.Leave: finish args: %+v\n", args)

				return
			}
		}
		ck.mylog.DFprintf("**ck.Leave: resend args: %+v, \n", args)

		time.Sleep(100 * time.Millisecond)
	}
}

func (ck *Clerk) Move(shard int, gid int) {
	args := &MoveArgs{}
	// Your code here.
	args.Shard = shard
	args.GID = gid

	args.ClientId = ck.clientId
	args.SeqId = atomic.AddInt64(&ck.seqId, 1)
	ck.mylog.DFprintf("**ck.Move: args: %+v\n", args)

	for {
		// try each known server.
		for _, srv := range ck.servers {
			var reply MoveReply
			ok := srv.Call("ShardCtrler.Move", args, &reply)
			if ok && reply.WrongLeader == false {
				ck.mylog.DFprintf("**ck.Move: finish args: %+v\n", args)

				return
			}
		}
		ck.mylog.DFprintf("**ck.Move: resend args: %+v\n", args)

		time.Sleep(100 * time.Millisecond)
	}
}
