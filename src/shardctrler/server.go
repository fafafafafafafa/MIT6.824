package shardctrler


import "6.824/raft"
import "6.824/labrpc"
import "sync"
import "6.824/labgob"
import "time"
import "sync/atomic"
import "sort"

type ShardCtrler struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg

	// Your data here.
	dead int32 // set by Kill()
	configs []Config // indexed by config num
	clientId2seqId	map[int64]int64 
	agreeChs map[int]chan Op
	stopCh chan struct{}

	// gid2count map[int]int
	// gids []int

	mylog *raft.Mylog
}


type Op struct {
	// Your data here.
	Method string //  "Join", "Leave", "Move", "Query"
	ClientId int64// 全局标识，初始化时随机生成，用于标识客户端
	SeqId int64 // 最新的序列码，随着请求自增，用于排除 重复和过期的请求

	Servers map[int][]string
	GIDs []int
	Shard int
	GID   int
	Num int 
}

func (sc *ShardCtrler) getAgreeChs(index int) chan Op{
	sc.mu.Lock()
	defer sc.mu.Unlock()
	ch, ok := sc.agreeChs[index]
	if !ok{
		ch = make(chan Op, 1)
		sc.agreeChs[index] = ch
	}
	return ch
}

func (sc *ShardCtrler) Join(args *JoinArgs, reply *JoinReply) {
	// Your code here.
	// sc.mu.Lock()
	// defer sc.mu.Unlock()
	
	op := Op{
		Method: "Join",
		ClientId: args.ClientId,
		SeqId: args.SeqId,
		Servers: args.Servers,
	}
	index, _, isleader := sc.rf.Start(op)
	if isleader{
		ch := sc.getAgreeChs(index)
		sc.mylog.DFprintf("**sc.Join: ShardCtrler: %v, get agreeChs[%v]\n", sc.me, index)
		// var opMsg Op
		select{
		// case opMsg = <- ch:
		case <-ch:
			reply.WrongLeader = false
			close(ch)
			
		case <- time.After(500*time.Millisecond): // 500ms
			sc.mylog.DFprintf("**sc.Join: ShardCtrler: %v, get opMsg timeout\n", sc.me)

			reply.WrongLeader = true
		}
	}else{
		reply.WrongLeader = true
	}
}

func (sc *ShardCtrler) Leave(args *LeaveArgs, reply *LeaveReply) {
	// Your code here.
	op := Op{
		Method: "Leave",
		ClientId: args.ClientId,
		SeqId: args.SeqId,
		GIDs: args.GIDs,

	}
	index, _, isleader := sc.rf.Start(op)
	if isleader{
		ch := sc.getAgreeChs(index)
		sc.mylog.DFprintf("**sc.Leave: ShardCtrler: %v, get agreeChs[%v]\n", sc.me, index)
		// var opMsg Op
		select{
		// case opMsg = <- ch:
		case <-ch:
			reply.WrongLeader = false
			close(ch)
			
		case <- time.After(500*time.Millisecond): // 500ms
			sc.mylog.DFprintf("**sc.Leave: ShardCtrler: %v, get opMsg timeout\n", sc.me)

			reply.WrongLeader = true
		}
	}else{
		reply.WrongLeader = true
	}
}

func (sc *ShardCtrler) Move(args *MoveArgs, reply *MoveReply) {
	// Your code here.
	op := Op{
		Method: "Move",
		ClientId: args.ClientId,
		SeqId: args.SeqId,
		Shard: args.Shard,
		GID: args.GID,
	}
	index, _, isleader := sc.rf.Start(op)
	if isleader{
		ch := sc.getAgreeChs(index)
		sc.mylog.DFprintf("**sc.Move: ShardCtrler: %v, get agreeChs[%v]\n", sc.me, index)
		// var opMsg Op
		select{
		// case opMsg = <- ch:
		case <-ch:
			reply.WrongLeader = false
			close(ch)
			
		case <- time.After(500*time.Millisecond): // 500ms
			sc.mylog.DFprintf("**sc.Move: ShardCtrler: %v, get opMsg timeout\n", sc.me)

			reply.WrongLeader = true
		}
	}else{
		reply.WrongLeader = true
	}
}

func (sc *ShardCtrler) Query(args *QueryArgs, reply *QueryReply) {
	// Your code here.
	op := Op{
		Method: "Query",
		ClientId: args.ClientId,
		SeqId: args.SeqId,
		Num: args.Num,

	}
	index, _, isleader := sc.rf.Start(op)
	if isleader{
		ch := sc.getAgreeChs(index)
		sc.mylog.DFprintf("**sc.Query: ShardCtrler: %v, get agreeChs[%v]\n", sc.me, index)
		var opMsg Op
		select{
		case opMsg = <- ch:
			reply.WrongLeader = false
			close(ch)
			sc.mu.Lock()
			if opMsg.Num == -1 || opMsg.Num >= len(sc.configs){
				reply.Config = sc.configs[len(sc.configs)-1]
			}else{
				reply.Config = sc.configs[opMsg.Num]
			}
			sc.mu.Unlock()
		case <- time.After(500*time.Millisecond): // 500ms
			sc.mylog.DFprintf("**sc.Query: ShardCtrler: %v, get opMsg timeout\n", sc.me)

			reply.WrongLeader = true
		}
	}else{
		reply.WrongLeader = true
	}
}
func CopyConfig(cfg Config) Config{
	cfg2 := Config{
		Num: cfg.Num,
		Shards: cfg.Shards,
		Groups: make(map[int][]string),
	}
	for key, value := range cfg.Groups{
		cfg2.Groups[key] = value
	}
	return cfg2
}
func getMinMaxGroup(gids []int, gid2shardIdx map[int] []int)(int, int){
	min := 257
	minGid := 0 
	max := 0
	maxGid := 0

	for _, gid := range gids {
		if len(gid2shardIdx[gid]) > max {
			max = len(gid2shardIdx[gid])
			maxGid = gid
		}
		if len(gid2shardIdx[gid]) < min {
			min = len(gid2shardIdx[gid])
			minGid = gid
		}
	}
	return minGid, maxGid


}
func (sc *ShardCtrler) rebalance(config *Config){
	// count
	var gids []int
	for gid, _ := range config.Groups{
		gids = append(gids, gid)
	}
	sort.Ints(gids)
	sc.mylog.DFprintf("**rebalance: ShardCtrler: %v, gids: %v\n", sc.me, gids)

	gid2shardIdx := map[int] []int{}
	
	for i, gid := range config.Shards {
		gid2shardIdx[gid] = append(gid2shardIdx[gid], i)
	}
	sc.mylog.DFprintf("**rebalance: with zero, ShardCtrler: %v, gid2shardIdx: %+v\n", sc.me, gid2shardIdx)
	if _, ok := gid2shardIdx[0]; ok{ 
		// have no-distributed shards
		for i := 0; i < len(gid2shardIdx[0]); i++{
			minGid, _ := getMinMaxGroup(gids, gid2shardIdx)
			if minGid == 0{
				// no group
				break
			}
			config.Shards[gid2shardIdx[0][i]] = minGid

			gid2shardIdx[minGid] = append(gid2shardIdx[minGid], gid2shardIdx[0][i])

		}
		delete(gid2shardIdx, 0)
	}
	sc.mylog.DFprintf("**rebalance: no zero, ShardCtrler: %v, gid2shardIdx: %+v\n", sc.me, gid2shardIdx)

	for{
		minGid, maxGid := getMinMaxGroup(gids, gid2shardIdx)
		if minGid == 0 || maxGid == 0 || len(gid2shardIdx[minGid])+1 >= len(gid2shardIdx[maxGid]){
			break
		}

		// move
		sc.mylog.DFprintf("**rebalance: sc: %v, move before: maxGidShardIdx[%v] %v, minGidShardIdx[%v] %v\n", 
		sc.me, maxGid, gid2shardIdx[maxGid], minGid, gid2shardIdx[minGid])

		i := gid2shardIdx[maxGid][0]
		gid2shardIdx[minGid] = append(gid2shardIdx[minGid], i)
		config.Shards[i] = minGid
		gid2shardIdx[maxGid] = gid2shardIdx[maxGid][1:]

		sc.mylog.DFprintf("**rebalance: sc: %v, move after: maxGidShardIdx[%v] %v, minGidShardIdx[%v] %v\n", 
		sc.me, maxGid, gid2shardIdx[maxGid], minGid, gid2shardIdx[minGid])

	}
	
}

func (sc *ShardCtrler) waitApply(){
	// 额外开一个线程，接受applyCh
	// 根据index向相应的chan发送信号
	lastApplied := 0

	for sc.killed()==false{
		select{
		case msg := <- sc.applyCh:			
			sc.mylog.DFprintf("**sc.waitApply: ShardCtrler: %v, Msg: %+v\n", sc.me, msg)
			if msg.SnapshotValid{
				sc.mu.Lock()
				if sc.rf.CondInstallSnapshot(msg.SnapshotTerm,
					msg.SnapshotIndex, msg.Snapshot) {

				}
				sc.mu.Unlock()

			}else if msg.CommandValid && msg.CommandIndex > lastApplied{
				if msg.Command != nil{
					sc.mu.Lock()
					var opMsg Op = msg.Command.(Op)
					sc.mylog.DFprintf("**sc.waitApply: ShardCtrler: %v, get opMsg: %+v\n", sc.me, opMsg)
					curSeqId, ok := sc.clientId2seqId[opMsg.ClientId]
					sc.mylog.DFprintf("**sc.waitApply: ShardCtrler: %v, ok: %v, opMsg.SeqId(%v)-curSeqId(%v)\n", sc.me, ok, opMsg.SeqId, curSeqId)
					if !ok || opMsg.SeqId > curSeqId{
						// only handle new request
						configNum := len(sc.configs)
						lastConfig := sc.configs[configNum-1]
						newConfig := CopyConfig(lastConfig)
						sc.mylog.DFprintf("**sc.waitApply: ShardCtrler: %v, newConfig.Groups: %+v\n", sc.me, newConfig.Groups)
						newConfig.Num = configNum

						switch opMsg.Method{
						case "Join":
							for gid, servers := range opMsg.Servers{
								if _, ok := lastConfig.Groups[gid]; !ok{
									newConfig.Groups[gid] = servers
								}
							}
							// rebalance
							sc.rebalance(&newConfig)
							sc.configs = append(sc.configs, newConfig)		

						case "Leave":
							for _, gid := range opMsg.GIDs{
								if _, ok := lastConfig.Groups[gid]; ok{
									delete(newConfig.Groups, gid)
									for i, ggid := range lastConfig.Shards{
										if ggid == gid{
											newConfig.Shards[i] = 0
										}
									}
								}
							}
							sc.rebalance(&newConfig)
							sc.configs = append(sc.configs, newConfig)		
						case "Move":
							newConfig.Shards[opMsg.Shard] = opMsg.GID 
							sc.configs = append(sc.configs, newConfig)		
						}
						// update clientId2seqId
						sc.mylog.DFprintf("**sc.waitApply: ShardCtrler: %v, ClientId: %v, SeqId from %v to %v\n", sc.me, opMsg.ClientId, curSeqId, opMsg.SeqId)
		
						sc.clientId2seqId[opMsg.ClientId] = opMsg.SeqId
						
					}
					lastApplied = msg.CommandIndex

					if (msg.CommandIndex+1) % 10 == 0{
					
				

					}
					sc.mu.Unlock()
					if _, isLeader := sc.rf.GetState(); isLeader {
						ch := sc.getAgreeChs(msg.CommandIndex)
						ch <- opMsg
					}
					
	
				}
			}
		case <- sc.stopCh:
			
		}

 
	}
	sc.mylog.DFprintf("**sc.waitApply(): end, ShardCtrler: %v\n", sc.me)
	// close(kv.applyCh)
	
}


//
// the tester calls Kill() when a ShardCtrler instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//

func (sc *ShardCtrler) killed() bool {
	z := atomic.LoadInt32(&sc.dead)
	return z == 1
}

func (sc *ShardCtrler) Kill() {
	sc.rf.Kill()
	
	// Your code here, if desired.
	atomic.StoreInt32(&sc.dead, 1)
	close(sc.stopCh)
}

// needed by shardkv tester
func (sc *ShardCtrler) Raft() *raft.Raft {
	return sc.rf
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant shardctrler service.
// me is the index of the current server in servers[].
//
func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, mylog *raft.Mylog) *ShardCtrler {
	sc := new(ShardCtrler)
	sc.me = me

	sc.configs = make([]Config, 1)
	sc.configs[0].Groups = map[int][]string{}

	labgob.Register(Op{})
	sc.applyCh = make(chan raft.ApplyMsg)
	sc.rf = raft.Make(servers, me, persister, sc.applyCh, mylog)

	// Your code here.
	sc.clientId2seqId = make(map[int64]int64, 0)
	sc.agreeChs = make(map[int]chan Op)
	sc.stopCh = make(chan struct{})

	sc.mylog = mylog
	go sc.waitApply()
	return sc
}
