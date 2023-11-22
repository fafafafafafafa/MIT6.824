package kvraft

import (
	"6.824/labgob"
	"6.824/labrpc"
	"6.824/raft"
	"log"
	"sync"
	"sync/atomic"
	"time"
	// "fmt"
	// "runtime/pprof"
	// "io"
	"bytes"
)

const Debug = true
// type Mylog struct{
// 	W io.Writer
// 	Debug bool
// } 

// func (mylog *raft.Mylog)DFprintf(format string, a ...interface{}) (n int, err error) {
// 	mylog.Debug = Debug
// 	if mylog.Debug {
// 		// log.Printf(format, a...)
// 		fmt.Fprintf(mylog.W, format, a...)
// 		log.Printf(format, a...)
// 	}
// 	return
// }
// func (mylog *raft.Mylog)GoroutineStack(){
// 	_ = pprof.Lookup("goroutine").WriteTo(mylog.W, 1)
// }

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}


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

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()

	maxraftstate int // snapshot if log grows this big

	// Your definitions here.
	dataset	map[string]string // key-value，存储的数据库
	clientId2seqId	map[int64]int64  //
	agreeChs map[int]chan Op  // 从applyCh接收信息，再通过agreeChs发送
	stopCh chan struct{}
	mylog *raft.Mylog
}


func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
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
			curSeqId, ok := kv.clientId2seqId[opMsg.ClientId]
			kv.mylog.DFprintf("*kv.Get: kvserver: %v, ok: %v, opMsg.SeqId(%v)-curSeqId(%v)\n", kv.me, ok, opMsg.SeqId, curSeqId)
			if !ok || opMsg.SeqId >= curSeqId{
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
				kv.mylog.DFprintf("*kv.Get: kvserver: %v, ClientId: %v, SeqId from %v to %v\n", kv.me, opMsg.ClientId, curSeqId, opMsg.SeqId)

				// kv.clientId2seqId[opMsg.ClientId] = opMsg.SeqId
				
			}
			kv.mu.Unlock()
			close(ch)
			
		case <- time.After(500*time.Millisecond): // 500ms
			reply.Err = ErrWrongLeader
		}

		if !isSameOp(op, opMsg){
			reply.Err = ErrWrongLeader
		}
	}else{
		kv.mylog.DFprintf("*kv.Get: is not leader, kvserver: %v\n", kv.me)

		reply.Err = ErrWrongLeader
	}
}
func (kv *KVServer) getAgreeChs(index int) chan Op{
	kv.mu.Lock()
	defer kv.mu.Unlock()
	ch, ok := kv.agreeChs[index]
	if !ok{
		ch = make(chan Op, 1)
		kv.agreeChs[index] = ch
	}
	return ch
}

func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
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
		var opMsg Op
		select{
		case opMsg = <- ch:

			reply.Err = OK
			close(ch)
			
		case <- time.After(500*time.Millisecond): // 500ms
			kv.mylog.DFprintf("*kv.PutAppend: kvserver: %v, get opMsg timeout\n", kv.me)

			reply.Err = ErrWrongLeader
		}

		if !isSameOp(op, opMsg){
			kv.mylog.DFprintf("*kv.PutAppend: different, kvserver: %v, ClientId(%v, %v), SeqId(%v, %v), Method(%v, %v), Key(%v, %v), Value(%v, %v)\n",
			kv.me,
			op.ClientId, opMsg.ClientId, 
			op.SeqId, opMsg.SeqId,
			op.Method, opMsg.Method,
			op.Key, opMsg.Key,
			op.Value, opMsg.Value,
			)

			reply.Err = ErrWrongLeader
		}
	}else{
		reply.Err = ErrWrongLeader
	}


}
func isSameOp(cmd1 Op, cmd2 Op) bool{
	return cmd1.ClientId == cmd2.ClientId && cmd1.SeqId == cmd2.SeqId && cmd1.Method == cmd2.Method && cmd1.Key == cmd2.Key && cmd1.Value == cmd2.Value 
}

func (kv *KVServer) readDataset(data []byte){
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}

	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var dataset map[string]string
	var clientId2seqId map[int64]int64
	if d.Decode(&dataset) != nil ||
	   d.Decode(&clientId2seqId) != nil{
		kv.mylog.DFprintf("*kv.readDataset: decode failed! \n")
	}else{
		kv.dataset = dataset
		kv.clientId2seqId = clientId2seqId
		kv.mylog.DFprintf("readDataset: load kvserver:%v , dataset: %v \n", kv.me, kv.dataset)
	}

}

func (kv *KVServer) waitApply(){
	// 额外开一个线程，接受applyCh
	// 根据index向相应的chan发送信号
	lastApplied := 0

	for kv.killed()==false{
		select{
		case msg := <- kv.applyCh:
			// kv.mylog.DFprintf("*kv.waitApply: kvserver: %v, CommandIndex: %v, opMsg: %+v\n", kv.me, msg.CommandIndex, msg.Command)
			kv.mylog.DFprintf("*kv.waitApply: kvserver: %v, Msg: %+v\n", kv.me, msg)
			if msg.SnapshotValid{
				kv.mu.Lock()
				if kv.rf.CondInstallSnapshot(msg.SnapshotTerm,
					msg.SnapshotIndex, msg.Snapshot) {
					
					kv.dataset = make(map[string]string)
					r := bytes.NewBuffer(msg.Snapshot)
					d := labgob.NewDecoder(r)
					var dataset map[string]string
					var clientId2seqId map[int64]int64

					if d.Decode(&dataset) != nil ||
					   d.Decode(&clientId2seqId) != nil{
						kv.mylog.DFprintf("*kv.waitApply: decode error\n")
						log.Fatalf("*kv.waitApply: decode error\n")
					}
					kv.dataset = dataset
					kv.clientId2seqId = clientId2seqId
					lastApplied = msg.SnapshotIndex

				}
				kv.mu.Unlock()

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
							kv.mylog.DFprintf("*kv.waitApply: Append, kvserver: %v, key: %v, value(%v)= (past)%v+(add)%v\n", 
							kv.me, opMsg.Key, kv.dataset[opMsg.Key]+opMsg.Value, kv.dataset[opMsg.Key], opMsg.Value)
							kv.dataset[opMsg.Key] += opMsg.Value
						}
						// update clientId2seqId
						kv.mylog.DFprintf("*kv.waitApply: kvserver: %v, ClientId: %v, SeqId from %v to %v\n", kv.me, opMsg.ClientId, curSeqId, opMsg.SeqId)
		
						kv.clientId2seqId[opMsg.ClientId] = opMsg.SeqId
						
					}
					lastApplied = msg.CommandIndex

					if (msg.CommandIndex+1) % 10 == 0{
						w := new(bytes.Buffer)
						e := labgob.NewEncoder(w)
						// v := kv.dataset
						e.Encode(kv.dataset)
						e.Encode(kv.clientId2seqId)
						// kv.mu.Unlock()
						kv.rf.Snapshot(msg.CommandIndex, w.Bytes())
						// kv.mu.Lock()

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
// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
//
func (kv *KVServer) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.rf.Kill()
	// Your code here, if desired.
	close(kv.stopCh)
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int, kvlog *raft.Mylog) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate

	// You may need initialization code here.
	kv.mylog = kvlog
	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh, kvlog)

	// You may need initialization code here.
	kv.dataset = make(map[string]string)
	kv.clientId2seqId = make(map[int64]int64)
	kv.agreeChs = make(map[int]chan Op)
	kv.stopCh = make(chan struct{})

	kv.readDataset(persister.ReadSnapshot())

	go kv.waitApply()
	kv.mylog.DFprintf("*StartKVServer(): me: %v\n", me)
	return kv
}






