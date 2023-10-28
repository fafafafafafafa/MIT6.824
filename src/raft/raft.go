package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"bytes"
	"sync"
	"sync/atomic"

	"6.824/labgob"
	"6.824/labrpc"
	"time"
	"math/rand"


)


//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}
const (
	FOLLOWER int = 0
	CANDIATE int = 1
	LEADER int = 2
)
//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	identity int	// true if leader 
	startTime int64
	outTime int64
	constCheckTime int
	constAppendTime int 

	currentTerm int	// latest term server has seen (initialized to 0 on first boot, increases monotonically)
	votedFor int	// candidateId that received vote in current term (or null if none)
	log *MapLog	//each entry contains command for state machine, and term when entry was received by leader (first index is 1)

	commitIndex int // index of highest log entry kanown to be committed (initialized to 0, increases monotonically)
	lastApplied int // index of highest log entry applied to state machine (initialized to 0, increases monotonically)

	nextIndex []int // index of the next log entry to send to that server (initialized to leader last log index + 1)
	matchIndex []int // index of highest log entry known to be replicated on server (initialized to 0, increases monotonically)

	//2D
	lastIncludedIndex int
	lastIncludedTerm int

	checkChan chan struct{}
	winElectionChan chan struct{}
	applyCh chan ApplyMsg
	applyCond *sync.Cond

	mylog *Mylog
}


// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).
	rf.mu.Lock()
	term = rf.currentTerm
	if rf.identity == LEADER{
		isleader = true
	}else{
		isleader = false
	}
	
	rf.mu.Unlock()
	
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
	rf.mylog.DFprintf("persist: save raft: %v, rf.currentTerm: %v, rf.votedFor: %v, len of rf.log: %v \n",
	rf.me, rf.currentTerm, rf.votedFor, rf.log.GetLen())
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.log.GetEntriesAfterIndex(0))

	data := w.Bytes()
	rf.persister.SaveRaftState(data)
}


//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }

	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var currentTerm int
	var votedFor int
	var log []Entry
	if d.Decode(&currentTerm) != nil ||
	   d.Decode(&votedFor) != nil||
	   d.Decode(&log) !=nil {
		rf.mylog.DFprintf("readPersist: decode failed! \n")
	} else {
		rf.currentTerm = currentTerm
		rf.votedFor = votedFor
		
		rf.log.DeleteEntriesAfterIndex(0)
		rf.log.AppendEntries(log)
		
		rf.mylog.DFprintf("readPersist: load raft:%v , rf.currentTerm: %v, rf.votedFor: %v, len of rf.log: %v \n", 
		rf.me, rf.currentTerm, rf.votedFor, rf.log.GetLen())
	}
}


//
// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
//
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).
	// lastIncludedIndex := index
	// lastIncludedTerm := rf.log[lastIncludedIndex].Term

	// for i := 1; rf.log[]?
}


//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term int	// candidate's term
	CandidateId	int		// candidate requesting vote
	LastLogIndex int	// index of candidate's last log entry
	LastLogTerm  int 	// term of candidate's last log entry
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term int	// currentTerm, for candidate to update itself
	VoteGranted	bool	// true means candidate received vote
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply){
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	reply.Term = rf.currentTerm
	reply.VoteGranted = false
	
	if args.Term > rf.currentTerm{
		// reset outdate leader or candiate
		// DPrintf("RequestVote: raft %v, has been reset to follower, from term %v to %v, votedfor %v to %v", 
		// rf.me, rf.currentTerm, args.Term, rf.votedFor, -1)
		rf.mylog.DFprintf("RequestVote: raft %v, has been reset to follower, from term %v to %v, votedfor %v to %v, startTime: %v, outTime: %v\n", 
		rf.me, rf.currentTerm, args.Term, rf.votedFor, -1, rf.startTime, rf.outTime)

		rf.currentTerm = args.Term
		rf.identity = FOLLOWER
		rf.votedFor = -1
		rf.persist()
		// rf.outTime = getRandTimeoutMillisecond()
		// rf.startTime = getNowTimeMillisecond()
		
		
		if (rf.votedFor == -1 || rf.votedFor == args.CandidateId){
		
			
			lastEntry := rf.log.GetLastEntry()
			// DPrintf("RequestVote: raft %v, VOTE: args.LastLogTerm = %v, rf.log[l].Term = %v. args.LastLogIndex = %v, rf.log[l].Index = %v\n", 
			// rf.me, args.LastLogTerm, rf.log[l].Term, args.LastLogIndex, rf.log[l].Index)
			rf.mylog.DFprintf("RequestVote: Candiate %v, raft %v, VOTE: args.LastLogTerm = %v, rf.log[l].Term = %v. args.LastLogIndex = %v, rf.log[l].Index = %v\n", 
			args.CandidateId, rf.me, args.LastLogTerm, lastEntry.Term, args.LastLogIndex, lastEntry.Index)
	
			if args.LastLogTerm > lastEntry.Term || (args.LastLogTerm == lastEntry.Term && args.LastLogIndex >= lastEntry.Index){
				// DPrintf("RequestVote: raft %v, term %v, vote for raft %v\n", rf.me, rf.currentTerm, args.CandidateId)
				rf.mylog.DFprintf("RequestVote: raft %v, term %v, vote for raft %v\n", rf.me, rf.currentTerm, args.CandidateId)
				// candiate's log is more up-to-date
				rf.votedFor = args.CandidateId
				// reply.Term = rf.currentTerm
				reply.VoteGranted = true
	
			}
		}
	}
	

}
type AppendEntriesArgs struct{
	Term int	// leader's term
	LeaderId int 
	PrevLogIndex int	// index of log entry immediately preceding new ones 
	PrevLogTerm	int 	// term of prevLogIndex entry
	Entries []Entry	//  log entries to store(empty for heartbeat)
	LeaderCommit int 	// leader's commitIndex

}
type AppendEntriesReply struct{
	Term int	// currentTerm, for leader to update itself
	Success bool	// true if follower contained enrty matching prevLogIndex and prevLogTerm
	ConflictIndex int
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply){
	rf.mu.Lock()
	 
	defer rf.mu.Unlock()

	reply.Term = rf.currentTerm
	reply.Success = false

	if args.Term >= rf.currentTerm{
		//DPrintf("reset raft %v", rf.me)
		
		rf.startTime = getNowTimeMillisecond()
		rf.outTime = getRandTimeoutMillisecond()

		rf.votedFor = -1
		rf.currentTerm = args.Term
		rf.identity = FOLLOWER
		rf.persist()

		lastEntry := rf.log.GetLastEntry()
		rf.mylog.DFprintf("AppendEntries: raft: %v to raft: %v, args.PrevLogIndex: %v to rf: %v, args.PrevLogTerm: %v to rf: %v, len of entries: %v, \n",
		args.LeaderId, rf.me, args.PrevLogIndex, lastEntry.Index, args.PrevLogTerm, lastEntry.Term, rf.log.GetLen())
		

		e, ok := rf.log.GetEntryFromIndex(args.PrevLogIndex);
		if !ok{
			reply.Success = false
			reply.ConflictIndex = lastEntry.Index
			return 
		}

		if args.PrevLogIndex == e.Index && e.Term == args.PrevLogTerm{
			// rf's log is same with leader's log (0 to PrevLogIndex)
			reply.Success = true
		
			lenEntries := len(args.Entries)

			//  rf.log if is appended must be more update than old one 
			if lenEntries > 0 && (args.Entries[lenEntries-1].Term > lastEntry.Term || 
					(args.Entries[lenEntries-1].Term == lastEntry.Term && args.Entries[lenEntries-1].Index > lastEntry.Index)){

				// for i := args.PrevLogIndex + 1; ; i++{
				// 	ee, okk := rf.log[i]
				// 	if okk{
				// 		l := len(rf.log)
				// 		delete(rf.log, i)
				// 		rf.mylog.DFprintf("AppendEntries: cmd(%v) is deleted, raft: %v, index: %v, len of log %v to %v\n", 
				// 		ee.Command, rf.me, ee.Index, l, len(rf.log))
				// 	}else{
				// 		break
				// 	}
				// }
				l := rf.log.GetLen()
				rf.log.DeleteEntriesAfterIndex(args.PrevLogIndex+1)
				rf.mylog.DFprintf("AppendEntries: delete raft: %v, len of log %v to %v \n", rf.me, l, rf.log.GetLen())
				
				// for _, entry := range args.Entries{
				// 	rf.log[entry.Index] = entry
				// 	// DPrintf("AppendEntries: cmd(%v) is appended, raft: %v, index: %v, len of log %v\n", entry.Command, rf.me, entry.Index, len(rf.log))
				// 	rf.mylog.DFprintf("AppendEntries: cmd(%v) is appended, raft: %v, index: %v, len of log %v\n", entry.Command, rf.me, entry.Index, len(rf.log))
				// }
				l = rf.log.GetLen()
				rf.log.AppendEntries(args.Entries)
				rf.mylog.DFprintf("AppendEntries: append raft: %v, len of log %v to %v \n", rf.me, l, rf.log.GetLen())
	
				rf.persist()
				
			}

			for args.LeaderCommit > rf.commitIndex{
				rf.commitIndex = rf.commitIndex + 1
				 
				_, ok := rf.log.GetEntryFromIndex(rf.commitIndex) 

				if !ok{
					rf.commitIndex = rf.commitIndex - 1
					break
				}
	
			}
			rf.applyCond.Broadcast()

			
		}else{
			reply.Success = false
			
			conflictTerm := e.Term
			
			conflictIndex := args.PrevLogIndex-1
			for ; conflictIndex > 0; conflictIndex--{
				if ee, _ := rf.log.GetEntryFromIndex(conflictIndex); ee.Term != conflictTerm{
					
					break
				}
			}
			reply.ConflictIndex = conflictIndex
			 
		}
		
		
	}
	
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).
	rf.mu.Lock()
	if rf.identity == LEADER{
		l := rf.log.GetLen()
		term = rf.currentTerm
		index = l
		logentry := Entry{
			Term: term,
			Index:index,
			Command: command,
		}
		rf.log.Append(logentry)
		// DPrintf("Start: cmd(%v) is append, raft: %v, index: %v, len of log: %v\n", command, rf.me, index, len(rf.log))
		rf.mylog.DFprintf("Start: cmd(%v) is append, raft: %v, index: %v, len of log: %v\n", command, rf.me, index, rf.log.GetLen())
		nowtime := getNowTimeMillisecond()
		rf.persist()
		rf.mylog.DFprintf("Start: persist use %v ms\n", getNowTimeMillisecond()-nowtime)
		rf.mylog.DFprintf("Start: rafts %v heartBeats immediately\n", rf.me)

		go rf.heartBeats()
		
	}else{
		isLeader = false
	}
	rf.mu.Unlock()

	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	rf.mylog.DFprintf("&&&& raft %v will be killed! &&&&\n", rf.me)
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
	rf.applyCond.Broadcast()
	
	rf.mylog.DFprintf("&&&& raft %v is killed! &&&&\n", rf.me)
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func getNowTimeMillisecond() int64{
	return time.Now().UnixNano() / int64(time.Millisecond)

}
func getRandTimeoutMillisecond() int64{
	randtime := int64(500 + rand.Intn(400))
	return randtime
}

func (rf *Raft) election(){
	rf.mu.Lock()
	// rf.startTime = getNowTimeMillisecond()
	// rf.outTime = getRandTimeoutMillisecond()	//reset outtime for election 
	// rf.votedFor = rf.me		// vote self
	// rf.currentTerm = rf.currentTerm + 1

	// DPrintf("election: timeout raft : rf.me=%v, rf.term=%v, rf.identity=%v\n, rf.outTime=%v, rf.startTime=%v\n", rf.me, rf.currentTerm, rf.identity, rf.outTime, rf.startTime)
	
	e := rf.log.GetLastEntry()
	lastLogIndex := e.Index
	lastLogTerm := e.Term

	args := &RequestVoteArgs{
		Term: rf.currentTerm, 		// candidate's term
		CandidateId: rf.me,			// candidate requesting vote
		LastLogIndex: lastLogIndex,	// index of candidate's last log entry
		LastLogTerm: lastLogTerm,   // term of candidate's last log entry
	}
	rf.mu.Unlock()

	count := 1
	for idx, _ := range rf.peers{
		if idx == rf.me{
			continue
		}
		go func(i int, count *int, rf *Raft){

			oldterm, _ := rf.GetState()
			reply := &RequestVoteReply{}
			ok :=  rf.sendRequestVote(i, args, reply)
			if ok{
				rf.mu.Lock()
				defer rf.mu.Unlock()

				if rf.identity == CANDIATE && oldterm == rf.currentTerm{
					// if reply is not outdate
					if reply.Term > rf.currentTerm{
						rf.identity = FOLLOWER
						rf.currentTerm = reply.Term
						rf.votedFor = -1
						rf.startTime = getNowTimeMillisecond()
						rf.outTime = getRandTimeoutMillisecond()
						rf.persist()
						
					}else{
						if reply.VoteGranted{
							*count = *count+1
						}
						
						if *count > len(rf.peers)/2 {
							rf.identity = LEADER
							// DPrintf("election: win the election raft : rf.me=%v, rf.term=%v, rf.identity=%v, rf.outTime=%v, rf.startTime=%v\n", rf.me, rf.currentTerm, rf.identity, rf.outTime, rf.startTime)
							rf.mylog.DFprintf("election: win the election raft : rf.me=%v, rf.term=%v, rf.identity=%v, rf.outTime=%v, rf.startTime=%v\n", rf.me, rf.currentTerm, rf.identity, rf.outTime, rf.startTime)
							rf.winElectionChan <- struct{}{}	// may be dead locked !!!
						}
					}
					
	
				}
				
				
			
			}
			
		}(idx, &count, rf)
		
	}
 
 

}
func (rf *Raft) heartBeats(){
	// term := 0
	// leaderId := 0
	prevLogIndex := 0
	prevLogTerm := 0
	// entries := entries
	// leaderCommit := 0
	rf.mu.Lock()
	rf.nextIndex[rf.me] = rf.log.GetLen()
	rf.matchIndex[rf.me] = rf.log.GetLen()-1
 
	// DPrintf("heartBeats: raft: %v, nextIndex: %v, matchIndex: %v\n", rf.me, rf.nextIndex, rf.matchIndex)
	rf.mylog.DFprintf("heartBeats: raft: %v, nextIndex: %v, matchIndex: %v, len of log: %v\n", rf.me, rf.nextIndex, rf.matchIndex, rf.log.GetLen())
	
	// oldterm , _:= rf.GetState()
	oldterm := rf.currentTerm
	for idx, _ := range rf.peers{
		entries := make([]Entry, 0)
		nextIndex := rf.nextIndex[idx]
		if nextIndex < 1{
			nextIndex = 1
		}
		
		prevLogIndex = nextIndex-1
		e, _ := rf.log.GetEntryFromIndex(prevLogIndex)
		prevLogTerm = e.Term
		// if e, ok := rf.log[prevLogIndex]; ok{
		// 	prevLogTerm = e.Term
		// }
		entries = rf.log.GetEntriesAfterIndex(nextIndex)
		// for ;nextIndex <= len(rf.log)-1;nextIndex ++{
		// 	entries = append(entries, rf.log[nextIndex])
		// }

		args := &AppendEntriesArgs{
			Term: rf.currentTerm,	// leader's term
			LeaderId: rf.me,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm: prevLogTerm,
			Entries: entries,
			LeaderCommit: rf.commitIndex,
		}
		rf.mylog.DFprintf("heartBeats: raft: %v send entries to raft %v, args.PrevLogIndex: %v, args.PrevLogTerm: %v, args.LeaderCommit: %v, len of entries: %v \n", 
		rf.me, idx, args.PrevLogIndex, args.PrevLogTerm, args.LeaderCommit, len(args.Entries))
		startTime := getNowTimeMillisecond()
		// old_term := rf.currentTerm 
		if idx != rf.me && rf.identity == LEADER && rf.currentTerm == oldterm{
			go func(i int, rf *Raft){

				reply := &AppendEntriesReply{}
				
				ok := rf.sendAppendEntries(i, args, reply)
				rf.mylog.DFprintf("sendAppendEntries to raft %v use time %v (ms)\n", i, getNowTimeMillisecond()-startTime)

				if ok{
					rf.mu.Lock()
					defer rf.mu.Unlock()
					 // if leader is stale
					if rf.currentTerm == oldterm && rf.identity == LEADER{
						// rf is not outdate
						if reply.Term > rf.currentTerm{
							rf.identity = FOLLOWER
							rf.votedFor = -1
							rf.currentTerm = reply.Term
							rf.startTime = getNowTimeMillisecond()
							rf.outTime = getRandTimeoutMillisecond()

							rf.persist()
							// DPrintf("heartBeats: raft: %v return to follower, currentTerm is %v\n", rf.me, rf.currentTerm)
							rf.mylog.DFprintf("heartBeats: raft: %v return to follower, currentTerm is %v\n", rf.me, rf.currentTerm)
			
						}else{
							if reply.Success{
								// rf.nextIndex[i] = rf.nextIndex[i] + len(entries)
								rf.mylog.DFprintf("heartBeats: raft: %v, append success, rf.matchIndex from %v to %v\n", 
								i, rf.matchIndex[i], args.PrevLogIndex + len(entries))
								rf.matchIndex[i] = args.PrevLogIndex + len(entries)
								rf.nextIndex[i] = rf.matchIndex[i] + 1
								// rf.matchIndex[i] = rf.nextIndex[i] - 1
								
							}else{
								// DPrintf("heartBeats: raft: %v, nextIndex decrease, from %v to %v\n ", i, rf.nextIndex[i], rf.nextIndex[i] - len(entries) - 1)
								rf.mylog.DFprintf("heartBeats: raft: %v, nextIndex decrease, from %v to %v\n", i, rf.nextIndex[i], reply.ConflictIndex + 1)
								// rf.nextIndex[i] = rf.nextIndex[i] - len(entries) - 1
								rf.nextIndex[i] = reply.ConflictIndex + 1
								
							}
							willCommitIndex := rf.commitIndex + 1
							for ; willCommitIndex <= rf.log.GetLen()-1; willCommitIndex++{
								if e , _ := rf.log.GetEntryFromIndex(willCommitIndex); e.Term != rf.currentTerm{
									continue
								}
								count := 0
								for i := 0; i < len(rf.peers); i++{
									if willCommitIndex <= rf.matchIndex[i]{
										count = count + 1
										
									}
			
								}
								if count > len(rf.peers)/2{
									// DPrintf("heartBeats: cmd(%v) is committed, raft: %v, commitIndex: %v\n", rf.log[willCommitIndex].Command, rf.me, willCommitIndex)
									e, _ := rf.log.GetEntryFromIndex(willCommitIndex)
									rf.mylog.DFprintf("heartBeats: cmd(%v) is committed, raft: %v, commitIndex: %v, len of log: %v\n", 
									e.Command, rf.me, willCommitIndex, rf.log.GetLen())
									rf.commitIndex = willCommitIndex
									// rf.applyCh <- ApplyMsg{
									// 	CommandValid: true,	
									// 	Command: rf.log[willCommitIndex].Command,
									// 	CommandIndex: willCommitIndex,
									// }	// may be dead locked !!!
									
								}else{
									
									break
								}
							}
							// rf.commitIndex = willCommitIndex - 1
							rf.applyCond.Broadcast()
	
						}
					} 
				}
	
	
			}(idx, rf)
		}
		
			
		
	}
	rf.mu.Unlock()
	// time.Sleep(rf.constAppendTime * time.Millisecond)
	
	 
}
// apply committed cmd 
func (rf *Raft) applier() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.lastApplied = 0

	for rf.killed() == false{
		
		if rf.lastApplied >= rf.commitIndex{
			rf.applyCond.Wait()
		}else{
			startTime := getNowTimeMillisecond()
		
			lastApplied := rf.lastApplied + 1
			for ;lastApplied <= rf.commitIndex; lastApplied++{
				e, _ := rf.log.GetEntryFromIndex(lastApplied)
				applyMsg := ApplyMsg{
					CommandValid: true,	
					Command: e.Command,
					CommandIndex: lastApplied,
				}
				rf.mu.Unlock()
				rf.applyCh <- applyMsg
				rf.mu.Lock()
			}
			rf.lastApplied = lastApplied-1
			rf.mylog.DFprintf("applier use time %v (ms)\n", getNowTimeMillisecond()-startTime)
		}
		
		
	}
	close(rf.applyCh)
 
}
// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	for rf.killed() == false{
		if _, isleader := rf.GetState(); !isleader{

			rf.mu.Lock()
			curtime := getNowTimeMillisecond()
			starttime := rf.startTime
			
			if curtime - starttime > rf.outTime{
				// rf data must be updated immediately not in func: election,
				// otherwise they may be update by RPCs call and then be update in func: election twice
				rf.identity = CANDIATE
				rf.startTime = getNowTimeMillisecond()
				rf.outTime = getRandTimeoutMillisecond()	//reset outtime for election 
				rf.votedFor = rf.me		// vote self
				rf.currentTerm = rf.currentTerm + 1

				rf.persist()
				rf.mylog.DFprintf("election: timeout raft : rf.me=%v, rf.term=%v, rf.identity=%v\n, rf.outTime=%v, rf.startTime=%v\n", rf.me, rf.currentTerm, rf.identity, rf.outTime, rf.startTime)

				rf.mu.Unlock()
				// start election
				rf.checkChan <- struct{}{}
			}else{
				rf.mu.Unlock()
			}

		}
		
		time.Sleep(time.Duration(rf.constCheckTime) * time.Millisecond)
	}
 
	close(rf.checkChan)
	

}
func (rf *Raft) stateTrans(){
	for rf.killed() == false {

		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		// time.Sleep().
		rf.mu.Lock()
		switch rf.identity{
		case FOLLOWER:
			rf.mu.Unlock()
			select{
			case <-rf.checkChan:
				// turn to candiate
			}
			break
		case CANDIATE:
			rf.mu.Unlock()

			go rf.election()
			select{
			case <-rf.checkChan:
				//block if not time out
				
			case <-rf.winElectionChan:
				// win the election
				// Reinitialize nextIndex[] and matchIndex[] after win an election
				// rf.mu.Lock()
				for i := 0; i < len(rf.peers); i++{
					rf.nextIndex[i] = rf.log.GetLen()
					rf.matchIndex[i] = 0
				}
				// rf.nextIndex = len(rf.log)	
				// rf.mu.Unlock()
			}
			break
		case LEADER:
			// DPrintf("raft %v term: %v, start to heartbeats", rf.me, rf.currentTerm)

			rf.mu.Unlock()
			rf.heartBeats()
			time.Sleep(time.Duration(rf.constAppendTime) * time.Millisecond)

			break
		}
	}
	close(rf.winElectionChan)
}
//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg, mylog *Mylog) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.identity = FOLLOWER
	rf.startTime = getNowTimeMillisecond()
	rf.outTime = getRandTimeoutMillisecond()
	rf.constCheckTime = 20	//ms
	rf.constAppendTime = 100	//ms

	rf.currentTerm = 0
	rf.votedFor = -1
	rf.log = GetMapLog(mylog)
	
	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.nextIndex = make([]int, len(peers))
	// DPrintf("nextIndex %v", rf.nextIndex)
	rf.matchIndex = make([]int , len(peers))
	// DPrintf("matchIndex %v", rf.matchIndex)

	rf.checkChan = make(chan struct{})
	rf.winElectionChan = make(chan struct{})
	rf.applyCond = sync.NewCond(&rf.mu)
	rf.applyCh = applyCh
	rf.mylog = mylog

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	// DPrintf("Make: make a raft : rf.me=%v, rf.term=%v, rf.identity=%v, rf.outTime=%v, rf.startTime=%v\n", rf.me, rf.currentTerm, rf.identity, rf.outTime, rf.startTime)
	rf.mylog.DFprintf("Make: make a raft : rf.me=%v, rf.term=%v, rf.identity=%v, rf.outTime=%v, rf.startTime=%v\n", rf.me, rf.currentTerm, rf.identity, rf.outTime, rf.startTime)
	go rf.ticker()
	go rf.stateTrans()
	go rf.applier()

	return rf
}
