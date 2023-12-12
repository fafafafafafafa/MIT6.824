package shardkv

//
// client code to talk to a sharded key/value service.
//
// the client first talks to the shardctrler to find out
// the assignment of shards (keys) to groups, and then
// talks to the group that holds the key's shard.
//

import "6.824/labrpc"
import "crypto/rand"
import "math/big"
import "6.824/shardctrler"
import "time"
import "6.824/raft"
import "sync/atomic"
//
// which shard is a key in?
// please use this function,
// and please do not change it.
//
func key2shard(key string) int {
	shard := 0
	if len(key) > 0 {
		shard = int(key[0])
	}
	shard %= shardctrler.NShards
	return shard
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

type Clerk struct {
	sm       *shardctrler.Clerk
	config   shardctrler.Config
	make_end func(string) *labrpc.ClientEnd
	// You will have to modify this struct.
	mylog *raft.Mylog
	clientId int64
	seqId int64

}

//
// the tester calls MakeClerk.
//
// ctrlers[] is needed to call shardctrler.MakeClerk().
//
// make_end(servername) turns a server name from a
// Config.Groups[gid][i] into a labrpc.ClientEnd on which you can
// send RPCs.
//
func MakeClerk(ctrlers []*labrpc.ClientEnd, make_end func(string) *labrpc.ClientEnd, mylog *raft.Mylog) *Clerk {
	ck := new(Clerk)
	ck.sm = shardctrler.MakeClerk(ctrlers, mylog)
	ck.make_end = make_end
	// You'll have to add code here.
	ck.clientId = nrand()
	ck.seqId = 0

	ck.mylog = mylog

	ck.config = ck.sm.Query(-1)

	return ck
}

//
// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
// You will have to modify this function.
//
func (ck *Clerk) Get(key string) string {
	args := GetArgs{}
	args.Key = key

	args.ClientId = ck.clientId
	args.SeqId = atomic.AddInt64(&ck.seqId, 1)
	ck.mylog.DFprintf("***ck.Get: args: %+v\n", args)

	for {
		shard := key2shard(key)
		gid := ck.config.Shards[shard]
		if servers, ok := ck.config.Groups[gid]; ok {
			// try each server for the shard.
			for si := 0; si < len(servers); si++ {
				srv := ck.make_end(servers[si])
				var reply GetReply
				ok := srv.Call("ShardKV.Get", &args, &reply)
				if ok && (reply.Err == OK || reply.Err == ErrNoKey) {
					ck.mylog.DFprintf("***ck.Get: finish\n args: %+v,\n get reply: %+v\n", args, reply)

					return reply.Value
				}
				if ok && (reply.Err == ErrWrongGroup) {
					ck.mylog.DFprintf("***ck.Get: ErrWrongGroup. not in group: %v, args: %+v\n", gid, args)
					break
				}
				
				if ok && (reply.Err == ErrNoKey){
					// returns "" if the key does not exist.
					ck.mylog.DFprintf("***ck.Get: ErrNoKey, args: %+v\n", args)

					return ""
				}
				// ... not ok, or ErrWrongLeader
			}
		}
		time.Sleep(100 * time.Millisecond)
		// ask controler for the latest configuration.
		ck.config = ck.sm.Query(-1)
	}

	// return ""
}

//
// shared by Put and Append.
// You will have to modify this function.
//
func (ck *Clerk) PutAppend(key string, value string, op string) {
	args := PutAppendArgs{}
	args.Key = key
	args.Value = value
	args.Op = op

	args.ClientId = ck.clientId
	args.SeqId = atomic.AddInt64(&ck.seqId, 1)
	ck.mylog.DFprintf("***ck.PutAppend: args: %+v\n", args)

	for {
		shard := key2shard(key)
		gid := ck.config.Shards[shard]
		if servers, ok := ck.config.Groups[gid]; ok {
			for si := 0; si < len(servers); si++ {
				srv := ck.make_end(servers[si])
				var reply PutAppendReply
				ok := srv.Call("ShardKV.PutAppend", &args, &reply)
				if ok && reply.Err == OK {
					ck.mylog.DFprintf("***ck.PutAppend: finish\n args: %+v,\n get reply: %+v\n", args, reply)

					return
				}
				if ok && reply.Err == ErrWrongGroup {
					ck.mylog.DFprintf("***ck.Get: ErrWrongGroup. not in group: %v, args: %+v\n", gid, args)

					break
				}
				// ... not ok, or ErrWrongLeader
			}
		}
		time.Sleep(100 * time.Millisecond)
		// ask controler for the latest configuration.
		ck.config = ck.sm.Query(-1)
	}
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}
