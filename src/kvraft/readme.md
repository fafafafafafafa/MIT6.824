go test -run 3A -timeout 1h/m/s(0 无时间限制) -race -args 1
go test -run 3A -timeout 0 -race -args 1


--------------------------------------3A--------------------------------------
任务需求:
Clerks 向 kvserver发送 Put()/Append()，kvserver 发送给leader(raft)

type Clerk struct {
	servers []*labrpc.ClientEnd
	// You will have to modify this struct.
	clientId int64// 全局标识，初始化时随机生成，用于标识客户端
	seqId int64 // 最新的序列码，随着请求自增，用于排除 重复和过期的请求
	leaderId int // 默认leaderId，初始化时随机生成 

}

func (ck *Clerk ) PutAppend(){
如何找到leader?随机发
向身份为leader的raft发送RPCs(Put,Append,Get)
// 向所有的raft都发送一遍，不需要找leader
	SeqId自增
	for{
		发送RPC请求
		若发送失败，或者 不是leader
			改变可能的leader，重新发送
	} 
}


type KVServer struct{
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()

	maxraftstate int // snapshot if log grows this big
	
	dataset	map[string]string // key-value，存储的数据库
	clientId2seqId	map[int64]int64  // 
	agreeChs map[int]chan Op // 从applyCh接收信息，再通过agreeChs发送
	
}

func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply){
向对应raft通过kv.rf.Start()发送信号，通过applyCh等待信号
若dead，释放线程

kv.rf.Start()
// applyCh 等待信号，不能这样，万一有两个PutAppend()，这时候到底谁先收，这时候就是一个
//一发多收的问题，因此可以根据index，创建chan接收
//接收后，记得关闭chan
	根据method进行操作(Put, Append)
若长时间未等到，
	则reply.Err = ErrWrongLeader
}

func (kv *KVServer) waitApply(){
// 额外开一个线程，接受applyCh
//根据index向相应的chan发送信号
	for{
		
	}

}


func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
为了保证线性一致性，get也从leader发送

}

func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer{

}


type Op struct{
	// 通过Start() 发送给raft
	Method string //  "Put", "Append"
	ClientId int64// 全局标识，初始化时随机生成，用于标识客户端
	SeqId int64 // 最新的序列码，随着请求自增，用于排除 重复和过期的请求
	Key string
	Value string

}
问题：
1. Get/PutAppent 在server生效后，由于网络不稳定，发送失败
Get再次发送，由于SeqId已经更新了，也不会被实施，
server.go 89
解决方法：
opMsg.SeqId >= curSeqId
但是PutAppent不能再次实施，opMsg.SeqId > curSeqId
TestUnreliable3A

2. applier 当raft崩溃后/换leader后，entry会重新apply，但是在waitApply()中会导致阻塞
解决方法：
// lastApplied 也进行保存
ch = make(chan Op, 1) // getAgreeChs()
TestOnePartition3A

3. TestSpeed3A
解决方法：
每当start()，便调用heartbeat()
DFprintf(), persist()导致耗时太多

4. 潜在的datarace1
raft stateTrans()
for i := 0; i < len(rf.peers); i++{
	rf.nextIndex[i] = rf.log.GetLen()+rf.lastIncludedIndex
	rf.matchIndex[i] = 0
}
应该lock

5.潜在的datarace2
func (listLog *ListLog) GetEntriesAfterIndex(index int) []Entry{
	// include index
	listLog.mu.Lock()
	defer listLog.mu.Unlock()
	if index < 0 || index > len(listLog.logEntry)-1{
		return []Entry{}
	}else{
		return listLog.logEntry[index:]
	}
}
这里返回了一个切片，而不是一个深拷贝。
于是，当调用sendPeerAppendEntries()，args.Entries 会在
func (listLog *ListLog) AppendEntries(entries []Entry) {
	listLog.logEntry = append(listLog.logEntry, entries...)

}

和
ok := rf.sendAppendEntries(i, args, reply)
中使用，导致datarace
解决方法:
进行深拷贝，
func (listLog *ListLog) GetEntriesAfterIndex(index int) []Entry{

	}else{
		slice := listLog.logEntry[index:]
		
		reEntries := make([]Entry, len(slice), len(slice))
		copy(reEntries, slice)
		return reEntries
	}
}

6. kvserver 崩溃后，需要重新恢复dataset
raft中lastapplied每次重启后，重0开始发送恢复

7. 网络分区恢复后，存在两个leader
start 可能会向旧leader 发送信息
在 func (rf *Raft) heartBeats()最开头 检测是否为leader