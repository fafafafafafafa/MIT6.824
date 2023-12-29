package shardkv


import "6.824/labrpc"
import "6.824/raft"
import "sync"
import "6.824/labgob"
import "bytes"
import "sync/atomic"
import "log"
import "time"
import "6.824/shardctrler"

type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	Method string //  "Put", "Append", "Update"
	ClientId int64// 全局标识，初始化时随机生成，用于标识客户端
	SeqId int64 // 最新的序列码，随着请求自增，用于排除 重复和过期的请求
	Key string
	Value string

	Config  shardctrler.Config
	Shard		int
	ShardData	map[string]string
	ClientId2seqId	map[int64]int64
	ConfigNum	int


}
type ApplyReply struct{
	Err Err
	Op	Op
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
	agreeChs 		map[int]chan ApplyReply  // 从applyCh接收信息，再通过agreeChs发送
	stopCh 			chan struct{}

	config 			shardctrler.Config
	lastConfig		shardctrler.Config

	mck				*shardctrler.Clerk	

	moveInShards	map[int]int	// shard -> group
	shardState		map[int]ShardState	// shard -> ShardState

	mylog 			*raft.Mylog		
}


func (kv *ShardKV) Get(args *GetArgs, reply *GetReply) {
	// if key that the server isn't responsible for
	kv.mu.Lock()
	shard := key2shard(args.Key)
	switch kv.shardState[shard]{
	case NOTSERVED, DEL:
		kv.mylog.DFprintf("***kv.Get, key: %v is not in the group: %v, config.Shards: %v\n", args.Key, kv.gid, kv.config.Shards)
		reply.Err = ErrWrongGroup
		kv.mu.Unlock()
		return

	// case NOTREADY:
	// 	kv.mylog.DFprintf("***kv.Get, key: %v in the group: %v is not ready, config.Shards: %v\n", args.Key, kv.gid, kv.config.Shards)
	// 	reply.Err = ErrNotReady
	// 	kv.mu.Unlock()
	// 	return
	default: 
		kv.mu.Unlock()
	}

	// Your code here.
	op := Op{
		Method: args.Op,
		ClientId: args.ClientId,
		SeqId: args.SeqId,
		Key: args.Key,
	}
	
	index, _, isLeader := kv.rf.Start(op)

	if isLeader{
		ch := kv.getAgreeChs(index)
		var applyMsg ApplyReply
		select{
		case applyMsg = <- ch:
			kv.mu.Lock()
			
			shard := key2shard(args.Key)
			switch kv.shardState[shard]{
			case NOTSERVED, DEL:
				kv.mylog.DFprintf("***kv.Get-, key: %v is not in the group: %v, config.Shards: %v\n", args.Key, kv.gid, kv.config.Shards)
				reply.Err = ErrWrongGroup
				kv.mu.Unlock()
				return

			case NOTREADY:
				kv.mylog.DFprintf("***kv.Get-, key: %v in the group: %v is not ready, config.Shards: %v\n", args.Key, kv.gid, kv.config.Shards)
				reply.Err = ErrNotReady
				kv.mu.Unlock()
				return
			case SERVED, READY:
				v, ok := kv.dataset[applyMsg.Op.Key]
				if !ok{
					reply.Err = ErrNoKey
				}else{
					reply.Err = OK
					reply.Value = v 
					kv.mylog.DFprintf("***kv.Get-: group: %v, kvserver: %v, get value: %v\n", kv.gid, kv.me, v)
	
				}
			default: 
			
			}
			
			kv.mu.Unlock()
			close(ch)
			
		case <- time.After(500*time.Millisecond): // 500ms
			reply.Err = ErrWrongLeader
		}

		// if !isSameOp(op, opMsg){
		// 	reply.Err = ErrWrongLeader
		// }
	}else{
		kv.mylog.DFprintf("***kv.Get: is not leader, group: %v, kvserver: %v\n", kv.gid, kv.me)

		reply.Err = ErrWrongLeader
	}
}

func (kv *ShardKV) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	// if key that the server isn't responsible for
	kv.mu.Lock()
	shard := key2shard(args.Key)
	switch kv.shardState[shard]{
	case NOTSERVED, DEL:
		kv.mylog.DFprintf("***kv.PutAppend, key: %v is not in the group: %v, config.Shards: %v\n", args.Key, kv.gid, kv.config.Shards)
		reply.Err = ErrWrongGroup
		kv.mu.Unlock()
		return

	// case NOTREADY:
	// 	kv.mylog.DFprintf("***kv.PutAppend, key: %v in the group: %v is not ready, config.Shards: %v\n", args.Key, kv.gid, kv.config.Shards)
	// 	reply.Err = ErrNotReady
	// 	kv.mu.Unlock()
	// 	return
	default: 
		kv.mu.Unlock()
	}
	

	op := Op{
		Method: args.Op,
		ClientId: args.ClientId,
		SeqId: args.SeqId,
		Key: args.Key,
		Value: args.Value,
	}
	kv.mylog.DFprintf("***kv.PutAppend: group: %v, kvserver: %v, op: %+v\n", kv.gid, kv.me, op)

	index, _, isLeader := kv.rf.Start(op)
	if isLeader{
		ch := kv.getAgreeChs(index)
		kv.mylog.DFprintf("***kv.PutAppend: group: %v, kvserver: %v, get agreeChs[%v]\n", kv.gid, kv.me, index)
		var applyMsg ApplyReply
		select{
		case applyMsg = <- ch:
			reply.Err = applyMsg.Err
			close(ch)
			
		case <- time.After(500*time.Millisecond): // 500ms
			kv.mylog.DFprintf("***kv.PutAppend: group: %v, kvserver: %v, get opMsg timeout\n", kv.gid, kv.me)

			reply.Err = ErrWrongLeader
		}

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
	atomic.StoreInt32(&kv.dead, 1)
	close(kv.stopCh)
}

func (kv *ShardKV) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}


func (kv *ShardKV) readDataset(data []byte){
	if data == nil || len(data) < 1 { // bootstrap without any state?
		 
		for i := 0; i < shardctrler.NShards; i++{
			kv.shardState[i] = NOTSERVED
		}
		 
		return
	}

	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var dataset map[string]string
	var clientId2seqId map[int64]int64
	var shardState map[int]ShardState
	var moveInShards map[int]int
	var lastConfig shardctrler.Config
	var config shardctrler.Config

	if d.Decode(&dataset) != nil ||
	   d.Decode(&clientId2seqId) != nil ||
	   d.Decode(&shardState) != nil	||
	   d.Decode(&moveInShards) != nil ||
	   d.Decode(&lastConfig) != nil ||
	   d.Decode(&config) != nil {
		kv.mylog.DFprintf("***kv.readDataset: group: %v, kvserver: %v, decode failed! \n", kv.gid, kv.me)
	}else{
		kv.dataset = dataset
		kv.clientId2seqId = clientId2seqId
		kv.shardState = shardState
		kv.moveInShards = moveInShards
		kv.lastConfig = lastConfig
		kv.config = config
		
		kv.mylog.DFprintf("***kv.readDataset: load group: %v, kvserver: %v,\n dataset: %v,\n clientId2seqId: %v,\n shardState: %v,\n kv.moveInShards: %v,\n lastConfig: %v,\n config: %v\n",
		kv.gid, kv.me, kv.dataset, kv.clientId2seqId, kv.shardState, kv.moveInShards, kv.lastConfig, kv.config)
	}

}
func (kv *ShardKV) getAgreeChs(index int) chan ApplyReply{
	kv.mu.Lock()
	defer kv.mu.Unlock()
	ch, ok := kv.agreeChs[index]
	if !ok{
		ch = make(chan ApplyReply, 1)
		kv.agreeChs[index] = ch
	}
	return ch
}

// func (kv *ShardKV) isKeyServed(key string)bool{
// 	shard := key2shard(key)
// 	g := kv.config.Shards[shard]
// 	kv.mylog.DFprintf("***kv.isKeyServed: key:%v, shard: %v, gid(%v)-kv.gid(%v)\n", key, shard, g, kv.gid)
// 	return g == kv.gid
// }



func (kv *ShardKV) doPut(opMsg Op) Err{

	curSeqId, ok := kv.clientId2seqId[opMsg.ClientId]
	kv.mylog.DFprintf("***kv.doPut: group: %v, kvserver: %v, ok: %v, opMsg.SeqId(%v)-curSeqId(%v)\n",
	kv.gid, kv.me, ok, opMsg.SeqId, curSeqId)
	if !ok || opMsg.SeqId > curSeqId{
		
		shard := key2shard(opMsg.Key)
		switch kv.shardState[shard]{
		case NOTSERVED, DEL:
			kv.mylog.DFprintf("***kv.doPut, key: %v is not in the group: %v, config.Shards: %v\n", opMsg.Key, kv.gid, kv.config.Shards)
			return ErrWrongGroup

		case NOTREADY:
			kv.mylog.DFprintf("***kv.doPut, key: %v in the group: %v is NOTREADY, config.Shards: %v\n", opMsg.Key, kv.gid, kv.config.Shards)
			return ErrNotReady
		}
		kv.mylog.DFprintf("***kv.doPut: Put, group: %v, kvserver: %v, key: %v, value: %v\n", kv.gid, kv.me, opMsg.Key, opMsg.Value)
		kv.dataset[opMsg.Key] = opMsg.Value

		// update clientId2seqId
		kv.mylog.DFprintf("***kv.doPut: group: %v, kvserver: %v, ClientId: %v, SeqId from %v to %v\n", 
		kv.gid, kv.me, opMsg.ClientId, curSeqId, opMsg.SeqId)
		
		kv.clientId2seqId[opMsg.ClientId] = opMsg.SeqId
		
	}
	return OK
}

func (kv *ShardKV) doAppend(opMsg Op)Err{
	curSeqId, ok := kv.clientId2seqId[opMsg.ClientId]
	kv.mylog.DFprintf("***kv.doAppend: group: %v, kvserver: %v, ok: %v, opMsg.SeqId(%v)-curSeqId(%v)\n", kv.gid, kv.me, ok, opMsg.SeqId, curSeqId)
	if !ok || opMsg.SeqId > curSeqId{
		shard := key2shard(opMsg.Key)
		switch kv.shardState[shard]{
		case NOTSERVED, DEL:
			kv.mylog.DFprintf("***kv.doAppend, key: %v is not in the group: %v, config.Shards: %v\n", opMsg.Key, kv.gid, kv.config.Shards)
			return ErrWrongGroup

		case NOTREADY:
			kv.mylog.DFprintf("***kv.doAppend, key: %v in the group: %v is NOTREADY, config.Shards: %v\n", opMsg.Key, kv.gid, kv.config.Shards)
			return ErrNotReady
		}
		
		kv.mylog.DFprintf("***kv.doAppend: Append, group: %v, kvserver: %v, key: %v, value(%v)=\n (past)%v+(add)%v\n", 
		kv.gid, kv.me, opMsg.Key, kv.dataset[opMsg.Key]+opMsg.Value, kv.dataset[opMsg.Key], opMsg.Value)

		kv.dataset[opMsg.Key] += opMsg.Value

		// update clientId2seqId
		kv.mylog.DFprintf("***kv.doAppend: group: %v, kvserver: %v, ClientId: %v, SeqId from %v to %v\n", kv.gid, kv.me, opMsg.ClientId, curSeqId, opMsg.SeqId)
		
		kv.clientId2seqId[opMsg.ClientId] = opMsg.SeqId
	}


	return OK
}

func (kv *ShardKV) doUpdate(opMsg Op){
	// if opMsg.Config.Num == kv.config.Num+1 {
	if opMsg.Config.Num == kv.config.Num+1 && kv.readyToUpdate(){
		kv.lastConfig = kv.config
		kv.config = opMsg.Config
		moveInShards := make(map[int]int)
		kv.mylog.DFprintf("***kv.doUpdate: group: %v, kvserver: %v,\n lastConfig: %+v,\n config: %+v,\n moveInShards: %+v\n kv.shardState: %+v\n", kv.gid, kv.me, 
		kv.lastConfig, kv.config, moveInShards, kv.shardState)

		for i := 0; i < shardctrler.NShards; i++{
			if kv.lastConfig.Shards[i] == kv.gid && kv.config.Shards[i] != kv.gid{ 
				// need move out 
				kv.shardState[i] = DEL
				
			}
			if kv.lastConfig.Shards[i] != kv.gid && kv.config.Shards[i] == kv.gid{ 
				// need move in
				if kv.lastConfig.Shards[i] == 0{
					// if get data from group 0 which is not exist
					kv.shardState[i] = SERVED
				}else{
					moveInShards[i] = kv.lastConfig.Shards[i]
					kv.shardState[i] = NOTREADY
				}
				
			}
		}
		kv.moveInShards = moveInShards

		kv.mylog.DFprintf("***kv.doUpdate-: group: %v, kvserver: %v,\n lastConfig: %+v,\n config: %+v,\n moveInShards: %+v\n kv.shardState: %+v\n", kv.gid, kv.me, 
		kv.lastConfig, kv.config, moveInShards, kv.shardState)

	}
}
func (kv *ShardKV) doMoveShard(opMsg Op){
	if opMsg.ConfigNum != kv.config.Num{
		kv.mylog.DFprintf("***kv.doMoveShard: group: %v, kvserver: %v, opMsg.ConfigNum(%v) - kv.config.Num(%v)\n opMsg: %+v\n", 
		kv.gid, kv.me, opMsg.ConfigNum, kv.config.Num, opMsg)
		return 
	}
	kv.mylog.DFprintf("***kv.doMoveShard: group: %v, kvserver: %v, opMsg: %+v,\n kv.dataset: %+v\n kv.moveInShards: %+v,\n kv.shardState:%+v\n", 
	kv.gid, kv.me, opMsg, kv.dataset, kv.moveInShards, kv.shardState)

	// if kv.shardState[opMsg.Shard] == SERVED{
	// 	// should not change
	// 	return 
	// }
	 
	if _, ok := kv.moveInShards[opMsg.Shard]; ok{

		// just do once, because those keys belong to the same shard 
		// delete(kv.moveInShards, opMsg.Shard)
		if kv.shardState[opMsg.Shard] == NOTREADY{
			for key, value := range opMsg.ShardData{
				kv.dataset[key] = value
			}
			// move clientid2seqid
			for clientId, seqId := range opMsg.ClientId2seqId{
				if id, ok := kv.clientId2seqId[clientId]; !ok || id < seqId{
					kv.clientId2seqId[clientId] = seqId
				} 
			}
			kv.shardState[opMsg.Shard] = READY
		}
	}
	kv.mylog.DFprintf("***kv.doMoveShard-: group: %v, kvserver: %v, opMsg: %+v,\n kv.dataset: %+v\n kv.moveInShards: %+v,\n kv.shardState:%+v\n", 
	kv.gid, kv.me, opMsg, kv.dataset, kv.moveInShards, kv.shardState)

}
func (kv *ShardKV) doDelShard(opMsg Op){
	if opMsg.ConfigNum != kv.config.Num{
		kv.mylog.DFprintf("***kv.doDelShard: group: %v, kvserver: %v, opMsg.ConfigNum(%v) - kv.config.Num(%v)\n opMsg: %+v\n", 
		kv.gid, kv.me, opMsg.ConfigNum, kv.config.Num, opMsg)
		return 
	}
	kv.mylog.DFprintf("***kv.doDelShard: group: %v, kvserver: %v, opMsg: %+v,\n kv.dataset: %+v\n kv.moveInShards: %+v,\n kv.shardState:%+v\n", 
	kv.gid, kv.me, opMsg, kv.dataset, kv.moveInShards, kv.shardState)

	switch kv.shardState[opMsg.Shard]{
	case DEL:
		var keys []string
		for key, _ := range kv.dataset{
			keys = append(keys, key)
		}
		for _, key := range keys{
			if key2shard(key) == opMsg.Shard{
				delete(kv.dataset, key)
			}
		}
		kv.shardState[opMsg.Shard] = NOTSERVED
	case READY:
		delete(kv.moveInShards, opMsg.Shard)
		kv.shardState[opMsg.Shard] = SERVED
	}
	
	kv.mylog.DFprintf("***kv.doDelShard-: group: %v, kvserver: %v, opMsg: %+v,\n kv.dataset: %+v\n kv.moveInShards: %+v,\n kv.shardState:%+v\n", 
	kv.gid, kv.me, opMsg, kv.dataset, kv.moveInShards, kv.shardState)
	
}

func (kv *ShardKV) snapshot(lastApplied int){
	interval := 1
	if kv.maxraftstate > 0 && (lastApplied+1) % interval==0{
		w := new(bytes.Buffer)
		e := labgob.NewEncoder(w)
		// v := kv.dataset
		e.Encode(kv.dataset)
		e.Encode(kv.clientId2seqId)
		e.Encode(kv.shardState)
		e.Encode(kv.moveInShards)
		e.Encode(kv.lastConfig)
		e.Encode(kv.config)
		// kv.mu.Unlock()
		kv.rf.Snapshot(lastApplied, w.Bytes())
		// kv.mu.Lock()
	
		kv.mylog.DFprintf(
		"***kv.snapshot: group: %v, kvserver: %v,\n dataset: %v,\n clientId2seqId: %v,\n shardState: %v,\n kv.moveInShards %v,\n lastConfig: %v,\n config: %v,\n lastApplied: %v\n",
		kv.gid, kv.me, kv.dataset, kv.clientId2seqId, kv.shardState, kv.moveInShards, kv.lastConfig, kv.config, lastApplied)
	
	}

}
func (kv *ShardKV) waitApply(){
	// 额外开一个线程，接受applyCh
	// 根据index向相应的chan发送信号
	lastApplied := 0

	for kv.killed()==false{
		select{
		case msg := <- kv.applyCh:
			// kv.mylog.DFprintf("*kv.waitApply: group: %v, kvserver: %v, CommandIndex: %v, opMsg: %+v\n", kv.me, msg.CommandIndex, msg.Command)
			kv.mylog.DFprintf("***kv.waitApply: group: %v, kvserver: %v, Msg: %+v\n lastapplied: %v\n", kv.gid, kv.me, msg, lastApplied)
			if msg.SnapshotValid{
				kv.mu.Lock()
				if kv.rf.CondInstallSnapshot(msg.SnapshotTerm,
					msg.SnapshotIndex, msg.Snapshot) {
					
					kv.dataset = make(map[string]string)
					r := bytes.NewBuffer(msg.Snapshot)
					d := labgob.NewDecoder(r)
					var dataset map[string]string
					var clientId2seqId map[int64]int64
					var shardState map[int]ShardState
					var moveInShards map[int]int
					var lastConfig shardctrler.Config
					var config shardctrler.Config

					if d.Decode(&dataset) != nil ||
					   d.Decode(&clientId2seqId) != nil ||
					   d.Decode(&shardState) != nil ||
					   d.Decode(&moveInShards) != nil ||
					   d.Decode(&lastConfig) != nil ||
	   				   d.Decode(&config) != nil{
						kv.mylog.DFprintf("***kv.waitApply: group: %v, kvserver: %v, decode error\n", kv.gid, kv.me)
						log.Fatalf("***kv.waitApply: group: %v, kvserver: %v, decode error\n", kv.gid, kv.me)
					}
					kv.dataset = dataset
					kv.clientId2seqId = clientId2seqId
					kv.shardState = shardState
					kv.moveInShards = moveInShards
					kv.lastConfig = lastConfig
					kv.config = config
					lastApplied = msg.SnapshotIndex

					kv.mylog.DFprintf("***kv.waitApply: CondInstallSnapshot: group: %v, kvserver: %v,\n dataset: %v,\n clientId2seqId: %v,\n shardState: %v,\n kv.moveInShards: %v,\n lastConfig: %v,\n config: %v\n",
					kv.gid, kv.me, kv.dataset, kv.clientId2seqId, kv.shardState, kv.moveInShards, kv.lastConfig, kv.config)
			
				}
				kv.mu.Unlock()

			}else if msg.CommandValid && msg.CommandIndex > lastApplied{
				// kv.mylog.DFprintf("***kv.waitApply: group: %v, kvserver: %v, test1 Msg: %+v\n", kv.gid, kv.me, msg)
				if msg.Command != nil{
					// kv.mylog.DFprintf("***kv.waitApply: group: %v, kvserver: %v, test2 Msg: %+v\n", kv.gid, kv.me, msg)
					// kv.mylog.DFprintf("***kv.waitApply: test3\n")

					var opMsg Op = msg.Command.(Op)
					kv.mu.Lock()
					kv.mylog.DFprintf("***kv.waitApply: group: %v, kvserver: %v, get opMsg: %+v\n", kv.gid, kv.me, opMsg)
					var reply = ApplyReply{Op: opMsg}
					switch opMsg.Method{
					case "Get":
						reply.Err = ""
					case "Put":
						reply.Err = kv.doPut(opMsg)
					case "Append":
						reply.Err = kv.doAppend(opMsg)
					case "Update":
						reply.Err = ""
						kv.doUpdate(opMsg)
					case "MoveShard":
						reply.Err = ""
						kv.doMoveShard(opMsg)
					case "DelShard":
						reply.Err = ""
						kv.doDelShard(opMsg)
					case "Empty":
						reply.Err = ""
						// do nothing
					}
		
					lastApplied = msg.CommandIndex

					kv.snapshot(lastApplied)
					// if kv.maxraftstate > 0 && (msg.CommandIndex+1) % 1 == 0{

					// 	w := new(bytes.Buffer)
					// 	e := labgob.NewEncoder(w)
					// 	// v := kv.dataset
					// 	e.Encode(kv.dataset)
					// 	e.Encode(kv.clientId2seqId)
					// 	e.Encode(kv.shardState)
					// 	e.Encode(kv.moveInShards)
					// 	e.Encode(kv.lastConfig)
					// 	e.Encode(kv.config)
					// 	// kv.mu.Unlock()
					// 	kv.rf.Snapshot(msg.CommandIndex, w.Bytes())
					// 	// kv.mu.Lock()

					// 	kv.mylog.DFprintf(
					// 	"***kv.waitApply: Snapshot: group: %v, kvserver: %v,\n dataset: %v,\n clientId2seqId: %v,\n shardState: %v,\n kv.moveInShards %v,\n lastConfig: %v,\n config: %v,\n lastApplied: %v\n",
					// 	kv.gid, kv.me, kv.dataset, kv.clientId2seqId, kv.shardState, kv.moveInShards, kv.lastConfig, kv.config, lastApplied)

					// }
					kv.mu.Unlock()
					if _, isLeader := kv.rf.GetState(); isLeader {
						ch := kv.getAgreeChs(msg.CommandIndex)
						ch <- reply
					}
					
	
				}
			}
		case <- kv.stopCh:
			
		}
	}
	kv.mylog.DFprintf("***waitApply(): end, kvserver: %v\n", kv.me)
	// close(kv.applyCh)
	
}

func (kv *ShardKV) readyToUpdate()bool{
	// kv.mu.Lock()
	// defer kv.mu.Unlock()
	for _, state := range kv.shardState{
		if state != SERVED && state != NOTSERVED{
			return false
		}
	}
	return true

}

func (kv *ShardKV)getKeys(dataset map[string]string) []string{
	kv.mu.Lock()
	defer kv.mu.Unlock()
	keys := []string{}
	for k, _ := range kv.dataset{
		keys = append(keys, k)
	}
	return keys 
}

func (kv *ShardKV) updateConfig(){
	for kv.killed() == false{
		if _, isleader := kv.rf.GetState(); isleader{
			kv.mu.Lock()
			nextConfigNum := kv.config.Num+1
			for !kv.readyToUpdate(){
				kv.mu.Unlock()
				time.Sleep(10*time.Millisecond)
				kv.mu.Lock()
			}
			kv.mu.Unlock()

			config := kv.mck.Query(nextConfigNum)		
			kv.mylog.DFprintf("***updateConfig(): group: %v, kvserver: %v, get config: %+v\n kv.config :%+v\n kv.shardState: %+v\n keys: %v\n",
			 kv.gid, kv.me, config, kv.config, kv.shardState, kv.getKeys(kv.dataset))
			kv.mu.Lock()
			if config.Num == kv.config.Num+1{
				kv.mylog.DFprintf("***updateConfig(): group: %v, kvserver: %v, kv.configNum: %v, send new config: %+v\n", kv.gid, kv.me, kv.config.Num, config)

				kv.mu.Unlock()
				op := Op{
					Method: "Update",
					Config: config,
				}
				index, _, isLeader := kv.rf.Start(op)

				if isLeader{
					ch := kv.getAgreeChs(index)
					select{
					case <- ch:
						close(ch)
					case <- time.After(500*time.Millisecond): // 500ms
						// leader change
					}
				}
			}else{
				kv.mu.Unlock()
			}

		}
		time.Sleep(50*time.Millisecond)
	}
}

func (kv *ShardKV) moveShard(){
	wg := &sync.WaitGroup{}
	for kv.killed() == false{
		if _, isleader := kv.rf.GetState(); isleader{
			// if shard should be moved
			kv.mu.Lock()
			kv.mylog.DFprintf("***moveShard():group: %v, kvserver: %v, kv.moveInShards: %+v,\n kv.shardState: %+v\n", 
			kv.gid, kv.me, kv.moveInShards, kv.shardState)
			for shard, state := range kv.shardState{
				if state == NOTREADY{
					g := kv.moveInShards[shard]
					wg.Add(1)
					go kv.callMoveShardData(shard, g, wg)
				}
			}
			kv.mylog.DFprintf("***moveShard():group: %v, kvserver: %v, is waiting kv.moveInShards: %+v,\n kv.shardState: %+v\n", 
			kv.gid, kv.me, kv.moveInShards, kv.shardState)

			kv.mu.Unlock()
		}

		wg.Wait()
		time.Sleep(50*time.Millisecond)
	}
}


func (kv *ShardKV) callMoveShardData(shard int, gid int, wg *sync.WaitGroup){
	defer wg.Done()
	kv.mu.Lock()
	args := MoveShardDataArgs{}
	args.ConfigNum = kv.config.Num
	args.Shard = shard
	kv.mylog.DFprintf("***callMoveShardData():group: %v, kvserver: %v, args: %+v,\n kv.lastConfig.Groups[gid]: %v\n", 
	kv.gid, kv.me, args, kv.lastConfig.Groups[gid])

	if servers, ok := kv.lastConfig.Groups[gid]; ok {
		kv.mu.Unlock()
		for{
			for si := 0; si < len(servers); si++ {
				srv := kv.make_end(servers[si])
				reply := MoveShardDataReply{}
				ok := srv.Call("ShardKV.MoveShardData", &args, &reply)
				kv.mylog.DFprintf("***callMoveShardData():group: %v, kvserver: %v, ok: %v,\n reply: %+v,\n args: %+v\n", kv.gid, kv.me, ok, reply, args)

				if ok && reply.Err == OK{
					kv.mylog.DFprintf("***callMoveShardData():group: %v, kvserver: %v, reply: %+v,\n args: %+v\n", kv.gid, kv.me, reply, args)
	
					op := Op{
						Method: "MoveShard",
						Shard: shard,
						ShardData: reply.ShardData,
						ClientId2seqId: reply.ClientId2seqId,
						ConfigNum: reply.ConfigNum,
					}
					index, _, isLeader := kv.rf.Start(op)
	
					if isLeader{
						ch := kv.getAgreeChs(index)
						select{
						// case opMsg = <- ch:
						case <-ch:
							close(ch)
							
						case <- time.After(500*time.Millisecond): // 500ms
						
						}	
					}
					return
				}
				if ok && reply.Err == ErrNotReady{
					kv.mylog.DFprintf("***callMoveShardData():group: %v, kvserver: %v, reply: %+v,\n args: %+v\n", kv.gid, kv.me, reply, args)

					time.Sleep(10*time.Millisecond)
					break
				}
				// if ok && (reply.Err == ErrWrongGroup) {
				// 	kv.mylog.DFprintf("***callMoveShardData():group: %v, kvserver: %v, reply: %+v,\n args: %+v\n", kv.gid, kv.me, reply, args)
				// 	return 
				// }

			}

		}
	}else{
		kv.mu.Unlock()
	}
}
func (kv *ShardKV) MoveShardData(args *MoveShardDataArgs, reply *MoveShardDataReply){
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if _, isleader := kv.rf.GetState(); !isleader{
		reply.Err = ErrWrongLeader
		return 
	}
	kv.mylog.DFprintf("***MoveShardData():group: %v, kvserver: %v, kv.configNum: %v, args: %+v\n kv.shardState: %+v\n",
	kv.gid, kv.me, kv.config.Num, args, kv.shardState)
	if args.ConfigNum > kv.config.Num{
		reply.Err = ErrNotReady
		return 
	}
	if args.ConfigNum < kv.config.Num || kv.shardState[args.Shard] != DEL{

		reply.Err = OK
		return 
	}

	shardData := make(map[string]string)
	for key, value := range kv.dataset{
		if key2shard(key) == args.Shard{
			shardData[key] = value
		}
	}
	clientId2seqId := make(map[int64]int64)
	for clientId, seqId := range kv.clientId2seqId{
		clientId2seqId[clientId] = seqId
	}
	
	reply.ShardData = shardData
	reply.ClientId2seqId = clientId2seqId
	reply.ConfigNum = args.ConfigNum
	reply.Err = OK
	
}

func (kv *ShardKV) deleteShard(){
	wg := &sync.WaitGroup{}
	for kv.killed() == false{
		if _, isleader := kv.rf.GetState(); isleader{
			// if shard should be moved
			kv.mu.Lock()
			kv.mylog.DFprintf("***deleteShard():group: %v, kvserver: %v, kv.moveInShards: %+v,\n kv.shardState: %+v\n", 
			kv.gid, kv.me, kv.moveInShards, kv.shardState)
			for shard, state := range kv.shardState{
				if state == READY{
					g := kv.moveInShards[shard]
					wg.Add(1)
					go kv.callDeleteShardData(shard, g, wg)
				}
			}
			kv.mylog.DFprintf("***deleteShard():group: %v, kvserver: %v, is waiting. kv.moveInShards: %+v,\n kv.shardState: %+v\n", 
			kv.gid, kv.me, kv.moveInShards, kv.shardState)
			kv.mu.Unlock()
		}

		wg.Wait()
		time.Sleep(50*time.Millisecond)
	}

}
func (kv *ShardKV) callDeleteShardData(shard int, gid int, wg *sync.WaitGroup){
	defer wg.Done()
	kv.mu.Lock()
	args := DeleteShardDataArgs{}
	args.ConfigNum = kv.config.Num
	args.Shard = shard

	if servers, ok := kv.lastConfig.Groups[gid]; ok {
		kv.mu.Unlock()
		for{
			for si := 0; si < len(servers); si++ {
				srv := kv.make_end(servers[si])
				reply := DeleteShardDataReply{}
				ok := srv.Call("ShardKV.DeleteShardData", &args, &reply)
				// kv.mylog.DFprintf("***callMoveShardData():group: %v, kvserver: %v, reply: %+v,\n args: %+v\n", kv.gid, kv.me, reply, args)

				if ok && reply.Err == OK{
					kv.mylog.DFprintf("***callDeleteShardData():group: %v, kvserver: %v, reply: %+v,\n args: %+v\n", kv.gid, kv.me, reply, args)
	
					op := Op{
						Method: "DelShard",
						Shard: shard,
						ConfigNum: args.ConfigNum,
					}
					index, _, isLeader := kv.rf.Start(op)
	
					if isLeader{
						ch := kv.getAgreeChs(index)
						select{
						case <-ch:
							close(ch)
							
						case <- time.After(500*time.Millisecond): // 500ms
						
						}	
					}
					return
				}
				if ok && reply.Err == ErrNotReady{
					kv.mylog.DFprintf("***callDeleteShardData():group: %v, kvserver: %v, reply: %+v,\n args: %+v\n", kv.gid, kv.me, reply, args)
					time.Sleep(10 * time.Millisecond)
					break
				}
			}

		}
	}else{
		kv.mu.Unlock()
	}
	
}

func (kv *ShardKV) DeleteShardData(args *DeleteShardDataArgs, reply *DeleteShardDataReply){
	
	if _, isleader := kv.rf.GetState(); !isleader{
		reply.Err = ErrWrongLeader
		return 
	}
	kv.mu.Lock()

	kv.mylog.DFprintf("***DeleteShardData():group: %v, kvserver: %v, kv.configNum: %v, args: %+v\n", kv.gid, kv.me, kv.config.Num, args)
	if args.ConfigNum < kv.config.Num{
		reply.Err = OK
		kv.mu.Unlock()
		return 
	}
	if args.ConfigNum > kv.config.Num{
		reply.Err = ErrNotReady
		kv.mu.Unlock()
		return 
	}
	kv.mu.Unlock()
	op := Op{
		Method: "DelShard",
		Shard: args.Shard,
		ConfigNum: args.ConfigNum,
	}

	index, _, isLeader := kv.rf.Start(op)
	
	if isLeader{
		ch := kv.getAgreeChs(index)
		// var applyMsg ApplyReply

		select{
		// case applyMsg = <- ch:
		case <- ch:
			// reply.Err = applyMsg.Err
			reply.Err = OK
			close(ch)
			
		case <- time.After(500*time.Millisecond): // 500ms
		
		}	
	}

}

func (kv *ShardKV) addEmptyEntry(){
	for kv.killed() == false{
		isLeader, hasTerm := kv.rf.HasCurrentTerm()
		kv.mylog.DFprintf("***addEmptyEntry(): group: %v, kvserver: %v, isLeader: %v, hasTerm: %v\n", 
			kv.gid, kv.me, isLeader, hasTerm)
		
		if isLeader && !hasTerm{
			op := Op{
				Method: "Empty",
			}
			
			index, _, isLeader := kv.rf.Start(op)
			kv.mylog.DFprintf("***addEmptyEntry(): group: %v, kvserver: %v, index: %v, isLeader: %v\n", 
			kv.gid, kv.me, index, isLeader)

			if isLeader{
				ch := kv.getAgreeChs(index)
				// var applyMsg ApplyReply
		
				select{
				// case applyMsg = <- ch:
				case <- ch:
					kv.mylog.DFprintf("***addEmptyEntry(): group: %v, kvserver: %v, index: %v, get\n", 
					kv.gid, kv.me, index)

					close(ch)
					
				case <- time.After(500*time.Millisecond): // 500ms
				
				}	
			}
		}
		time.Sleep(50*time.Millisecond)
	}
	kv.mylog.DFprintf("***addEmptyEntry(): group: %v, kvserver: %v done\n", 
	kv.gid, kv.me)
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
	kv.agreeChs = make(map[int]chan ApplyReply)
	kv.stopCh = make(chan struct{})
	
	kv.mck = shardctrler.MakeClerk(ctrlers, mylog)
	kv.shardState = make(map[int]ShardState)

	kv.readDataset(persister.ReadSnapshot())


	go kv.waitApply()
	go kv.updateConfig()
	go kv.moveShard()
	go kv.deleteShard()
	go kv.addEmptyEntry()
	kv.mylog.DFprintf("***StartKVServer(): me: %v\n", me)
	return kv
}
