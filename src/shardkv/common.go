package shardkv

//
// Sharded key/value server.
// Lots of replica groups, each running Raft.
// Shardctrler decides which group serves each shard.
// Shardctrler may change shard assignment from time to time.
//
// You will have to modify these definitions.
//
type Err string
const (
	OK             = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongGroup  = "ErrWrongGroup"
	ErrWrongLeader = "ErrWrongLeader"
	ErrNotReady	   = "ErrNotReady"
)
// for shard state
type ShardState string
const (
	SERVED	= "SERVED"
	NOTSERVED	= "NOTSERVED"
	NOTREADY	= "NOTREADY"
)


// Put or Append
type PutAppendArgs struct {
	// You'll have to add definitions here.
	Key   string
	Value string
	Op    string // "Put" or "Append"
	// You'll have to add definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	ClientId int64
	SeqId int64
}

type PutAppendReply struct {
	Err Err
}

type GetArgs struct {
	Key string
	Op    string // "Get"

	// You'll have to add definitions here.
	ClientId int64
	SeqId int64
}

type GetReply struct {
	Err   Err
	Value string
}

type MoveShardDataArgs struct{
	// Method	string // "get" or "send"
	ConfigNum 	int	
	Shard		int	// which shard want get
}

type MoveShardDataReply struct{
	Err			Err
	ShardData	map[string]string
	ConfigNum	int
}


