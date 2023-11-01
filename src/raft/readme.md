
--------------------------------2A--------------------------------
follower
timeout结束，转变为candidate
candidate
term ++ 
votefor = me
向follower/candidate/ leader 发送 requestvote

	对于可能存在 有多个candidate, 过期leader重新上线的情况
		对每一个raft, 若  args.term>term, 则将其转化为follower, votefor = -1
		若过期leader接收requestvote，即args.term>term，则自动转换为follower, votefor = -1
		若其他candidate接收requestvote，若过期，即args.term>term，则自动转换为follower, votefor = -1

	收集requestvote返回的信号，
		检测自身是否还是candidate（有可能被leader或其他candidate打回follower）
		若仍是candidate，
			若接收到reply.grantvote=true, 则count++
			若count 一直少于半数，则一直等待，直到满足要求，或者timeout或者其他raft打回follower
	requestvote 投票的条件:
		1.仍未投票，2.对方log比我方新(term大，或term相同但log长)

leader 
向 follower/candidate 发送 heartbeats, 重置timeout
	若leader outdate, 转化为follower
	若遇到candidate，term >= candid
ate， 将其打回为follower

--------------------------------2B--------------------------------

Start() 
判断是否为leader
	否，isLeader=false
	是，添加entry, isLeader=true


LEADER.heartBeats(), 这里rf均为leader
1.更新 rf.nextIndex,  rf.matchIndex
2.遍历followers, 设某follower的下标为i
	根据rf.nextIndex[i]，获取相应的args，包括PrevLogIndex, PrevLogTerm, Entries
	prevLogIndex=rf.nextIndex[i]-1, entries = rf.log[prevLogIndex: ]
3. 开启额外线程进行发送
4.接收反馈
	若reply.Success==true, 
		更新rf.nextIndex[i]=args.PrevLogIndex + len(Entries) + 1
		rf.matchIndex[i]=rf.nextIndex[i]-1
	若false，
		rf.nextIndex[i]=reply.ConflictIndex+1
		如果一次只回档一个entry有点浪费，所以在AppendEntries()中一次性回退到ConflictIndex
	
	更新rf.commitIndex
	唤醒applyCond

applier()
	若被唤醒， 检查是否有entry需要apply，若有则更新rf.lastApplied

AppendEntries(), rf为对应的follower
	令e = rf.log[args.PrevLogIndex]
	
	若args.PrevLogIndex==0 || (e.Index==args.PrevLogIndex && e.Term==args.PrevLogTerm), 说明PrevLogIndex之前的entry均相同
		reply.Success=true

		删除rf.log[PrevLogIndex+1:]
		添加args.Entries
		若args.LeaderCommit>rf.commitIndex,
			更新rf.commitIndex
		唤醒applyCond
	若不匹配，
		reply.ConflictIndex回退至 term不等于e.Term, 有好有坏	
		reply.Success=false


--------------------------------2D--------------------------------
lastIncludedTerm,  lastIncludedIndex 也需要进行持久化
log的下标都需要进行修改，得使用 lastIncludedIndex+ len()-1这种的

snapshot 就是 lastIncludedIndex对应的cmd，直接read就行，不需要自己生成

当leader的lastIncludedIndex>=follower的nextIndex，就直接发InstallSnapshot(), 否则发送AppendEntries
commitIndex>=lastIncludedIndex
server重启后，rf.lastApplied 记得 更新， rf.lastApplied = rf.lastIncludedIndex, 不然，会从1开始重新apply，产生错误


InstallSnapshot()
	leader to follower， 使得滞后follower快速更新
	接收信号， 将snapshot信号通过ApplyMsg 发送给service
	

Snapshot(index int, snapshot []byte)
	snapshot里面只有index 对应的command 
	service to raft, 通知raft剪裁log
	step:
	1. 获取index对应的真实下标, realIndex = index- rf.lastIncludedIndex
	2. 对log进行裁剪，去掉index之前的部分
	3. 保存snapshot（ cmd）
	4. 更新持久化rf.lastIncludedTerm, rf.lastIncludedIndex, log, ....
	

CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool
	InstallSnapshot() 发送 ApplyMsg to service后
	service to server， 告知server， service 将要转换到 传入的snapshot的状态
	
	step:
	1. 若 lastIncludedTerm > rf.lastIncludedTerm || (lastIncludedTerm == rf.lastIncludedTerm && lastIncludedIndex> rf.lastIncludedIndex) 
	则说明snapshot 是新的
		2. 删除旧rf.log
		3. 根据lastIncludedTerm 和 lastIncludedIndex 生成新的 rf.log
		3. return true
	4.否则，return false


