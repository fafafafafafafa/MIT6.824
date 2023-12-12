package shardkv


import "6.824/labrpc"
import "6.824/raft"
import "sync"
import "6.824/labgob"
import "bytes"
import "sync/atomic"
// import "log"
import "time"

type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	Method string //  "Put", "Append" 
	ClientId int64// 全局标识，初始化时随机生成，用于标识客户端
	SeqId int64 // 最新的序列码，随着请求自增，用于排除 重复和过期的请求
	Key string
	Value string
}

type ShardKV struct {
	mu           sync.Mutex
	me           int
	rf           *raft.Raft
	applyCh      chan raft.ApplyMsg
	make_end     func(string) *labrpc.ClientEnd
	gid          int
	ctrlers      []*labrpc.ClientEnd
	maxraftstate int // snapshot if log grows this big

	// Your definitions here.
	dead		 	int32
	dataset			map[string]string // key-value，存储的数据库
	clientId2seqId	map[int64]int64  //
	agreeChs 		map[int]chan Op  // 从applyCh接收信息，再通过agreeChs发送
	stopCh 			chan struct{}
	mylog 			*raft.Mylog
}


func (kv *ShardKV) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	op := Op{
		ClientId: args.ClientId,
		SeqId: args.SeqId,
		Key: args.Key,
	}
	index, _, isLeader := kv.rf.Start(op)
	if isLeader{
		ch := kv.getAgreeChs(index)
		var opMsg Op
		select{
		case opMsg = <- ch:
			kv.mu.Lock()
			// curSeqId, ok := kv.clientId2seqId[opMsg.ClientId]
			// kv.mylog.DFprintf("*kv.Get: kvserver: %v, ok: %v, opMsg.SeqId(%v)-curSeqId(%v)\n", kv.me, ok, opMsg.SeqId, curSeqId)
			// if !ok || opMsg.SeqId >= curSeqId{
				// only handle new request
				v, ok := kv.dataset[opMsg.Key]
				if !ok{
					reply.Err = ErrNoKey
				}else{
					reply.Err = OK
					reply.Value = v 
					kv.mylog.DFprintf("*kv.Get: kvserver: %v, get value: %v\n", kv.me, v)

				}
				// update clientId2seqId
				// kv.mylog.DFprintf("*kv.Get: kvserver: %v, ClientId: %v, SeqId from %v to %v\n", kv.me, opMsg.ClientId, curSeqId, opMsg.SeqId)

				// kv.clientId2seqId[opMsg.ClientId] = opMsg.SeqId
				
			// }
			kv.mu.Unlock()
			close(ch)
			
		case <- time.After(500*time.Millisecond): // 500ms
			reply.Err = ErrWrongLeader
		}

		// if !isSameOp(op, opMsg){
		// 	reply.Err = ErrWrongLeader
		// }
	}else{
		kv.mylog.DFprintf("*kv.Get: is not leader, kvserver: %v\n", kv.me)

		reply.Err = ErrWrongLeader
	}
}

func (kv *ShardKV) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	op := Op{
		Method: args.Op,
		ClientId: args.ClientId,
		SeqId: args.SeqId,
		Key: args.Key,
		Value: args.Value,
	}
	index, _, isLeader := kv.rf.Start(op)
	if isLeader{
		ch := kv.getAgreeChs(index)
		kv.mylog.DFprintf("*kv.PutAppend: kvserver: %v, get agreeChs[%v]\n", kv.me, index)
		// var opMsg Op
		select{
		// case opMsg = <- ch:
		case <-ch:
			reply.Err = OK
			close(ch)
			
		case <- time.After(500*time.Millisecond): // 500ms
			kv.mylog.DFprintf("*kv.PutAppend: kvserver: %v, get opMsg timeout\n", kv.me)

			reply.Err = ErrWrongLeader
		}

		// if !isSameOp(op, opMsg){
		// 	kv.mylog.DFprintf("*kv.PutAppend: different, kvserver: %v, ClientId(%v, %v), SeqId(%v, %v), Method(%v, %v), Key(%v, %v), Value(%v, %v)\n",
		// 	kv.me,
		// 	op.ClientId, opMsg.ClientId, 
		// 	op.SeqId, opMsg.SeqId,
		// 	op.Method, opMsg.Method,
		// 	op.Key, opMsg.Key,
		// 	op.Value, opMsg.Value,
		// 	)

		// 	reply.Err = ErrWrongLeader
		// }
	}else{
		reply.Err = ErrWrongLeader
	}

}

//
// the tester calls Kill() when a ShardKV instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (kv *ShardKV) Kill() {
	kv.rf.Kill()
	// Your code here, if desired.
}

func (kv *ShardKV) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}


func (kv *ShardKV) readDataset(data []byte){
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}

	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var dataset map[string]string
	var clientId2seqId map[int64]int64
	if d.Decode(&dataset) != nil ||
	   d.Decode(&clientId2seqId) != nil{
		kv.mylog.DFprintf("*kv.readDataset: kvserver: %v, decode failed! \n", kv.me)
	}else{
		kv.dataset = dataset
		kv.clientId2seqId = clientId2seqId
		kv.mylog.DFprintf("readDataset: load kvserver: %v,\n dataset: %v,\n clientId2seqId: %v \n", kv.me, kv.dataset, kv.clientId2seqId)
	}

}
func (kv *ShardKV) getAgreeChs(index int) chan Op{
	kv.mu.Lock()
	defer kv.mu.Unlock()
	ch, ok := kv.agreeChs[index]
	if !ok{
		ch = make(chan Op, 1)
		kv.agreeChs[index] = ch
	}
	return ch
}

func (kv *ShardKV) waitApply(){
	// 额外开一个线程，接受applyCh
	// 根据index向相应的chan发送信号
	lastApplied := 0

	for kv.killed()==false{
		select{
		case msg := <- kv.applyCh:
			// kv.mylog.DFprintf("*kv.waitApply: kvserver: %v, CommandIndex: %v, opMsg: %+v\n", kv.me, msg.CommandIndex, msg.Command)
			kv.mylog.DFprintf("*kv.waitApply: kvserver: %v, Msg: %+v\n", kv.me, msg)
			if msg.SnapshotValid{
				// kv.mu.Lock()
				// if kv.rf.CondInstallSnapshot(msg.SnapshotTerm,
				// 	msg.SnapshotIndex, msg.Snapshot) {
					
				// 	kv.dataset = make(map[string]string)
				// 	r := bytes.NewBuffer(msg.Snapshot)
				// 	d := labgob.NewDecoder(r)
				// 	var dataset map[string]string
				// 	var clientId2seqId map[int64]int64

				// 	if d.Decode(&dataset) != nil ||
				// 	   d.Decode(&clientId2seqId) != nil{
				// 		kv.mylog.DFprintf("*kv.waitApply: kvserver: %v, decode error\n", kv.me)
				// 		log.Fatalf("*kv.waitApply: kvserver: %v, decode error\n", kv.me)
				// 	}
				// 	kv.dataset = dataset
				// 	kv.clientId2seqId = clientId2seqId
				// 	lastApplied = msg.SnapshotIndex
				// 	kv.mylog.DFprintf("*kv.waitApply: CondInstallSnapshot, kvserver: %v,\n kv.dataset: %v,\n kv.clientId2seqId: %v,\n lastApplied: %v \n",
				// 	kv.me, kv.dataset, kv.clientId2seqId, lastApplied)
				// }
				// kv.mu.Unlock()

			}else if msg.CommandValid && msg.CommandIndex > lastApplied{
				if msg.Command != nil{
					kv.mu.Lock()
					var opMsg Op = msg.Command.(Op)
					kv.mylog.DFprintf("*kv.waitApply: kvserver: %v, get opMsg: %+v\n", kv.me, opMsg)
					curSeqId, ok := kv.clientId2seqId[opMsg.ClientId]
					kv.mylog.DFprintf("*kv.waitApply: kvserver: %v, ok: %v, opMsg.SeqId(%v)-curSeqId(%v)\n", kv.me, ok, opMsg.SeqId, curSeqId)
					if !ok || opMsg.SeqId > curSeqId{
						// only handle new request
						switch opMsg.Method{
						case "Put":
							kv.mylog.DFprintf("*kv.waitApply: Put, kvserver: %v, key: %v, value: %v\n", kv.me, opMsg.Key, opMsg.Value)
							kv.dataset[opMsg.Key] = opMsg.Value
						case "Append":
							kv.mylog.DFprintf("*kv.waitApply: Append, kvserver: %v, key: %v, value(%v)=\n (past)%v+(add)%v\n", 
							kv.me, opMsg.Key, kv.dataset[opMsg.Key]+opMsg.Value, kv.dataset[opMsg.Key], opMsg.Value)
							kv.dataset[opMsg.Key] += opMsg.Value
						}
						// update clientId2seqId
						kv.mylog.DFprintf("*kv.waitApply: kvserver: %v, ClientId: %v, SeqId from %v to %v\n", kv.me, opMsg.ClientId, curSeqId, opMsg.SeqId)
		
						kv.clientId2seqId[opMsg.ClientId] = opMsg.SeqId
						
					}
					lastApplied = msg.CommandIndex

					if (msg.CommandIndex+1) % 10 == 0{
						// w := new(bytes.Buffer)
						// e := labgob.NewEncoder(w)
						// // v := kv.dataset
						// e.Encode(kv.dataset)
						// e.Encode(kv.clientId2seqId)
						// // kv.mu.Unlock()
						// kv.rf.Snapshot(msg.CommandIndex, w.Bytes())
						// // kv.mu.Lock()

						// kv.mylog.DFprintf("*kv.waitApply: Snapshot, kvserver: %v,\n kv.dataset: %v,\n kv.clientId2seqId: %v,\n lastApplied: %v \n",
						// kv.me, kv.dataset, kv.clientId2seqId, lastApplied)

					}
					kv.mu.Unlock()
					if _, isLeader := kv.rf.GetState(); isLeader {
						ch := kv.getAgreeChs(msg.CommandIndex)
						ch <- opMsg
					}
					
	
				}
			}
		case <- kv.stopCh:
			
		}

 
	}
	kv.mylog.DFprintf("*waitApply(): end, kvserver: %v\n", kv.me)
	// close(kv.applyCh)
	
}

//
// servers[] contains the ports of the servers in this group.
//
// me is the index of the current server in servers[].
//
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
//
// the k/v server should snapshot when Raft's saved state exceeds
// maxraftstate bytes, in order to allow Raft to garbage-collect its
// log. if maxraftstate is -1, you don't need to snapshot.
//
// gid is this group's GID, for interacting with the shardctrler.
//
// pass ctrlers[] to shardctrler.MakeClerk() so you can send
// RPCs to the shardctrler.
//
// make_end(servername) turns a server name from a
// Config.Groups[gid][i] into a labrpc.ClientEnd on which you can
// send RPCs. You'll need this to send RPCs to other groups.
//
// look at client.go for examples of how to use ctrlers[]
// and make_end() to send RPCs to the group owning a specific shard.
//
// StartServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, 
	maxraftstate int, gid int, ctrlers []*labrpc.ClientEnd, make_end func(string) *labrpc.ClientEnd, mylog *raft.Mylog) *ShardKV {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(ShardKV)
	kv.me = me
	kv.maxraftstate = maxraftstate
	kv.make_end = make_end
	kv.gid = gid
	kv.ctrlers = ctrlers

	// Your initialization code here.

	// Use something like this to talk to the shardctrler:
	// kv.mck = shardctrler.MakeClerk(kv.ctrlers)

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh, mylog)
	kv.mylog = mylog

	kv.dataset = make(map[string]string)
	kv.clientId2seqId = make(map[int64]int64)
	kv.agreeChs = make(map[int]chan Op)
	kv.stopCh = make(chan struct{})

	kv.readDataset(persister.ReadSnapshot())

	go kv.waitApply()
	kv.mylog.DFprintf("***StartKVServer(): me: %v\n", me)
	return kv
}
