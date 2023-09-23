
follower
timeout结束，转变为candiate

candiate
term ++ 
votefor = me
向follower/candiate/ leader 发送 requestvote
	对follower timeout重置
	对于可能存在 有多个candiate, 过期leader重新上线的情况
		对每一个raft, 若  args.term>term, 则将其转化为follower, votefor = -1
		若过期leader接收requestvote，即args.term>term，则自动转换为follower, votefor = -1
		//若其他candiate接收requestvote，若过期，即args.term>term，则自动转换为follower, votefor = -1

	收集requestvote返回的信号，
		检测自身是否还是candiate（有可能被leader或其他candiate打回follower）
		若仍是candiate，
			若接收到reply.grantvote=true, 则count++
			若count 一直少于半数，则一直等待，直到满足要求，或者timeout或者其他raft打回follower
	requestvote 投票的条件:
		1.仍未投票，2.对方log比我方新(term大，或term相同但log长)

leader 
向 follower/candiate 发送 heartbeats, 重置timeout
	若leader outdate，即reply.term > leader.term , 转化为follower
	若args.term >= raft.term， 将其打回为follower,  更新reaft.term
	

