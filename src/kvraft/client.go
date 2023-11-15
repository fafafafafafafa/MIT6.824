package kvraft

import "6.824/labrpc"
import "crypto/rand"
import "math/big"
import "6.824/raft"
import "sync/atomic"


type Clerk struct {
	servers []*labrpc.ClientEnd
	// You will have to modify this struct.
	clientId int64// 全局标识，初始化时随机生成，用于标识客户端
	seqId int64 // 最新的序列码，随着请求自增，用于排除 重复和过期的请求
	leaderId int // 
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
	// You'll have to add code here.
	ck.clientId = nrand()
	ck.seqId = 0
	ck.leaderId = 0
	ck.mylog = mylog
	
	return ck
}

//
// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//
func (ck *Clerk) Get(key string) string {

	// You will have to modify this function.
	// seqId := ck.seqId+1
	seqId := atomic.AddInt64(&ck.seqId, 1)
	args := &GetArgs{
		Key: key,
		ClientId: ck.clientId,
		SeqId: seqId,
	}
	for{

		ck.mylog.DFprintf("*ck.Get: sendGet, leaderId: %v, args.Key: %v, args.ClientId: %v, args.SeqId: %v\n", 
		ck.leaderId, args.Key, args.ClientId, args.SeqId)

		reply := &GetReply{}
		ok := ck.sendGet(ck.leaderId, args, reply)

		ck.mylog.DFprintf("*ck.Get: ok: %v, reply: %+v\n", ok, reply)
		if !ok || reply.Err == ErrWrongLeader{
			ck.mylog.DFprintf("*ck.Get: leaderId change, from %v to %v \n", ck.leaderId, (ck.leaderId+1)%len(ck.servers))

			ck.leaderId = (ck.leaderId+1)%len(ck.servers)
			continue
		}else if reply.Err == ErrNoKey{
			ck.mylog.DFprintf("*ck.Get: get ErrNoKey\n")

			return ""
		}
		ck.mylog.DFprintf("*ck.Get: get key: %v, value: %v\n", args.Key, reply.Value)
		return reply.Value
	}

}

//
// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//
func (ck *Clerk) PutAppend(key string, value string, op string) {
	// You will have to modify this function.
	// seqId := ck.seqId+1
	seqId := atomic.AddInt64(&ck.seqId, 1)

	args := &PutAppendArgs{
		Key: key,
		Value: value,
		Op: op, 
		ClientId: ck.clientId,
		SeqId: seqId,
	}
	for{
		ck.mylog.DFprintf("*ck.PutAppend: sendPutAppend, leaderId: %v, args.Op: %v, args.Key: %v, args.Value: %v, args.ClientId: %v, args.SeqId: %v\n", 
		ck.leaderId, args.Op, args.Key, args.Value, args.ClientId, args.SeqId)
		reply := &PutAppendReply{}
		ok := ck.sendPutAppend(ck.leaderId, args, reply)
		ck.mylog.DFprintf("*ck.PutAppend: ok: %v, reply: %+v\n", ok, reply)
		if !ok || reply.Err==ErrWrongLeader{
			ck.mylog.DFprintf("*ck.PutAppend: leaderId change, from %v to %v \n", ck.leaderId, (ck.leaderId+1)%len(ck.servers))
			ck.leaderId = (ck.leaderId+1)%len(ck.servers)
			continue
		}
		break
	}
	
	
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}

func (ck *Clerk) sendPutAppend(server int, args *PutAppendArgs, reply *PutAppendReply) bool{
	ok := ck.servers[server].Call("KVServer.PutAppend", args, reply)
	return ok
}

func (ck *Clerk) sendGet(server int, args *GetArgs, reply *GetReply) bool{
	ok := ck.servers[server].Call("KVServer.Get", args, reply)
	return ok
}


