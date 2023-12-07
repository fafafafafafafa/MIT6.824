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

	else{
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

8. raft 发送 installSnapshot()
command = nil，所以在 kv.waitApply() 检测 msg.Command != nil
这个在3B中应该已经得到解决，因为首先会判断是否为SnapshotValid，而且
raft中也是从lastIncludedIndex开始发送cmd。


9. duplicate
old2.txt
*kv.waitApply: Append, kvserver: 2, key: 3, not duplicate-----37174
*----------------ShutdownAll------------------37459

readDataset: load kvserver:2  -----38649
have 3:x 3 0 yx 3 1 yx 3 2 yx 3 3 yx 3 4 yx 3 5 yx 3 6 yx 3 7 yx 3 8 yx 3 9 yx 3 10 yx 3 11 yx 3 12 yx 3 13 y 

*----------------ConnectAll------------------38671
*kv.waitApply: Append, kvserver: 2, key: 3, duplicate-----44512

因为raft保存的是多个client发送的cmd，所以snapshot后，可能会重新发送已经应用在kvserver上的cmd
所以应该保存seqId.

10. TestSnapshotUnreliableRecoverConcurrentPartition3B
Fail: 4 missing element x 4 15 y in Append result  
原因：CondInstallSnapshot 不能偷懒全部删除，可能会导致apply无效数据

11. TestSnapshotUnreliableRecoverConcurrentPartitionLinearizable3B
raft.sendPeerAppendEntries 索引非法 
Linearizable3Bpro1.txt
election: win the election raft : rf.me=3, ---------59062

fail: err! raft: 3, prevLogIndex(448)-rf.lastIncludedIndex(409) = 39 ------------------59454

raft: 508 for ; conflictIndex-rf.lastIncludedIndex > 0; conflictIndex--


Linearizable3Bpro1-2.txt
*------------partition: p1([1 3 4 6]), p2([0 2 5])------------62691
election: win the election raft : rf.me=6, ------65157
election: win the election raft : rf.me=6, ------69143
*------------partition: p1([1 3 4 5]), p2([0 2 6])------------79935
heartBeats: raft: 6, nextIndex: [615 589 614 592 592 590 618] 
		matchIndex: [614 588 612 591 591 589 617], len of log: 29, lastIncludedIndex: 589,---------81534
election: win the election raft : rf.me=3, -------81537
heartBeats: raft: 3, nextIndex: [593 593 593 594 593 593 593],
		 matchIndex: [0 0 0 593 0 0 0], len of log: 15, lastIncludedIndex: 579, -------81567
heartBeats: raft: 3, nextIndex: [593 608 593 609 608 608 593], 
		matchIndex: [0 607 0 608 607 607 0], len of log: 10, lastIncludedIndex: 599,-------83060

*------------partition: p1([3 6]), p2([0 1 2 4 5])------------83174
heartBeats: raft: 3, nextIndex: [593 610 593 612 610 610 593],
		 matchIndex: [0 609 0 611 609 609 0], len of log: 3, lastIncludedIndex: 609,-------83227
*kv.waitApply: CondInstallSnapshot, kvserver: 6,  lastApplied: 609 -------------83249
election: win the election raft : rf.me=5 ------------84676
*------------partition: p1([2 3 4]), p2([0 1 5 6])------------84743

fail: err! raft(5) to raft(6), prevLogIndex(619)-rf.lastIncludedIndex(599) = 20 ------------84986

CondInstallSnapshot(), log裁剪错误导致的，详情见博客lab3B问题2.


12. TestSnapshotRecoverManyClients3B testmanyclientpro1-2

*----------------ShutdownAll------------------44401
*StartKVServer(): me: 0 ---------------------------------------45575
*----------------ShutdownAll------------------101071
*StartKVServer(): me: 0 ---------------------------------------102295
*----------------ShutdownAll------------------155396
*StartKVServer(): me: 0 ---------------------------------------156616
election: timeout raft : rf.me=2, rf.term=6 ---160787
election: timeout raft : rf.me=4, rf.term=7, ---160917
election: win the election raft : rf.me=2, rf.term=6,  ----160958
election: win the election raft : rf.me=4, rf.term=7,  ----161165
heartBeats: raft: 4, nextIndex: [0 0 0 0 1579], matchIndex: [0 0 0 0 1578], len of log: 30, lastIncludedIndex: 1549 --161211

election: timeout raft : rf.me=1, rf.term=8,  ---453140
election: timeout raft : rf.me=1, rf.term=9，---453314
election: timeout raft : rf.me=4, rf.term=9，---453332
election: timeout raft : rf.me=3, rf.term=10, ---453401
election: win the election raft : rf.me=3, rf.term=10 ----453413

testmanyclientpro1.txt

*----------------ShutdownAll------------------40344
*----------------ShutdownAll------------------92848
*----------------ShutdownAll------------------146847

*StartKVServer(): me: 0 ----------- 148069
readPersist: load raft:0 , rf.currentTerm: 8, rf.votedFor: -1, len of rf.log: 11, rf.lastIncludedIndex: 1459, rf.lastIncludedTerm: 8, 
readPersist: load raft:1 , rf.currentTerm: 8, rf.votedFor: 1, len of rf.log: 22, rf.lastIncludedIndex: 1449,
readPersist: load raft:2 , rf.currentTerm: 8, rf.votedFor: -1, len of rf.log: 11, rf.lastIncludedIndex: 1459
readPersist: load raft:3 , rf.currentTerm: 8, rf.votedFor: -1, len of rf.log: 11, rf.lastIncludedIndex: 1459,
readPersist: load raft:4 , rf.currentTerm: 8, rf.votedFor: -1, len of rf.log: 11, rf.lastIncludedIndex: 1459,
*StartKVServer(): me: 4 ----------148129

election: win the election raft : rf.me=3, rf.term=9, -----152123
election: timeout raft : rf.me=0, rf.term=10, -----152143
election: win the election raft : rf.me=0, rf.term=10 -----152262
election: timeout raft : rf.me=2, rf.term=11 -----152280
election: timeout raft : rf.me=4, rf.term=12, -----152409
election: timeout raft : rf.me=1, rf.term=13 ------152975
election: win the election raft : rf.me=1, rf.term=13, ------153108
heartBeats: raft: 1, nextIndex: [0 1472 0 0 0], matchIndex: [0 1471 0 0 0], len of log: 23 ----------153195
heartBeats: raft: 1, nextIndex: [1450 1487 1450 1450 1450], matchIndex: [1449 1486 1449 1449 1449], 
len of log: 38, lastIncludedIndex: 1449  ------------------------------------------153287

election: timeout raft : rf.me=0, rf.term=14, ------258547
election: win the election raft : rf.me=0, rf.term=14, ------258707
conflict设置错误，

if (args.PrevLogIndex-rf.lastIncludedIndex) < 0{
	reply.ConflictIndex = rf.lastIncludedIndex // here
}else{
	reply.ConflictIndex = lastEntry.Index
}



