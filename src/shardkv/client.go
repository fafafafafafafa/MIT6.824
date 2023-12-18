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
	args.Op = "Get"
	args.ClientId = ck.clientId
	args.SeqId = atomic.AddInt64(&ck.seqId, 1)
	ck.mylog.DFprintf("***ck.Get: args: %+v\n", args)
	for {
		shard := key2shard(key)
		gid := ck.config.Shards[shard]
		ck.mylog.DFprintf("***ck.Get: num: %v, config.Shards: %v, args: %+v\n", ck.config.Num, ck.config.Shards, args)

		if servers, ok := ck.config.Groups[gid]; ok {
			ck.mylog.DFprintf("***ck.Get: to servers(%v), args: %+v\n", servers, args)

			// try each server for the shard.
			for si := 0; si < len(servers); si++ {
				srv := ck.make_end(servers[si])
				ck.mylog.DFprintf("***ck.Get: get srv(%+v), args: %+v\n", srv, args)

				var reply GetReply
				ok := srv.Call("ShardKV.Get", &args, &reply)
				ck.mylog.DFprintf("***ck.Get: ok:%v,\n args: %+v,\n get reply: %+v\n", ok, args, reply)

				if ok && (reply.Err == OK || reply.Err == ErrNoKey) {
					ck.mylog.DFprintf("***ck.Get: finish from group: %v\n args: %+v,\n get reply: %+v\n", gid, args, reply)
			
					return reply.Value
				}
				if ok && (reply.Err == ErrWrongGroup) {
					ck.mylog.DFprintf("***ck.Get: ErrWrongGroup. not in group: %v, args: %+v\n", gid, args)
					time.Sleep(100 * time.Millisecond)
					// ask controler for the latest configuration.
					ck.config = ck.sm.Query(-1)
					break
				}
				if ok && (reply.Err == ErrNotReady){
					// wait and retry
					ck.mylog.DFprintf("***ck.Get: ErrNotReady. group: %v, args: %+v\n", gid, args)

					time.Sleep(10 * time.Millisecond)
					break
				}
				
				// if ok && (reply.Err == ErrNoKey){
				// 	// returns "" if the key does not exist.
				// 	ck.mylog.DFprintf("***ck.Get: ErrNoKey, args: %+v\n", args)

				// 	return ""
				// }
				// ... not ok, or ErrWrongLeader
			}
		}else{
			time.Sleep(100 * time.Millisecond)
			// ask controler for the latest configuration.
			ck.config = ck.sm.Query(-1)
		}

 		
 
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
		ck.mylog.DFprintf("***ck.PutAppend: num: %v, config.Shards: %v, args: %+v\n", ck.config.Num, ck.config.Shards, args)

		if servers, ok := ck.config.Groups[gid]; ok {
			ck.mylog.DFprintf("***ck.PutAppend: to servers(%v), args: %+v\n", servers, args)

			for si := 0; si < len(servers); si++ {
				srv := ck.make_end(servers[si])
				var reply PutAppendReply
				ok := srv.Call("ShardKV.PutAppend", &args, &reply)
				ck.mylog.DFprintf("***ck.PutAppend: ok:%v,\n args: %+v,\n get reply: %+v\n", ok, args, reply)

				if ok && reply.Err == OK {
					ck.mylog.DFprintf("***ck.PutAppend: finish\n args: %+v,\n get reply: %+v\n", args, reply)
					return
				}
				if ok && reply.Err == ErrWrongGroup {
					ck.mylog.DFprintf("***ck.PutAppend: ErrWrongGroup. not in group: %v, args: %+v\n", gid, args)
					time.Sleep(100 * time.Millisecond)
					// ask controler for the latest configuration.
					ck.config = ck.sm.Query(-1)
					break
				}

				if ok && (reply.Err == ErrNotReady){
					// wait and retry
					ck.mylog.DFprintf("***ck.PutAppend: ErrNotReady. group: %v, args: %+v\n", gid, args)

					time.Sleep(10 * time.Millisecond)
					break
				}
				// ... not ok, or ErrWrongLeader
			}
		}else{
			time.Sleep(100 * time.Millisecond)
			// ask controler for the latest configuration.
			ck.config = ck.sm.Query(-1)
		}
		
		
	}
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}
