4A

shard controller
configuration 管理 哪个 replica group 服务 哪些 shards
要求：
1. 使用raft备份
2. shards 可以在 replica group之间转移
3. 实现 RPCs Join、Leave、Move、Query，应该过滤重复请求

Join: 添加新 replica group，ShardCtrler 应该创建新的configuration，replica groups间的shards 应该尽可能平均

Leave： 删除replica group，ShardCtrler 应该创建新的configuration，将shards转移到剩下的replica groups中，
replica groups间的shards 应该尽可能平均

Move：将shard进行移动到对应gid的replica group上，ShardCtrler 应该创建新的configuration

Query: -1或大于最大版本，代表查询最新版本

replica groups


4B
for i in {1..10};do go test -run TestStaticShards/TestConcurrent3 -race; done
go test -run TestUnreliable3 -timeout 0  -race -args 100


hint: Add code to server.go to periodically fetch the latest configuration from the shardctrler,
 and 

go updateConfig()
额外开一个协程更新config，为了所有server都在同一个节点更新，将更新命令也加入到命令序列中，即
可以只有leader发送到raft备份，然后通过waitApply()更新

add code to reject client requests if the receiving group isn't responsible for the client's key's shard.

在 case opMsg = <- ch: 之后判断，
shard := key2shard(args.Key)
gid := kv.config.Shards[shard]
if gid != kv.me{}

hint:Your server should respond with an ErrWrongGroup error to a client RPC with a key that the server isn't responsible for
 (i.e. for a key whose shard is not assigned to the server's group). 
Make sure your Get, Put, and Append handlers make this decision correctly in the face of a concurrent re-configuration.

将更新config命令加入到 get等命令的序列命令中

hint: Think about how the shardkv client and server should deal with ErrWrongGroup.
 
 Should the client change the sequence number if it receives ErrWrongGroup?
不需要 

 Should the server update the client state if it returns ErrWrongGroup when executing a Get/Put request?
需要，更新config


hint: When group G1 needs a shard from G2 during a configuration change, 
does it matter at what point during its processing of log entries G2 sends the shard to G1?

没有关系，G2在发送shard前，会更新自身的config，所以不会出现G2将shard发送给G1后，G2仍旧处理该shard内key的请求的情况
没有G2没有处理的请求会返回ErrWSrongGroup，请求转而发送到G1

需要一个额外的协程检测是否需要迁移shard，通过channel进行通知，其余时间沉睡
和之间的config做对比，增加还是减少了

思路：

除了普通的kv数据库的put，get，append操作之外。还需要时刻检查config，以及config之后shard数据的迁移。后面两个也都是异步进行的。

改进：
1. apply时也需要检测 请求是否合法，所以需要有返回值.
2. 

problem

1. TestStaticShards  # bug4B_1

可能死循环 **ck.Query: resend arg 一直查询不成功
问题：test_test.go 117, 每次循环有5个协程阻塞，导致协程泄露
解决方案：改回for i in {1..10};do go test -run TestJoinLeave -race; done测试方法。
*-------------join 0, 1----------------168
此时shardctrl已经分好了，没问题**rebalance: sc: 0, move after: maxGidShardIdx[100] [5 6 7 8 9], minGidShardIdx[101] [0 1 2 3 4]
***updateConfig(): group: 102, kvserver: 1, get config: {1 [100 100 100 100 100 100 100 100 100 100] -----357
***updateConfig(): group: 101, kvserver: 1, get config: {1 [100 100 100 100 100 100 100 100 100 100] -----496
***updateConfig(): group: 100, kvserver: 1, get config: {1 [100 100 100 100 100 100 100 100 100 100] -----875

***updateConfig(): group: 102, kvserver: 1, get config: {2 [101 101 101 101 101 100 100 100 100 100]  ----951
***updateConfig(): group: 101, kvserver: 1, get config: {2 [101 101 101 101 101 100 100 100 100 100] -----984

***kv.doUpdate: group: 102, kvserver: 1, -------1009
***kv.doUpdate: group: 101, kvserver: 1, -------1062
***kv.doUpdate: group: 102, kvserver: 2, -------1075
***kv.doUpdate: group: 102, kvserver: 0, -------1110
 lastConfig: {Num:1 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]},
 config: {Num:2 Shards:[101 101 101 101 101 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2]]},
 moveInShards: map[]

***kv.doUpdate: group: 101, kvserver: 2, -------1192
***kv.doUpdate: group: 101, kvserver: 0, -------1206
 lastConfig: {Num:1 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]},
 config: {Num:2 Shards:[101 101 101 101 101 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2]]},
 moveInShards: map[0:100 1:100 2:100 3:100 4:100]

***updateConfig(): group: 100, kvserver: 1, get config: {2 [101 101 101 101 101 100 100 100 100 100] ----1233

***kv.doUpdate: group: 100, kvserver: 0, -------1510
 lastConfig: {Num:1 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]},
 config: {Num:2 Shards:[101 101 101 101 101 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2]]},
 moveInShards: map[]

*-------------ShutdownGroup 1----------------8856
***ck.Get: to servers([server-100-0 server-100-1 server-100-2]), args: {Key:0 Op:Get ClientId:1517649249356830459 SeqId:1} ----8960 没有问题ascii "0" - 48
***ck.Get: args: {Key:1 Op:Get ClientId:57775277739985767 SeqId:1} ---------9118
***ck.Get: to servers([server-100-0 server-100-1 server-100-2]), args: {Key:1 Op:Get ClientId:57775277739985767 SeqId:1} ---9119 key1同server100

***ck.Get: to servers([server-101-0 server-101-1 server-101-2]), args: {Key:2 Op:Get ClientId:2395603429312598135 SeqId:1} ---9253

***ck.Get: finish --------------------------9513 怎么还真给返回了，为什么没有errWrongGroup， 没有问题ascii "0" - 48
 args: {Key:0 Op:Get ClientId:1517649249356830459 SeqId:1},
 get reply: {Err:OK Value:zffWSwrWJBML8UHDWBtm}

election: win the election raft : rf.me=0, rf.term=2, rf.identity=2, rf.outTime=636, rf.startTime=1702715687949 ---13421

**ck.Query: resend args: &{Num:1 ClientId:1607179071014274220 SeqId:1}, ----13821 大频率出现
election: timeout raft : rf.me=2, rf.term=2, rf.identity=1 -----22038



2. TestJoinLeave # bug4B_2  文件误删了
错误:
fail: Get(0): expected:
skFN3kBv4Q
received:
问题:
server.go 458
moveInShards 被move完成后没有改变，导致多次发送moveshard rpcs，导致旧数据覆盖新数据
client 向group 1 请求key: "0"的value，但是key: "0"的数据还没准备好，导致 errnokey

解决方案：需要一个变量，标志shard 对应的数据的状态: 
served: 可以服务
notServed：不在服务范围内，
notReady：数据正在拉取，未准备好
接收请求和应用请求时，检测shard状态

*-----------join 0------------163
*-----------join 1------------2784

*-----------leave 0------------6699
***ck.Get: to servers([server-100-0 server-100-1 server-100-2]), args: {Key:0 Op:Get ClientId:1403282895137698809 SeqId:41} -----6760

***kv.doMoveShard: group: 101, kvserver: 1, ConfigNum: 2, ----6786
 kv.dataset: map[2:lPRMR 3:yZDwd 4:WZf6N 5:Wpu707nt9Q 6:kXrmhDtYJD]
 ShardData: map[5:Wpu70]
***kv.doMoveShard: group: 101, kvserver: 1, ----6789 把append的结果覆盖了
 kv.dataset: map[2:lPRMR 3:yZDwd 4:WZf6N 5:Wpu70 6:kXrmhDtYJD]

***updateConfig(): group: 100, kvserver: 2, kv.configNum: 2, --------------6812
  send new config: {Num:3 Shards:[101 101 101 101 101 101 101 101 101 101] Groups:map[101:[server-101-0 server-101-1 server-101-2]]}

***updateConfig(): group: 101, kvserver: 2, kv.configNum: 2, --------------6812
  send new config: {Num:3 Shards:[101 101 101 101 101 101 101 101 101 101] Groups:map[101:[server-101-0 server-101-1 server-101-2]]}

***kv.doUpdate: group: 100, kvserver: 2, ----------6939
 lastConfig: {Num:2 Shards:[101 101 101 101 101 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2]]},
 config: {Num:3 Shards:[101 101 101 101 101 101 101 101 101 101] Groups:map[101:[server-101-0 server-101-1 server-101-2]]},
 moveInShards: map[]
***kv.doUpdate: group: 100, kvserver: 1, ------7037
***kv.doUpdate: group: 100, kvserver: 0, ------7058

***ck.Get: ErrWrongGroup. not in group: 100, args: {Key:0 Op:Get ClientId:1403282895137698809 SeqId:41} ----------7044

***shardMove():group: 101, kvserver: 2, kv.moveInShards: map[0:100 1:100 2:100 3:100 4:100] -----7130 最后一次请求move

***kv.doUpdate: group: 101, kvserver: 2, -----7222
 lastConfig: {Num:2 Shards:[101 101 101 101 101 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2]]},
 config: {Num:3 Shards:[101 101 101 101 101 101 101 101 101 101] Groups:map[101:[server-101-0 server-101-1 server-101-2]]},
 moveInShards: map[5:100 6:100 7:100 8:100 9:100]
***kv.doUpdate: group: 101, kvserver: 1, -----7378
***kv.doUpdate: group: 101, kvserver: 0, -----7383

***ck.Get: to servers([server-101-0 server-101-1 server-101-2]), args: {Key:0 Op:Get ClientId:1403282895137698809 SeqId:41} ----7613


fail: Get(0): expected: ------7706
skFN3kBv4Q
received:

3. TestSnapshot# bug4B_3
错误：死循环
问题：
由于向group 100要数据时， group100已经更新到了num4，而此时还在要num3的数据，
因为func MoveShardData()中
if args.ConfigNum == kv.config.Num {的判断，导致一直无法获取数据
解决方法：
保存各个num的数据显然不合理。
如果不验证num，直接从dataset中获取，会不会有问题。
试想一下这种情况，group 1 的shard 0数据，在num=1时需要移入group2，num=2时又从group2移回。
如果group2的版本迟滞了，group2还是在num1，但是还没从group1中要到数据，这时候group1向group2要数据就会返回notready。
这时等待group1拿到数据就好。
因为同一个shard的数据只能在一个kvshard中出现，所以不管group1是什么版本了，它的shard0一定是唯一且最新的，我们拿走后删了就行，不删也行。

优化：为了简化，防止更复杂的情况出现。一次updateConfig, 一次moveShard，只有数据转移（需要的数据全部移入，移除不关心，否则太影响效率）完才允许继续更新。

*-----------join 0------------402
*-----------join 1 2, leave 0------------8818
**rebalance: ShardCtrler: 1, gids: [100 101] ----8863

***kv.doUpdate-: group: 100, kvserver: 2, -----9032
***kv.doUpdate-: group: 100, kvserver: 0, -----9405
***kv.doUpdate-: group: 100, kvserver: 1, -----9426
 lastConfig: {Num:1 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]},
 config: {Num:2 Shards:[101 101 101 101 101 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2]]},
 moveInShards: map[]
 kv.shardState: map[0:NOTSERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:SERVED 6:SERVED 7:SERVED 8:SERVED 9:SERVED]

**rebalance: ShardCtrler: 0, gids: [100 101] ----9062
**rebalance: ShardCtrler: 2, gids: [100 101] ----9094
**rebalance: sc: 1, move after: config.Shards: [102 101 101 101 101 102 102 100 100 100], -----9166
**rebalance: sc: 2, move after: config.Shards: [102 101 101 101 101 102 102 100 100 100], -----9257
**rebalance: sc: 0, move after: config.Shards: [102 101 101 101 101 102 102 100 100 100], -----9271

**rebalance: no zero, ShardCtrler: 1, gid2shardIdx: map[101:[1 2 3 4 8] 102:[0 5 6 7 9]] ------9313
**rebalance: no zero, ShardCtrler: 0, gid2shardIdx: map[101:[1 2 3 4 8] 102:[0 5 6 7 9]] ------9500
**rebalance: no zero, ShardCtrler: 2, gid2shardIdx: map[101:[1 2 3 4 8] 102:[0 5 6 7 9]] ------9518
--------------------------------------------------------------------------------------------------------------
***kv.doUpdate-: group: 101, kvserver: 0, ----9126
***kv.doUpdate-: group: 101, kvserver: 1, ----9725
***kv.doUpdate-: group: 101, kvserver: 2, ----9770
 lastConfig: {Num:1 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]},
 config: {Num:2 Shards:[101 101 101 101 101 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2]]},
 moveInShards: map[0:100 1:100 2:100 3:100 4:100]
 kv.shardState: map[0:NOTREADY 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]
 

***kv.doUpdate-: group: 102, kvserver: 0, -----10536
***kv.doUpdate-: group: 102, kvserver: 2, -----10609
***kv.doUpdate-: group: 102, kvserver: 1, -----10627
 lastConfig: {Num:2 Shards:[101 101 101 101 101 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2]]},
 config: {Num:3 Shards:[102 101 101 101 101 102 102 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[0:101 5:100 6:100]
 kv.shardState: map[0:NOTREADY 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTREADY 6:NOTREADY 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***kv.doMoveShard-: group: 102, kvserver: 2, -----10738 成功move了shard 0 后面shard 5、6就一直没有获取
***shardMove():group: 102, kvserver: 0, kv.moveInShards: map[0:101 5:100 6:100] -----10570  
问题在这里，由于向group 100要数据时， group100已经更新到了num4，而此时还在要num3的数据，
因为func MoveShardData()中
if args.ConfigNum == kv.config.Num {的判断，导致一直无法获取数据

***kv.doUpdate-: group: 100, kvserver: 2, -----10515
***kv.doUpdate-: group: 100, kvserver: 0, -----10965
***kv.doUpdate-: group: 100, kvserver: 1, -----10989
 lastConfig: {Num:3 Shards:[102 101 101 101 101 102 102 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {Num:4 Shards:[102 101 101 101 101 102 102 102 101 102] Groups:map[101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[]
 kv.shardState: map[0:NOTSERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]


***kv.doUpdate-: group: 101, kvserver: 0, -----10804
***kv.doUpdate-: group: 101, kvserver: 2, -----11269 后来更新到了num4
***kv.doUpdate-: group: 101, kvserver: 1, -----11296
 lastConfig: {Num:3 Shards:[102 101 101 101 101 102 102 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {Num:4 Shards:[102 101 101 101 101 102 102 102 101 102] Groups:map[101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[8:100]
 kv.shardState: map[0:NOTSERVED 1:SERVED 2:SERVED 3:SERVED 4:SERVED 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTREADY 9:NOTSERVED]

***kv.doUpdate-: group: 101, kvserver: 2, -----10817   怎么才更新到num3，有点滞后
***kv.doUpdate-: group: 101, kvserver: 1, -----10833
 lastConfig: {Num:2 Shards:[101 101 101 101 101 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2]]},
 config: {Num:3 Shards:[102 101 101 101 101 102 102 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[]
 kv.shardState: map[0:NOTSERVED 1:SERVED 2:SERVED 3:SERVED 4:SERVED 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***kv.doUpdate-: group: 102, kvserver: 0, -------11030
***kv.doUpdate-: group: 102, kvserver: 2, -------11155
***kv.doUpdate-: group: 102, kvserver: 1, -------11183
 lastConfig: {Num:3 Shards:[102 101 101 101 101 102 102 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {Num:4 Shards:[102 101 101 101 101 102 102 102 101 102] Groups:map[101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[7:100 9:100]
 kv.shardState: map[0:SERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTREADY 6:NOTREADY 7:NOTREADY 8:NOTSERVED 9:NOTREADY]


***kv.doMoveShard: group: 101, kvserver: 0, ConfigNum: 4, ----11406
***kv.doMoveShard: group: 101, kvserver: 1, ConfigNum: 4, ----11423
***kv.doMoveShard: group: 101, kvserver: 2, ConfigNum: 4, ----11451
 ShardData: map[0:nMejF7p6MJZWFtmNBj1BHuZgaq6E4E4ZZPhm7Bhb]
 kv.moveInShards: map[8:100]


4. TestSnapshot# bug4B_4
错误：死循环

*-----------join 0, leave 1------------ 13085
**rebalance: sc: 1, move after: config.Shards: [100 100 100 100 100 102 102 102 102 102],  ----13239
**rebalance: sc: 0, move after: config.Shards: [100 100 100 100 100 102 102 102 102 102],  ----13308
**rebalance: sc: 2, move after: config.Shards: [100 100 100 100 100 102 102 102 102 102],  ----13357


***kv.doUpdate-: group: 102, kvserver: 2, ---------------13674
***kv.doUpdate-: group: 102, kvserver: 0, ---------------13948
***kv.doUpdate-: group: 102, kvserver: 1, ---------------13983
 lastConfig: {Num:4 Shards:[102 101 101 101 101 102 102 102 101 102] Groups:map[101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {Num:5 Shards:[102 102 102 102 102 102 102 102 102 102] Groups:map[102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[1:101 2:101 3:101 4:101 8:101]
 kv.shardState: map[0:SERVED 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:SERVED 6:SERVED 7:SERVED 8:NOTREADY 9:SERVED]

***kv.doUpdate-: group: 100, kvserver: 0, ---------------13209
***kv.doUpdate-: group: 100, kvserver: 2, ---------------14606
***kv.doUpdate-: group: 100, kvserver: 1, ---------------14624
 lastConfig: {Num:5 Shards:[102 102 102 102 102 102 102 102 102 102] Groups:map[102:[server-102-0 server-102-1 server-102-2]]},
 config: {Num:6 Shards:[100 100 100 100 100 102 102 102 102 102] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[0:102 1:102 2:102 3:102 4:102]
 kv.shardState: map[0:NOTREADY 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***kv.doMoveShard: group: 102, kvserver: 2, opMsg.ConfigNum(6) - kv.config.Num(5) ---------14190 问题 group102一直无法完成更新
由于 group102向group100请求数据时，group100已经更新到了num6，所以返回时reply.ConfigNum = 6, 导致了kv.doMoveShard更新失败
然后 group102更新到了num6，清空了moveInShards，也就不再请求shard1的数据了，这就导致了下面的问题二

***kv.doUpdate-: group: 102, kvserver: 2,  -----14549
***kv.doUpdate-: group: 102, kvserver: 0,  -----14754
***kv.doUpdate-: group: 102, kvserver: 1,  -----14768
 lastConfig: {Num:5 Shards:[102 102 102 102 102 102 102 102 102 102] Groups:map[102:[server-102-0 server-102-1 server-102-2]]},
 config: {Num:6 Shards:[100 100 100 100 100 102 102 102 102 102] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[]
 kv.shardState: map[0:NOTSERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:SERVED 6:SERVED 7:SERVED 8:NOTREADY 9:SERVED]


***kv.doMoveShard-: group: 100, kvserver: 0, -------14911
***kv.doMoveShard-: group: 100, kvserver: 2, -------15288
***kv.doMoveShard-: group: 100, kvserver: 1, -------15333
 kv.dataset: map[0:NVvtw2EsiT1JMAvQX7CHPPo9CSt_0CRVrEmA1uyH 1:wbcKRUVPOwirLwJurI0ijPFnXXSOAPhZWpiCP1Xy 10:ns2YgYnEZFiTSs8gRgqs 11:wS_lUxNpAgiYK6iFV_Yj 12:gb3aDRppJR4Y-Q2UNwgO 13:7BR_4Q3FogAfEg0yesj2 14:BwWQf-nXo_Sdtz8-91S9 15:5b5R1KCiqpwZxSV4-dk2 16:ognFyu5_gFrmUDAAyC_f 17:EufVtmcFbDVBt--Ch_KP 18:dzIeZTkx_ZQkuaZ6umMP 19:veNyAYU2Joe8Icp-2CQi 2:Zba-P8OL-a_quQUdUjbHy9ZghShr306XDEHosMHX 20:iXqCuEFWY7d8jyDUq1Iq--nQhKNv6K47rq-a9BNK 21:3_skt1b-bvMt454mkYgHxvcaBqOWMqRbshTPm2fD 22:c29j2zXBU-eQ48ceFFu80KhFGV6hJNYC8dj02wZe 23:9c57ZouM75wUkXW0gBjnfdpqdgxk3F6heLmfc2XU 24:9fNmkd8eKV0KKEEo5IX9k8QcL8sEviPbQ-CN5hOO 25:kX0RbtPvCagvm3f9H66_9qDsjqVLH27pnXxDOiLm 26:lYQM_x_rkiJZ30aI91M-CznGccLSz_iFvuKjqdOG 27:TtxoSuhMID5blXYLSVktj8aU3LFHPxQIZAeu58Ql 28:IqJjLzAtoLCgJ6gcobqeq00ta7jEydSKSNCfy0ih 29:qoY8-lweg3zU3QtdRxg2zqQRAEAJMAepkjl7vde4 3:M19rOdMfJ3TayqFE-adb 4:hih1k8lU7YVnTsWW62jE 5:93VfCdaIOfTay2Raui_b 6:9kokcAX9rK88K7V0Fhye 7:CF_vozCYxVRlyFmwNHbe 8:P9ewIP7e7kBHo5PoVbiO 9:xkqR7ZwULXm0OaXTYzri]
 kv.moveInShards: map[1:102 2:102 3:102 4:102],  *******************************shard 0 完成了
 kv.shardState:map[0:SERVED 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]


***kv.doMoveShard-: group: 100, kvserver: 1, ------------183825 一直无法获取剩下的数据
 kv.dataset: map[0:NVvtw2EsiT1JMAvQX7CHPPo9CSt_0CRVrEmA1uyH 1:wbcKRUVPOwirLwJurI0ijPFnXXSOAPhZWpiCP1Xy 10:ns2YgYnEZFiTSs8gRgqs 11:wS_lUxNpAgiYK6iFV_Yj 12:gb3aDRppJR4Y-Q2UNwgO 13:7BR_4Q3FogAfEg0yesj2 14:BwWQf-nXo_Sdtz8-91S9 15:5b5R1KCiqpwZxSV4-dk2 16:ognFyu5_gFrmUDAAyC_f 17:EufVtmcFbDVBt--Ch_KP 18:dzIeZTkx_ZQkuaZ6umMP 19:veNyAYU2Joe8Icp-2CQi 2:Zba-P8OL-a_quQUdUjbHy9ZghShr306XDEHosMHX 20:iXqCuEFWY7d8jyDUq1Iq--nQhKNv6K47rq-a9BNK 21:3_skt1b-bvMt454mkYgHxvcaBqOWMqRbshTPm2fD 22:c29j2zXBU-eQ48ceFFu80KhFGV6hJNYC8dj02wZe 23:9c57ZouM75wUkXW0gBjnfdpqdgxk3F6heLmfc2XU 24:9fNmkd8eKV0KKEEo5IX9k8QcL8sEviPbQ-CN5hOO 25:kX0RbtPvCagvm3f9H66_9qDsjqVLH27pnXxDOiLm 26:lYQM_x_rkiJZ30aI91M-CznGccLSz_iFvuKjqdOG 27:TtxoSuhMID5blXYLSVktj8aU3LFHPxQIZAeu58Ql 28:IqJjLzAtoLCgJ6gcobqeq00ta7jEydSKSNCfy0ih 29:qoY8-lweg3zU3QtdRxg2zqQRAEAJMAepkjl7vde4 3:M19rOdMfJ3TayqFE-adb 4:hih1k8lU7YVnTsWW62jE 5:93VfCdaIOfTay2Raui_b 6:9kokcAX9rK88K7V0Fhye 7:CF_vozCYxVRlyFmwNHbe 8:P9ewIP7e7kBHo5PoVbiO 9:xkqR7ZwULXm0OaXTYzri]
 kv.moveInShards: map[1:102 2:102 3:102 4:102],
 kv.shardState:map[0:SERVED 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***MoveShardData():group: 102, kvserver: 2, kv.configNum: 6, args: &{ConfigNum:6 Shard:1} --------185706
***callMoveShardData():group: 100, kvserver: 0, reply: {Err:OK ShardData:map[] ConfigNum:6} 一直获取，
但是获取到数据一直为空，导致doMoveShard一直未删除moveInShards

这里有两个问题，
一是没有考虑对应shard为空的情况，若获取为空，便会不断获取造成死循环
二是这里的shard对应的数据肯定是不为空的，为什么获取不到
解决方案：
一：传入需要move的shard，无论是否为空都只能执行一次
二：server.go (kv *ShardKV) MoveShardData, reply.ConfigNum = kv.config.Num 改为 reply.ConfigNum = args.ConfigNum


5. TestSnapshot# bug4B_5
错误：死循环
问题：kvserver2滞后，被leader强行同步，但是却没有同步moveInShards，导致shard 8还是notready状态，最终造成不停请求shard8数据的死循环
解决方法：
moveInShards也进行持久化


***kv.doUpdate-: group: 102, kvserver: 0, -------14299
***kv.doUpdate-: group: 102, kvserver: 1,  ------14401 但是keserver2就没有经历这一步
 lastConfig: {Num:4 Shards:[102 101 101 101 101 102 102 102 101 102] Groups:map[101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {Num:5 Shards:[102 102 102 102 102 102 102 102 102 102] Groups:map[102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[1:101 2:101 3:101 4:101 8:101]
 kv.shardState: map[0:SERVED 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:SERVED 6:SERVED 7:SERVED 8:NOTREADY 9:SERVED]

***kv.waitApply: CondInstallSnapshot: group: 102, kvserver: 2, -----14376 kvserver2被强行同步了，但是却没有同步moveInShards，导致出错
 dataset: map[1:9P4AbxvmZZtJErpZIiTdoHh3WVjO4YV17emCwPka 10:CLKihUa66cltPL5p2Xixgg_gTf_OTtNlncxx8L2e 11:1WaXlMc6ar_Fai3Dnmxby_K7_03aHg_jhOZLzgTG 12:FvAIMEtTNV1ayHgLO0ZuOHA7WaNTifKWLnHKu0dQ 13:HI2e_ldE7zdbcgM2FJGRs1wVoTVXt7J6KxdiSuiS 14:oNZXyzZ0Elf4vlunTHwNX0D29RFWz3TT1EOlNIxu 15:wWXZSg0OSs8qkQkkzkz52E8WHvA74gdMqRi6ZyCD 16:POdljuRkrX8St3vNbTcun694CjDRQceEnV2xvwmV 17:GT23wldzv2VscwZJXxz5S8E25hmFvI3ENVGh2gg8 18:WfS8SjUY2l4Npk2itACW1JHe0HyD-y2MNPSWF4vX 19:hihX2w8YAS46BBqBqE7sZ3_7KB0CAmqQTv-7dp9H 2:2jRiuNOQoeGGMMG1Uzi03kl1KVzSuZgF27MHMzj_ 20:ZwOn5MAjBjwbkghyEd98gip9oJEDpTqpuiuI2GlW 21:OOXyVRdVbJqhlw91iys0WKHXTXScbiJjTWFKKcgU 22:R4vNAUM1QDlNWZ6VErOwJxLuKL7ZRuXaR1xAD2iw 23:cOWHa5hstznf3a8jP54g1j6Hm9obeTCLxKdUuL_u 24:wNtnO2MmdLbWyVEMxINV8YGaEzOf78LeIVEIf_l5 25:q7dUri7LI28WZR5CPwQfTenWgCIK_EmzEUmaNXRo 26:KON7W7ws156zg0ehxEWbkU9ZJO1MRaZtXp8H_gea 27:bQCzDLh7FUiU0Y7bPGlxevnpplq8evuhqtgVDCN8 28:hEvQ6Gup3TCvCTvdNlZVlO8TCNE0ZDqbgXZygBTk 29:_nz5eyecoM6AnB7XW8jUO-7ko2Y2w0PTAU-1DbyI 7:FV4Fnd0JkqoIl_b6nK663Ibid6uKVccoC781aHfp 8:5RPxpzi4SZCjFaeRB-9fYsls7vBdWLNM8hHMyNNH 9:jp3jEKgc3m-Ry3jlDhifjK_bTxlMyNja9XYOJIA0],
 clientId2seqId: map[3141885465673129522:120],
 shardState: map[0:SERVED 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:SERVED 6:SERVED 7:SERVED 8:NOTREADY 9:SERVED]
, lastConfig: {4 [102 101 101 101 101 102 102 102 101 102] map[101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}
, config: {5 [102 102 102 102 102 102 102 102 102 102] map[102:[server-102-0 server-102-1 server-102-2]]}
, lastApplied: 59 


***kv.doMoveShard-: group: 102, kvserver: 0,  ------14888
***kv.doMoveShard-: group: 102, kvserver: 1,  ------14903
opMsg: {Method:MoveShard ClientId:0 SeqId:0 Key: Value: Config:{Num:0 Shards:[0 0 0 0 0 0 0 0 0 0] Groups:map[]} Shard:8 ShardData:map[0:M_dxr4N05FdSZdVjxf64doMIMzb87jOqUZfGOWDjv8O3bG8QVO_AjDTcJxwg] ConfigNum:5},
 kv.dataset: map[0:M_dxr4N05FdSZdVjxf64doMIMzb87jOqUZfGOWDjv8O3bG8QVO_AjDTcJxwg 1:9P4AbxvmZZtJErpZIiTdoHh3WVjO4YV17emCwPkaGUYP_1-cREAKfYDlhi4R 10:CLKihUa66cltPL5p2Xixgg_gTf_OTtNlncxx8L2e 11:1WaXlMc6ar_Fai3Dnmxby_K7_03aHg_jhOZLzgTG 12:FvAIMEtTNV1ayHgLO0ZuOHA7WaNTifKWLnHKu0dQ 13:HI2e_ldE7zdbcgM2FJGRs1wVoTVXt7J6KxdiSuiS 14:oNZXyzZ0Elf4vlunTHwNX0D29RFWz3TT1EOlNIxu 15:wWXZSg0OSs8qkQkkzkz52E8WHvA74gdMqRi6ZyCD 16:POdljuRkrX8St3vNbTcun694CjDRQceEnV2xvwmV 17:GT23wldzv2VscwZJXxz5S8E25hmFvI3ENVGh2gg8 18:WfS8SjUY2l4Npk2itACW1JHe0HyD-y2MNPSWF4vX 19:hihX2w8YAS46BBqBqE7sZ3_7KB0CAmqQTv-7dp9H 2:2jRiuNOQoeGGMMG1Uzi03kl1KVzSuZgF27MHMzj_8nXSfu4PGFOcqQWAoTcf 20:ZwOn5MAjBjwbkghyEd98gip9oJEDpTqpuiuI2GlW 21:OOXyVRdVbJqhlw91iys0WKHXTXScbiJjTWFKKcgU 22:R4vNAUM1QDlNWZ6VErOwJxLuKL7ZRuXaR1xAD2iw 23:cOWHa5hstznf3a8jP54g1j6Hm9obeTCLxKdUuL_u 24:wNtnO2MmdLbWyVEMxINV8YGaEzOf78LeIVEIf_l5 25:q7dUri7LI28WZR5CPwQfTenWgCIK_EmzEUmaNXRo 26:KON7W7ws156zg0ehxEWbkU9ZJO1MRaZtXp8H_gea 27:bQCzDLh7FUiU0Y7bPGlxevnpplq8evuhqtgVDCN8 28:hEvQ6Gup3TCvCTvdNlZVlO8TCNE0ZDqbgXZygBTk 29:_nz5eyecoM6AnB7XW8jUO-7ko2Y2w0PTAU-1DbyI 7:FV4Fnd0JkqoIl_b6nK663Ibid6uKVccoC781aHfp 8:5RPxpzi4SZCjFaeRB-9fYsls7vBdWLNM8hHMyNNH 9:jp3jEKgc3m-Ry3jlDhifjK_bTxlMyNja9XYOJIA0]
 kv.moveInShards: map[1:101 2:101 3:101 4:101],
 kv.shardState:map[0:SERVED 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:SERVED 6:SERVED 7:SERVED 8:SERVED 9:SERVED]

***kv.doMoveShard: group: 102, kvserver: 2,  ------14999 由于kv.moveInShards: 为[]，导致shard8的状态并没有被转换
opMsg: {Method:MoveShard ClientId:0 SeqId:0 Key: Value: Config:{Num:0 Shards:[0 0 0 0 0 0 0 0 0 0] Groups:map[]} Shard:8 ShardData:map[0:M_dxr4N05FdSZdVjxf64doMIMzb87jOqUZfGOWDjv8O3bG8QVO_AjDTcJxwg] ConfigNum:5},
 kv.dataset: map[1:9P4AbxvmZZtJErpZIiTdoHh3WVjO4YV17emCwPkaGUYP_1-cREAKfYDlhi4R 10:CLKihUa66cltPL5p2Xixgg_gTf_OTtNlncxx8L2e 11:1WaXlMc6ar_Fai3Dnmxby_K7_03aHg_jhOZLzgTG 12:FvAIMEtTNV1ayHgLO0ZuOHA7WaNTifKWLnHKu0dQ 13:HI2e_ldE7zdbcgM2FJGRs1wVoTVXt7J6KxdiSuiS 14:oNZXyzZ0Elf4vlunTHwNX0D29RFWz3TT1EOlNIxu 15:wWXZSg0OSs8qkQkkzkz52E8WHvA74gdMqRi6ZyCD 16:POdljuRkrX8St3vNbTcun694CjDRQceEnV2xvwmV 17:GT23wldzv2VscwZJXxz5S8E25hmFvI3ENVGh2gg8 18:WfS8SjUY2l4Npk2itACW1JHe0HyD-y2MNPSWF4vX 19:hihX2w8YAS46BBqBqE7sZ3_7KB0CAmqQTv-7dp9H 2:2jRiuNOQoeGGMMG1Uzi03kl1KVzSuZgF27MHMzj_8nXSfu4PGFOcqQWAoTcf 20:ZwOn5MAjBjwbkghyEd98gip9oJEDpTqpuiuI2GlW 21:OOXyVRdVbJqhlw91iys0WKHXTXScbiJjTWFKKcgU 22:R4vNAUM1QDlNWZ6VErOwJxLuKL7ZRuXaR1xAD2iw 23:cOWHa5hstznf3a8jP54g1j6Hm9obeTCLxKdUuL_u 24:wNtnO2MmdLbWyVEMxINV8YGaEzOf78LeIVEIf_l5 25:q7dUri7LI28WZR5CPwQfTenWgCIK_EmzEUmaNXRo 26:KON7W7ws156zg0ehxEWbkU9ZJO1MRaZtXp8H_gea 27:bQCzDLh7FUiU0Y7bPGlxevnpplq8evuhqtgVDCN8 28:hEvQ6Gup3TCvCTvdNlZVlO8TCNE0ZDqbgXZygBTk 29:_nz5eyecoM6AnB7XW8jUO-7ko2Y2w0PTAU-1DbyI 7:FV4Fnd0JkqoIl_b6nK663Ibid6uKVccoC781aHfp 8:5RPxpzi4SZCjFaeRB-9fYsls7vBdWLNM8hHMyNNH 9:jp3jEKgc3m-Ry3jlDhifjK_bTxlMyNja9XYOJIA0]
 kv.moveInShards: map[],
 kv.shardState:map[0:SERVED 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:SERVED 6:SERVED 7:SERVED 8:NOTREADY 9:SERVED]


***kv.waitApply: Snapshot: group: 102, kvserver: 1, -----24886
 dataset: map[0:M_dxr4N05FdSZdVjxf64doMIMzb87jOqUZfGOWDjv8O3bG8QVO_AjDTcJxwg 1:9P4AbxvmZZtJErpZIiTdoHh3WVjO4YV17emCwPkaGUYP_1-cREAKfYDlhi4R 10:CLKihUa66cltPL5p2Xixgg_gTf_OTtNlncxx8L2eKaQ13qU2w33CbEBgMM23 11:1WaXlMc6ar_Fai3Dnmxby_K7_03aHg_jhOZLzgTG1FFMOdPZ7FzN5A1e_RVZ 12:FvAIMEtTNV1ayHgLO0ZuOHA7WaNTifKWLnHKu0dQ7pdr4ALA0Q4SbHuxgtDv 13:HI2e_ldE7zdbcgM2FJGRs1wVoTVXt7J6KxdiSuiS5vB0Iv2CFJpEE72GwhrU 14:oNZXyzZ0Elf4vlunTHwNX0D29RFWz3TT1EOlNIxuNy1VULh8C_3Urc9-xizi 15:wWXZSg0OSs8qkQkkzkz52E8WHvA74gdMqRi6ZyCDKlNmbQEc5r3zLwSnZPUf 16:POdljuRkrX8St3vNbTcun694CjDRQceEnV2xvwmVHoFxMuwYNIQHxJazXUt6 17:GT23wldzv2VscwZJXxz5S8E25hmFvI3ENVGh2gg8WwNxwcbmVa0TEXLdV6Kq 18:WfS8SjUY2l4Npk2itACW1JHe0HyD-y2MNPSWF4vX7dFLzer_rgrnuCYR9wkg 19:hihX2w8YAS46BBqBqE7sZ3_7KB0CAmqQTv-7dp9HXhWS18Yg3RyB_ZuIHITd 2:2jRiuNOQoeGGMMG1Uzi03kl1KVzSuZgF27MHMzj_8nXSfu4PGFOcqQWAoTcf 20:ZwOn5MAjBjwbkghyEd98gip9oJEDpTqpuiuI2GlW 21:OOXyVRdVbJqhlw91iys0WKHXTXScbiJjTWFKKcgU 22:R4vNAUM1QDlNWZ6VErOwJxLuKL7ZRuXaR1xAD2iw 23:cOWHa5hstznf3a8jP54g1j6Hm9obeTCLxKdUuL_u 24:wNtnO2MmdLbWyVEMxINV8YGaEzOf78LeIVEIf_l5 25:q7dUri7LI28WZR5CPwQfTenWgCIK_EmzEUmaNXRo 26:KON7W7ws156zg0ehxEWbkU9ZJO1MRaZtXp8H_gea 27:bQCzDLh7FUiU0Y7bPGlxevnpplq8evuhqtgVDCN8 28:hEvQ6Gup3TCvCTvdNlZVlO8TCNE0ZDqbgXZygBTk 29:_nz5eyecoM6AnB7XW8jUO-7ko2Y2w0PTAU-1DbyI 3:4p-huAOrfo3HF9kbt1xqMHGAwSOSjddyT5xkz0wp 4:tHwHD7WyVnQVUzFq0doiHdZLSou1SLOnT63IuipA 5:OHeYPZhzBAz5cSvDZQ_-GAsQ_QN_kqVgc3A-Yawm 6:VnLdXn3tpvQcBE0gSoBfifkE2CuxMcF9Xbwe_LFs 7:FV4Fnd0JkqoIl_b6nK663Ibid6uKVccoC781aHfpv_BKv9FpODxSOgRDjzRY 8:5RPxpzi4SZCjFaeRB-9fYsls7vBdWLNM8hHMyNNHMQqdNSw3fKbWAdnzMNDH 9:jp3jEKgc3m-Ry3jlDhifjK_bTxlMyNja9XYOJIA02iKGz1QCUmBOuZnJiXwV],
 clientId2seqId: map[3141885465673129522:160],
 shardState: map[0:NOTSERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:SERVED 6:SERVED 7:SERVED 8:SERVED 9:SERVED]
, lastConfig: {5 [102 102 102 102 102 102 102 102 102 102] map[102:[server-102-0 server-102-1 server-102-2]]}
, config: {6 [100 100 100 100 100 102 102 102 102 102] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}
, lastApplied: 109 

***kv.waitApply: Snapshot: group: 102, kvserver: 2,  -----24934
 dataset: map[0:M_dxr4N05FdSZdVjxf64doMIMzb87jOqUZfGOWDjv8O3bG8QVO_AjDTcJxwg 1:9P4AbxvmZZtJErpZIiTdoHh3WVjO4YV17emCwPkaGUYP_1-cREAKfYDlhi4R 10:CLKihUa66cltPL5p2Xixgg_gTf_OTtNlncxx8L2eKaQ13qU2w33CbEBgMM23 11:1WaXlMc6ar_Fai3Dnmxby_K7_03aHg_jhOZLzgTG1FFMOdPZ7FzN5A1e_RVZ 12:FvAIMEtTNV1ayHgLO0ZuOHA7WaNTifKWLnHKu0dQ7pdr4ALA0Q4SbHuxgtDv 13:HI2e_ldE7zdbcgM2FJGRs1wVoTVXt7J6KxdiSuiS5vB0Iv2CFJpEE72GwhrU 14:oNZXyzZ0Elf4vlunTHwNX0D29RFWz3TT1EOlNIxuNy1VULh8C_3Urc9-xizi 15:wWXZSg0OSs8qkQkkzkz52E8WHvA74gdMqRi6ZyCDKlNmbQEc5r3zLwSnZPUf 16:POdljuRkrX8St3vNbTcun694CjDRQceEnV2xvwmVHoFxMuwYNIQHxJazXUt6 17:GT23wldzv2VscwZJXxz5S8E25hmFvI3ENVGh2gg8WwNxwcbmVa0TEXLdV6Kq 18:WfS8SjUY2l4Npk2itACW1JHe0HyD-y2MNPSWF4vX7dFLzer_rgrnuCYR9wkg 19:hihX2w8YAS46BBqBqE7sZ3_7KB0CAmqQTv-7dp9HXhWS18Yg3RyB_ZuIHITd 2:2jRiuNOQoeGGMMG1Uzi03kl1KVzSuZgF27MHMzj_8nXSfu4PGFOcqQWAoTcf 20:ZwOn5MAjBjwbkghyEd98gip9oJEDpTqpuiuI2GlW 21:OOXyVRdVbJqhlw91iys0WKHXTXScbiJjTWFKKcgU 22:R4vNAUM1QDlNWZ6VErOwJxLuKL7ZRuXaR1xAD2iw 23:cOWHa5hstznf3a8jP54g1j6Hm9obeTCLxKdUuL_u 24:wNtnO2MmdLbWyVEMxINV8YGaEzOf78LeIVEIf_l5 25:q7dUri7LI28WZR5CPwQfTenWgCIK_EmzEUmaNXRo 26:KON7W7ws156zg0ehxEWbkU9ZJO1MRaZtXp8H_gea 27:bQCzDLh7FUiU0Y7bPGlxevnpplq8evuhqtgVDCN8 28:hEvQ6Gup3TCvCTvdNlZVlO8TCNE0ZDqbgXZygBTk 29:_nz5eyecoM6AnB7XW8jUO-7ko2Y2w0PTAU-1DbyI 3:4p-huAOrfo3HF9kbt1xqMHGAwSOSjddyT5xkz0wp 4:tHwHD7WyVnQVUzFq0doiHdZLSou1SLOnT63IuipA 5:OHeYPZhzBAz5cSvDZQ_-GAsQ_QN_kqVgc3A-Yawm 6:VnLdXn3tpvQcBE0gSoBfifkE2CuxMcF9Xbwe_LFs 7:FV4Fnd0JkqoIl_b6nK663Ibid6uKVccoC781aHfpv_BKv9FpODxSOgRDjzRY 8:5RPxpzi4SZCjFaeRB-9fYsls7vBdWLNM8hHMyNNHMQqdNSw3fKbWAdnzMNDH 9:jp3jEKgc3m-Ry3jlDhifjK_bTxlMyNja9XYOJIA02iKGz1QCUmBOuZnJiXwV],
 clientId2seqId: map[3141885465673129522:160],
 shardState: map[0:NOTSERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:SERVED 6:SERVED 7:SERVED 8:NOTREADY 9:SERVED]
, lastConfig: {5 [102 102 102 102 102 102 102 102 102 102] map[102:[server-102-0 server-102-1 server-102-2]]}
, config: {6 [100 100 100 100 100 102 102 102 102 102] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}
, lastApplied: 109 
为什么同一个lastApplied: ， 但是状态却不一样。 shard 8

*-----------shutdown and restart all------------27222

***kv.readDataset: load group: 100, kvserver: 0, ----------27263
***kv.readDataset: load group: 100, kvserver: 1, ----------27276
***kv.readDataset: load group: 100, kvserver: 2, ----------27288
 dataset: map[0:M_dxr4N05FdSZdVjxf64doMIMzb87jOqUZfGOWDj 1:9P4AbxvmZZtJErpZIiTdoHh3WVjO4YV17emCwPka 10:CLKihUa66cltPL5p2Xix 11:1WaXlMc6ar_Fai3Dnmxb 12:FvAIMEtTNV1ayHgLO0Zu 13:HI2e_ldE7zdbcgM2FJGR 14:oNZXyzZ0Elf4vlunTHwN 15:wWXZSg0OSs8qkQkkzkz5 16:POdljuRkrX8St3vNbTcu 17:GT23wldzv2VscwZJXxz5 18:WfS8SjUY2l4Npk2itACW 19:hihX2w8YAS46BBqBqE7s 2:2jRiuNOQoeGGMMG1Uzi03kl1KVzSuZgF27MHMzj_8nXSfu4PGFOcqQWAoTcf 20:ZwOn5MAjBjwbkghyEd98gip9oJEDpTqpuiuI2GlWtZ1aaQqvedoD2xkybyY9 21:OOXyVRdVbJqhlw91iys0WKHXTXScbiJjTWFKKcgUi2LEWSICDXwJ6Q-EFOwA 22:R4vNAUM1QDlNWZ6VErOwJxLuKL7ZRuXaR1xAD2iwGI8gpKzuIvxVGCeBZvH2 23:cOWHa5hstznf3a8jP54g1j6Hm9obeTCLxKdUuL_ubmeBywEEEfY1OzjMEJM0 24:wNtnO2MmdLbWyVEMxINV8YGaEzOf78LeIVEIf_l5azd_Sr6cy8mlaxb194t8 25:q7dUri7LI28WZR5CPwQfTenWgCIK_EmzEUmaNXRooM7lg-YI1XLszInzBfUj 26:KON7W7ws156zg0ehxEWbkU9ZJO1MRaZtXp8H_geaU0e_N3IPPgq9H97UA_d9 27:bQCzDLh7FUiU0Y7bPGlxevnpplq8evuhqtgVDCN8xa2L2eXuYdeMvBI9-SeJ 28:hEvQ6Gup3TCvCTvdNlZVlO8TCNE0ZDqbgXZygBTkG5Bc4Iv86xCk98yjnG8V 29:_nz5eyecoM6AnB7XW8jUO-7ko2Y2w0PTAU-1DbyItys_I-d2S_frxP9AQfLP 3:4p-huAOrfo3HF9kbt1xqMHGAwSOSjddyT5xkz0wpEGNlZnr7fSse_H2H8BiU 4:tHwHD7WyVnQVUzFq0doiHdZLSou1SLOnT63IuipAcXjoDlfn-Px3MXh80Toh 5:OHeYPZhzBAz5cSvDZQ_-GAsQ_QN_kqVgc3A-YawmSiGgkuNhm81AM2QhUS59 6:VnLdXn3tpvQcBE0gSoBfifkE2CuxMcF9Xbwe_LFs_KnCJVPqatqM5X9s3jp5 7:FV4Fnd0JkqoIl_b6nK66 8:5RPxpzi4SZCjFaeRB-9f 9:jp3jEKgc3m-Ry3jlDhif],
 clientId2seqId: map[3141885465673129522:180],
 shardState: map[0:SERVED 1:SERVED 2:SERVED 3:SERVED 4:SERVED 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]
, lastConfig: {5 [102 102 102 102 102 102 102 102 102 102] map[102:[server-102-0 server-102-1 server-102-2]]}
, config: {6 [100 100 100 100 100 102 102 102 102 102] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}


***kv.readDataset: load group: 101, kvserver: 0, ----------27306
***kv.readDataset: load group: 101, kvserver: 1, ----------27312
***kv.readDataset: load group: 101, kvserver: 2, ----------27324
 dataset: map[0:M_dxr4N05FdSZdVjxf64doMIMzb87jOqUZfGOWDjv8O3bG8QVO_AjDTcJxwg 2:2jRiuNOQoeGGMMG1Uzi03kl1KVzSuZgF27MHMzj_ 20:ZwOn5MAjBjwbkghyEd98 21:OOXyVRdVbJqhlw91iys0 22:R4vNAUM1QDlNWZ6VErOw 23:cOWHa5hstznf3a8jP54g 24:wNtnO2MmdLbWyVEMxINV 25:q7dUri7LI28WZR5CPwQf 26:KON7W7ws156zg0ehxEWb 27:bQCzDLh7FUiU0Y7bPGlx 28:hEvQ6Gup3TCvCTvdNlZV 29:_nz5eyecoM6AnB7XW8jU 3:4p-huAOrfo3HF9kbt1xqMHGAwSOSjddyT5xkz0wp 4:tHwHD7WyVnQVUzFq0doiHdZLSou1SLOnT63IuipA 5:OHeYPZhzBAz5cSvDZQ_-GAsQ_QN_kqVgc3A-Yawm 6:VnLdXn3tpvQcBE0gSoBfifkE2CuxMcF9Xbwe_LFs],
 clientId2seqId: map[3141885465673129522:122],
 shardState: map[0:NOTSERVED 1:SERVED 2:SERVED 3:SERVED 4:SERVED 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:SERVED 9:NOTSERVED]
, lastConfig: {3 [102 101 101 101 101 102 102 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}
, config: {4 [102 101 101 101 101 102 102 102 101 102] map[101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}


***kv.readDataset: load group: 102, kvserver: 0, ----------27335
***kv.readDataset: load group: 102, kvserver: 1, ----------27348
 dataset: map[0:M_dxr4N05FdSZdVjxf64doMIMzb87jOqUZfGOWDjv8O3bG8QVO_AjDTcJxwg 1:9P4AbxvmZZtJErpZIiTdoHh3WVjO4YV17emCwPkaGUYP_1-cREAKfYDlhi4R 10:CLKihUa66cltPL5p2Xixgg_gTf_OTtNlncxx8L2eKaQ13qU2w33CbEBgMM23 11:1WaXlMc6ar_Fai3Dnmxby_K7_03aHg_jhOZLzgTG1FFMOdPZ7FzN5A1e_RVZ 12:FvAIMEtTNV1ayHgLO0ZuOHA7WaNTifKWLnHKu0dQ7pdr4ALA0Q4SbHuxgtDv 13:HI2e_ldE7zdbcgM2FJGRs1wVoTVXt7J6KxdiSuiS5vB0Iv2CFJpEE72GwhrU 14:oNZXyzZ0Elf4vlunTHwNX0D29RFWz3TT1EOlNIxuNy1VULh8C_3Urc9-xizi 15:wWXZSg0OSs8qkQkkzkz52E8WHvA74gdMqRi6ZyCDKlNmbQEc5r3zLwSnZPUf 16:POdljuRkrX8St3vNbTcun694CjDRQceEnV2xvwmVHoFxMuwYNIQHxJazXUt6 17:GT23wldzv2VscwZJXxz5S8E25hmFvI3ENVGh2gg8WwNxwcbmVa0TEXLdV6Kq 18:WfS8SjUY2l4Npk2itACW1JHe0HyD-y2MNPSWF4vX7dFLzer_rgrnuCYR9wkg 19:hihX2w8YAS46BBqBqE7sZ3_7KB0CAmqQTv-7dp9HXhWS18Yg3RyB_ZuIHITd 2:2jRiuNOQoeGGMMG1Uzi03kl1KVzSuZgF27MHMzj_8nXSfu4PGFOcqQWAoTcf 20:ZwOn5MAjBjwbkghyEd98gip9oJEDpTqpuiuI2GlW 21:OOXyVRdVbJqhlw91iys0WKHXTXScbiJjTWFKKcgU 22:R4vNAUM1QDlNWZ6VErOwJxLuKL7ZRuXaR1xAD2iw 23:cOWHa5hstznf3a8jP54g1j6Hm9obeTCLxKdUuL_u 24:wNtnO2MmdLbWyVEMxINV8YGaEzOf78LeIVEIf_l5 25:q7dUri7LI28WZR5CPwQfTenWgCIK_EmzEUmaNXRo 26:KON7W7ws156zg0ehxEWbkU9ZJO1MRaZtXp8H_gea 27:bQCzDLh7FUiU0Y7bPGlxevnpplq8evuhqtgVDCN8 28:hEvQ6Gup3TCvCTvdNlZVlO8TCNE0ZDqbgXZygBTk 29:_nz5eyecoM6AnB7XW8jUO-7ko2Y2w0PTAU-1DbyI 3:4p-huAOrfo3HF9kbt1xqMHGAwSOSjddyT5xkz0wp 4:tHwHD7WyVnQVUzFq0doiHdZLSou1SLOnT63IuipA 5:OHeYPZhzBAz5cSvDZQ_-GAsQ_QN_kqVgc3A-Yawm 6:VnLdXn3tpvQcBE0gSoBfifkE2CuxMcF9Xbwe_LFs 7:FV4Fnd0JkqoIl_b6nK663Ibid6uKVccoC781aHfpv_BKv9FpODxSOgRDjzRY 8:5RPxpzi4SZCjFaeRB-9fYsls7vBdWLNM8hHMyNNHMQqdNSw3fKbWAdnzMNDH 9:jp3jEKgc3m-Ry3jlDhifjK_bTxlMyNja9XYOJIA02iKGz1QCUmBOuZnJiXwV],
 clientId2seqId: map[3141885465673129522:160],
 shardState: map[0:NOTSERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:SERVED 6:SERVED 7:SERVED 8:SERVED 9:SERVED]
, lastConfig: {5 [102 102 102 102 102 102 102 102 102 102] map[102:[server-102-0 server-102-1 server-102-2]]}
, config: {6 [100 100 100 100 100 102 102 102 102 102] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}
 
***kv.readDataset: load group: 102, kvserver: 2, ----------27359
 dataset: map[0:M_dxr4N05FdSZdVjxf64doMIMzb87jOqUZfGOWDjv8O3bG8QVO_AjDTcJxwg 1:9P4AbxvmZZtJErpZIiTdoHh3WVjO4YV17emCwPkaGUYP_1-cREAKfYDlhi4R 10:CLKihUa66cltPL5p2Xixgg_gTf_OTtNlncxx8L2eKaQ13qU2w33CbEBgMM23 11:1WaXlMc6ar_Fai3Dnmxby_K7_03aHg_jhOZLzgTG1FFMOdPZ7FzN5A1e_RVZ 12:FvAIMEtTNV1ayHgLO0ZuOHA7WaNTifKWLnHKu0dQ7pdr4ALA0Q4SbHuxgtDv 13:HI2e_ldE7zdbcgM2FJGRs1wVoTVXt7J6KxdiSuiS5vB0Iv2CFJpEE72GwhrU 14:oNZXyzZ0Elf4vlunTHwNX0D29RFWz3TT1EOlNIxuNy1VULh8C_3Urc9-xizi 15:wWXZSg0OSs8qkQkkzkz52E8WHvA74gdMqRi6ZyCDKlNmbQEc5r3zLwSnZPUf 16:POdljuRkrX8St3vNbTcun694CjDRQceEnV2xvwmVHoFxMuwYNIQHxJazXUt6 17:GT23wldzv2VscwZJXxz5S8E25hmFvI3ENVGh2gg8WwNxwcbmVa0TEXLdV6Kq 18:WfS8SjUY2l4Npk2itACW1JHe0HyD-y2MNPSWF4vX7dFLzer_rgrnuCYR9wkg 19:hihX2w8YAS46BBqBqE7sZ3_7KB0CAmqQTv-7dp9HXhWS18Yg3RyB_ZuIHITd 2:2jRiuNOQoeGGMMG1Uzi03kl1KVzSuZgF27MHMzj_8nXSfu4PGFOcqQWAoTcf 20:ZwOn5MAjBjwbkghyEd98gip9oJEDpTqpuiuI2GlW 21:OOXyVRdVbJqhlw91iys0WKHXTXScbiJjTWFKKcgU 22:R4vNAUM1QDlNWZ6VErOwJxLuKL7ZRuXaR1xAD2iw 23:cOWHa5hstznf3a8jP54g1j6Hm9obeTCLxKdUuL_u 24:wNtnO2MmdLbWyVEMxINV8YGaEzOf78LeIVEIf_l5 25:q7dUri7LI28WZR5CPwQfTenWgCIK_EmzEUmaNXRo 26:KON7W7ws156zg0ehxEWbkU9ZJO1MRaZtXp8H_gea 27:bQCzDLh7FUiU0Y7bPGlxevnpplq8evuhqtgVDCN8 28:hEvQ6Gup3TCvCTvdNlZVlO8TCNE0ZDqbgXZygBTk 29:_nz5eyecoM6AnB7XW8jUO-7ko2Y2w0PTAU-1DbyI 3:4p-huAOrfo3HF9kbt1xqMHGAwSOSjddyT5xkz0wp 4:tHwHD7WyVnQVUzFq0doiHdZLSou1SLOnT63IuipA 5:OHeYPZhzBAz5cSvDZQ_-GAsQ_QN_kqVgc3A-Yawm 6:VnLdXn3tpvQcBE0gSoBfifkE2CuxMcF9Xbwe_LFs 7:FV4Fnd0JkqoIl_b6nK663Ibid6uKVccoC781aHfpv_BKv9FpODxSOgRDjzRY 8:5RPxpzi4SZCjFaeRB-9fYsls7vBdWLNM8hHMyNNHMQqdNSw3fKbWAdnzMNDH 9:jp3jEKgc3m-Ry3jlDhifjK_bTxlMyNja9XYOJIA02iKGz1QCUmBOuZnJiXwV],
 clientId2seqId: map[3141885465673129522:160],
 shardState: map[0:NOTSERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:SERVED 6:SERVED 7:SERVED 8:NOTREADY 9:SERVED]
shard 8 的状态是notready！！！

***updateConfig(): group: 101, kvserver: 0, kv.configNum: 4, ---------------31640 后面就没有了，一直没有更新到num6， 通过log回溯更新了
 send new config: {Num:5 Shards:[102 102 102 102 102 102 102 102 102 102] Groups:map[102:[server-102-0 server-102-1 server-102-2]]}

***kv.doUpdate-: group: 101, kvserver: 0, ------31686
***kv.doUpdate-: group: 101, kvserver: 1, ------31807
***kv.doUpdate-: group: 101, kvserver: 2, ------31850
 lastConfig: {Num:4 Shards:[102 101 101 101 101 102 102 102 101 102] Groups:map[101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {Num:5 Shards:[102 102 102 102 102 102 102 102 102 102] Groups:map[102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[]
 kv.shardState: map[0:NOTSERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***kv.doUpdate-: group: 101, kvserver: 0, ------31718
***kv.doUpdate-: group: 101, kvserver: 1, ------31826
***kv.doUpdate-: group: 101, kvserver: 2, ------31868
 lastConfig: {Num:5 Shards:[102 102 102 102 102 102 102 102 102 102] Groups:map[102:[server-102-0 server-102-1 server-102-2]]},
 config: {Num:6 Shards:[100 100 100 100 100 102 102 102 102 102] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[]
 kv.shardState: map[0:NOTSERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]


***kv.Get-, key: 0 in the group: 102 is not ready, config.Shards: [100 100 100 100 100 102 102 102 102 102] -------一直没有ready


6. TestConcurrent3 # bug4B_6
错误：
fail: Get(2): expected: key2 属于shard0
m-MDwGkeGdUiitWMSZE
received:
m-MDwWMSZE
问题：
当前一个版本的config所需要的数据未转移完成，不能进行下一个config的更新，但是假设这个情况：
当num=6，config所需的数据未转移完成，kvshard shutdown再start，此时便重新执行updateconfig()，导致config更新，使得最新的数据没有转移成功，
append在旧数据上实施，发生错误
解决方法：

func (kv *ShardKV) ticker(){
	for kv.killed() == false{
		kv.mu.Lock()
		l := len(kv.moveInShards)
		kv.mu.Unlock()
		if len > 0{
			kv.shardMove()
		}else{
			kv.updateConfig()
		}		
		time.Sleep(50*time.Millisecond) 
	}
}
但这仍无法保证一种情况，l := len(kv.moveInShards) 和 kv.updateConfig()之间，因为log回溯进行了updateConfig()导致moveInShards发生变化。
所以在更新时，既要保证新config.Num是旧config的后一版本，还要保证len(kv.moveInShards) == 0 即没有要转入的数据。
func (kv *ShardKV) doUpdate(opMsg Op){
	if opMsg.Config.Num == kv.config.Num+1 && len(kv.moveInShards) == 0{

*-----------join 2 1------------ 4177
***ck.PutAppend: finish ------8529    m-MDwG已经添加到了，看来是被覆盖了
 args: {Key:2 Value:G Op:Append ClientId:4187485241530242204 SeqId:5},
 get reply: {Err:OK}

*-----------shutdown 0 1 2, start 0 1 2------------ 8873

***kv.readDataset: load group: 100, kvserver: 0, ------8975
***kv.readDataset: load group: 100, kvserver: 1, ------8995 
***kv.readDataset: load group: 100, kvserver: 2, ------9029
 dataset: map[0:75v7bTOr4 1:JFqgekESm 2:m-MDw 3:02Wk 4:sXIV 5:wHIK 6:ql2 7:ic994 8:xKpB 9:-Tmf],
 clientId2seqId: map[14132034788159840:3 170493724742603415:8 329716370506775378:3 381007544930920199:2 1140221141850216790:3 1806007842906888491:3 1913053818396621436:8 2300578112370263533:4 4061258987902816970:10 4187485241530242204:4 4244922913676436403:3],
 shardState: map[0:NOTSERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTSERVED 6:NOTSERVED 7:SERVED 8:SERVED 9:SERVED],
 kv.moveInShards: map[],
 lastConfig: {2 [102 102 102 102 102 100 100 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {3 [101 102 102 102 102 101 101 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}

***kv.readDataset: load group: 102, kvserver: 0, ---------9086
***kv.readDataset: load group: 102, kvserver: 1, ---------9098
***kv.readDataset: load group: 102, kvserver: 2, ---------9112
 dataset: map[2:m-MDw 3:02Wk 4:sXIV 5:wHIK- 6:ql2],
 clientId2seqId: map[14132034788159840:4 170493724742603415:5 329716370506775378:1 381007544930920199:2 1140221141850216790:3 1806007842906888491:3 1913053818396621436:6 2300578112370263533:3 4061258987902816970:10 4187485241530242204:4 4244922913676436403:1],
 shardState: map[0:NOTSERVED 1:SERVED 2:SERVED 3:SERVED 4:SERVED 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED],
 kv.moveInShards: map[],
 lastConfig: {2 [102 102 102 102 102 100 100 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {3 [101 102 102 102 102 101 101 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}

***kv.waitApply: Snapshot: group: 101, kvserver: 1,  -------8959
 dataset: map[2:m-MDwGk 7:ic994],
 clientId2seqId: map[14132034788159840:4 170493724742603415:8 329716370506775378:3 381007544930920199:3 1140221141850216790:3 1806007842906888491:3 1913053818396621436:8 2300578112370263533:4 4061258987902816970:10 4187485241530242204:6 4244922913676436403:3],
 shardState: map[0:SERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:SERVED 6:NOTREADY 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED],
 kv.moveInShards map[6:100],
 lastConfig: {2 [102 102 102 102 102 100 100 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {3 [101 102 102 102 102 101 101 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 lastApplied: 9


*-----------leave 1 2------------ 12948

***ck.PutAppend: finish ---------18004   
 args: {Key:2 Value:e Op:Append ClientId:4187485241530242204 SeqId:7},
 get reply: {Err:OK}

***kv.doUpdate-: group: 102, kvserver: 0, -------------19627
 lastConfig: {Num:5 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]},
 config: {Num:6 Shards:[102 102 102 102 102 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[0:100 1:100 2:100 3:100 4:100]
 kv.shardState: map[0:NOTREADY 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]


*-----------shutdown 0 1 2, start 0 1 2------------19704
***kv.readDataset: load group: 101, kvserver: 0, ------19897 之间group101shutdown之前没有达到snapshot的标准，所以没有readDataset ，现在group101拥有最新的shard0数据
***kv.readDataset: load group: 101, kvserver: 1, ------19909
***kv.readDataset: load group: 101, kvserver: 2, ------19939
 dataset: map[2:m-MDwGk 7:ic994],
 clientId2seqId: map[14132034788159840:4 170493724742603415:8 329716370506775378:3 381007544930920199:3 1140221141850216790:3 1806007842906888491:3 1913053818396621436:8 2300578112370263533:4 4061258987902816970:10 4187485241530242204:6 4244922913676436403:3],
 shardState: map[0:SERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:SERVED 6:NOTREADY 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED],
 kv.moveInShards: map[6:100],
 lastConfig: {2 [102 102 102 102 102 100 100 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {3 [101 102 102 102 102 101 101 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}

***kv.readDataset: load group: 102, kvserver: 0, -------------19968
 dataset: map[2:m-MDw 3:02WksSzmeZar 4:sXIV8X9tcFOr9 5:wHIK-P5FOlVz4 6:ql284H9Sa96 8:xKpB],
 clientId2seqId: map[14132034788159840:12 170493724742603415:11 329716370506775378:4 381007544930920199:10 1140221141850216790:12 1806007842906888491:11 1913053818396621436:10 2300578112370263533:4 4061258987902816970:10 4187485241530242204:6 4244922913676436403:3],
 shardState: map[0:NOTREADY 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED],
 kv.moveInShards: map[0:100 1:100 2:100 3:100 4:100],
 lastConfig: {5 [100 100 100 100 100 100 100 100 100 100] map[100:[server-100-0 server-100-1 server-100-2]]},
 config: {6 [102 102 102 102 102 100 100 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}

***kv.readDataset: load group: 102, kvserver: 1, -------------20005
***kv.readDataset: load group: 102, kvserver: 2, -------------20024
 dataset: map[2:m-MDw 3:02WksSzmeZa 4:sXIV8X9tcFO 5:wHIK-P5FOlVz 6:ql284H9Sa9 8:xKpB],
 clientId2seqId: map[14132034788159840:11 170493724742603415:11 329716370506775378:4 381007544930920199:9 1140221141850216790:10 1806007842906888491:10 1913053818396621436:10 2300578112370263533:4 4061258987902816970:10 4187485241530242204:6 4244922913676436403:3],
 shardState: map[0:NOTSERVED 1:SERVED 2:SERVED 3:SERVED 4:SERVED 5:NOTSERVED 6:SERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED],
 kv.moveInShards: map[],
 lastConfig: {3 [101 102 102 102 102 101 101 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {4 [100 102 102 102 102 100 102 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}

***shardMove():group: 102, kvserver: 0, kv.moveInShards: map[0:100 1:100 2:100 3:100 4:100]  ------20097

***kv.doUpdate-: group: 102, kvserver: 1, -------------22256
***kv.doUpdate-: group: 102, kvserver: 2, -------------22342
 lastConfig: {Num:5 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]},
 config: {Num:6 Shards:[102 102 102 102 102 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[0:100 1:100 2:100 3:100 4:100]
 kv.shardState: map[0:NOTREADY 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***updateConfig(): group: 102, kvserver: 0, kv.configNum: 6, ------22490
 send new config: {Num:7 Shards:[101 102 102 102 102 101 101 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}


***kv.doUpdate-: group: 102, kvserver: 0,  -------------22708
 上一个config需要的数据还没有move成功，就update到下一个config了,不可能啊，会肯定卡在move那一步
发现问题：这里才是restart后的第一次update，然后之前log回溯的到num6所需的数据还没获取，shardState就被覆盖了！！！
***kv.doUpdate-: group: 102, kvserver: 1,  -------------23261
***kv.doUpdate-: group: 102, kvserver: 2,  -------------23289
 lastConfig: {Num:6 Shards:[102 102 102 102 102 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {Num:7 Shards:[101 102 102 102 102 101 101 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[]
 kv.shardState: map[0:NOTSERVED 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***kv.doAppend: Append, group: 100, kvserver: 2, key: 2, value(m-MDwGkeGd)=  -----23341
 (past)m-MDwGkeG+(add)d
***kv.doAppend: Append, group: 100, kvserver: 2, key: 2, value(m-MDwGkeGdU)= -----23373
 (past)m-MDwGkeGd+(add)U

***ck.PutAppend: finish ----------- 23618
 args: {Key:2 Value:U Op:Append ClientId:4187485241530242204 SeqId:10},
 get reply: {Err:OK}

***kv.doAppend: Append, group: 100, kvserver: 2, key: 2, value(m-MDwGkeGdUi)= -----23910
 (past)m-MDwGkeGdU+(add)i
***kv.doAppend: Append, group: 100, kvserver: 0, key: 2, value(m-MDwGkeGdUii)= -----25458
 (past)m-MDwGkeGdUi+(add)i
***kv.doAppend: Append, group: 100, kvserver: 0, key: 2, value(m-MDwGkeGdUiit)= -----26612
 (past)m-MDwGkeGdUii+(add)t

*-----------shutdown 0 1 2, start 0 1 2------------27316
***kv.readDataset: load group: 100, kvserver: 0, ----------27440
 dataset: map[0:75v7bTOr4YFrwjINrVl 1:JFqgekESm5MHPtEb2hy 2:m-MDwGkeGdUii 3:02WksSzmeZar 4:sXIV8X9tcFOr9 5:wHIK-P5FOlVz4 6:ql284H9Sa96 7:ic994F2c- 8:xKpB 9:-Tmf_mpp8Nc00],
 clientId2seqId: map[14132034788159840:12 170493724742603415:18 329716370506775378:12 381007544930920199:10 1140221141850216790:12 1806007842906888491:11 1913053818396621436:18 2300578112370263533:8 4061258987902816970:10 4187485241530242204:12 4244922913676436403:3],
 shardState: map[0:SERVED 1:SERVED 2:SERVED 3:SERVED 4:SERVED 5:SERVED 6:SERVED 7:SERVED 8:SERVED 9:SERVED],
 kv.moveInShards: map[],
 lastConfig: {4 [100 102 102 102 102 100 102 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {5 [100 100 100 100 100 100 100 100 100 100] map[100:[server-100-0 server-100-1 server-100-2]]}

***kv.readDataset: load group: 100, kvserver: 1, ----------27453
***kv.readDataset: load group: 100, kvserver: 2, ----------27466 现在shard0在group102上
 dataset: map[0:75v7bTOr4YFrwjINrVl44 1:JFqgekESm5MHPtEb2hyML 2:m-MDwGkeGdUiit 3:02WksSzmeZar 4:sXIV8X9tcFOr9 5:wHIK-P5FOlVz4 6:ql284H9Sa96 7:ic994F2c-EW 8:xKpB5 9:-Tmf_mpp8Nc00_],
 clientId2seqId: map[14132034788159840:12 170493724742603415:20 329716370506775378:13 381007544930920199:10 1140221141850216790:12 1806007842906888491:11 1913053818396621436:20 2300578112370263533:10 4061258987902816970:10 4187485241530242204:13 4244922913676436403:4],
 shardState: map[0:NOTSERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:SERVED 6:SERVED 7:SERVED 8:SERVED 9:SERVED],
 kv.moveInShards: map[],
 lastConfig: {5 [100 100 100 100 100 100 100 100 100 100] map[100:[server-100-0 server-100-1 server-100-2]]},
 config: {6 [102 102 102 102 102 100 100 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}

***kv.readDataset: load group: 101, kvserver: 0, ----------27479 怎么num才到3
***kv.readDataset: load group: 101, kvserver: 1, ----------27501
***kv.readDataset: load group: 101, kvserver: 2, ----------27514
 dataset: map[2:m-MDwGk 7:ic994],
 clientId2seqId: map[14132034788159840:4 170493724742603415:8 329716370506775378:3 381007544930920199:3 1140221141850216790:3 1806007842906888491:3 1913053818396621436:8 2300578112370263533:4 4061258987902816970:10 4187485241530242204:6 4244922913676436403:3],
 shardState: map[0:SERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:SERVED 6:NOTREADY 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED],
 kv.moveInShards: map[6:100],
 lastConfig: {2 [102 102 102 102 102 100 100 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {3 [101 102 102 102 102 101 101 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}

***kv.readDataset: load group: 102, kvserver: 0, ------27552
***kv.readDataset: load group: 102, kvserver: 1, ------27581
***kv.readDataset: load group: 102, kvserver: 2, ------27595
 dataset: map[2:m-MDw 3:02WksSzmeZar 4:sXIV8X9tcFOr9 5:wHIK-P5FOlVz4 6:ql284H9Sa96 8:xKpB],
 clientId2seqId: map[14132034788159840:12 170493724742603415:11 329716370506775378:4 381007544930920199:10 1140221141850216790:12 1806007842906888491:11 1913053818396621436:10 2300578112370263533:4 4061258987902816970:10 4187485241530242204:6 4244922913676436403:3],
 shardState: map[0:NOTSERVED 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:NOTSERVED 6:NOTREADY 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED],
 kv.moveInShards: map[6:101], **************************moveinshards和config、lastConfig不匹配！！！
 lastConfig: {7 [101 102 102 102 102 101 101 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {8 [100 102 102 102 102 100 102 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}



*-----------leave 1 2------------30376
***kv.doAppend: Append, group: 100, kvserver: 0, key: 2, value(m-MDwGkeGdUiit)= -----30831
 (past)m-MDwGkeGdUii+(add)t

***kv.doUpdate-: group: 101, kvserver: 1, -----31162
 lastConfig: {Num:6 Shards:[102 102 102 102 102 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {Num:7 Shards:[101 102 102 102 102 101 101 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[0:102 5:100 6:100]
 kv.shardState: map[0:NOTREADY 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTREADY 6:NOTREADY 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***kv.doMoveShard: group: 101, kvserver: 1,  -----31170
 opMsg: {Method:MoveShard ClientId:0 SeqId:0 Key: Value: Config:{Num:0 Shards:[0 0 0 0 0 0 0 0 0 0] Groups:map[]} Shard:0 ShardData:map[2:m-MDw] ClientId2seqId:map[14132034788159840:12 170493724742603415:11 329716370506775378:4 381007544930920199:10 1140221141850216790:12 1806007842906888491:11 1913053818396621436:10 2300578112370263533:4 4061258987902816970:10 4187485241530242204:6 4244922913676436403:3] ConfigNum:7},
 kv.dataset: map[2:m-MDwGk 7:ic994 8:xKpB]
 kv.moveInShards: map[0:102 5:100 6:100],
 kv.shardState:map[0:NOTREADY 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTREADY 6:NOTREADY 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]
***kv.doMoveShard-: group: 101, kvserver: 1,  -----31174  
这里group101向group102获取了旧数据，并将shard0状态改成了served
不对的地方：
group102的shard0数据应该至少比group100num5的新才对，num5->num6 shard0的数据应该转移到group102了才对

opMsg: {Method:MoveShard ClientId:0 SeqId:0 Key: Value: Config:{Num:0 Shards:[0 0 0 0 0 0 0 0 0 0] Groups:map[]} Shard:0 ShardData:map[2:m-MDw] ClientId2seqId:map[14132034788159840:12 170493724742603415:11 329716370506775378:4 381007544930920199:10 1140221141850216790:12 1806007842906888491:11 1913053818396621436:10 2300578112370263533:4 4061258987902816970:10 4187485241530242204:6 4244922913676436403:3] ConfigNum:7},
 kv.dataset: map[2:m-MDw 7:ic994 8:xKpB]
 kv.moveInShards: map[5:100 6:100],
 kv.shardState:map[0:SERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTREADY 6:NOTREADY 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]


***kv.doAppend: Append, group: 101, kvserver: 1, key: 2, value(m-MDwW)= -------31214 有问题
 (past)m-MDw+(add)W

*-----------join 2 1------------32602


***kv.readDataset: load group: 100, kvserver: 2,  -------59364这里shard2数据已经有问题了
 dataset: map[0:75v7bTOr4YFrwjINrVl44U6wjhOf6iD-3I 1:JFqgekESm5MHPtEb2hyMLSYeXyBogGhmbAp 2:m-MDwWMSZ 3:02WksSzmeZar 4:sXIV8X9tcFOr9 5:wHIK-P5FOlVz4 6:ql284H9Sa96 7:ic994Q5y 8:xKpBW 9:-Tmf_mpp8Nc00_pjiyuOCKiiwOMY3],
 clientId2seqId: map[14132034788159840:12 170493724742603415:34 329716370506775378:28 381007544930920199:10 1140221141850216790:12 1806007842906888491:11 1913053818396621436:33 2300578112370263533:15 4061258987902816970:10 4187485241530242204:17 4244922913676436403:8],
 shardState: map[0:SERVED 1:SERVED 2:SERVED 3:SERVED 4:SERVED 5:SERVED 6:SERVED 7:SERVED 8:SERVED 9:SERVED],
 kv.moveInShards: map[],
 lastConfig: {8 [100 102 102 102 102 100 102 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {9 [100 100 100 100 100 100 100 100 100 100] map[100:[server-100-0 server-100-1 server-100-2]]}

***kv.readDataset: load group: 102, kvserver: 0, -------59509
***kv.readDataset: load group: 102, kvserver: 1, -------59546
***kv.readDataset: load group: 102, kvserver: 2, -------59558
 dataset: map[2:m-MDw 3:02WksSzmeZar 4:sXIV8X9tcFOr9 5:wHIK-P5FOlVz4 6:ql284H9Sa96 8:xKpB],
 clientId2seqId: map[14132034788159840:12 170493724742603415:11 329716370506775378:4 381007544930920199:10 1140221141850216790:12 1806007842906888491:11 1913053818396621436:10 2300578112370263533:4 4061258987902816970:10 4187485241530242204:14 4244922913676436403:3],
 shardState: map[0:NOTREADY 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED],
 kv.moveInShards: map[0:100 1:100 2:100 3:100 4:100],
 lastConfig: {9 [100 100 100 100 100 100 100 100 100 100] map[100:[server-100-0 server-100-1 server-100-2]]},
 config: {10 [102 102 102 102 102 100 100 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}

Num:0 Shards:[0 0 0 0 0 0 0 0 0 0]
Num:1 Shards:[100 100 100 100 100 100 100 100 100 100]
Num:2 Shards:[102 102 102 102 102 100 100 100 100 100]
Num:3 Shards:[101 102 102 102 102 101 101 100 100 100]
Num:4 Shards:[100 102 102 102 102 100 102 100 100 100]
Num:5 Shards:[100 100 100 100 100 100 100 100 100 100]
Num:6 Shards:[102 102 102 102 102 100 100 100 100 100]
Num:7 Shards:[101 102 102 102 102 101 101 100 100 100]
Num:8 Shards:[100 102 102 102 102 100 102 100 100 100]
Num:9 Shards:[100 100 100 100 100 100 100 100 100 100]



challenge 
1.GC
num5，group0 向 group1 pull shard0 数据。group 0 获取后执行domoveshard，完成后向group1 发送del 请求删除 group1中的shard0数据。
要求：group1 shard0数据未删除的状态下，更新到num6。
假如发生这种情况：group1更新到num6，group1 向 group0 请求shard0 数据，

shardState状态增加至：
served 提供服务
notready 数据未准备好
ready 数据准备好，但远程数据未删除
del 需要被删除
notserved 不提供服务

对于 shard0 数据，若在当前config 需要从group0 转移到group1中，然后删除，它的状态变化流程为：
group1：notserved -> notready（向group0获取数据） -> ready（获得数据，向group0请求删除其本地数据） -> served
group0: served -> del -> notserved（接收到删除请求）

served 和 notserved为shard的在config下的最终状态，而notready，ready，del都是中间状态。若处在中间状态，便不能进行config更新

7. TestStaticShards bug4B_7
问题：
deadlock了
server.go 562, updateConfig()中，如果kv.mu.Lock()在for !kv.readyToUpdate(){前面，会导致updateConfig() deadlock。
解决方法：每次readyToUpdate后释放锁

***moveShard():group: 101, kvserver: 0, kv.moveInShards: map[0:100 1:100 2:100 3:100 4:100],  ----728
 kv.shardState: map[0:NOTREADY 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]
***deleteShard():group: 101, kvserver: 0, kv.moveInShards: map[0:100 1:100 2:100 3:100 4:100],  ----738 就一直不发送了
 kv.shardState: map[0:NOTREADY 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***kv.doUpdate-: group: 101, kvserver: 0, ------699
***kv.doUpdate-: group: 101, kvserver: 1, ------1136
***kv.doUpdate-: group: 101, kvserver: 2, ------1158
 lastConfig: {Num:1 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]},
 config: {Num:2 Shards:[101 101 101 101 101 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2]]},
 moveInShards: map[0:100 1:100 2:100 3:100 4:100]
 kv.shardState: map[0:NOTREADY 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***kv.doUpdate-: group: 100, kvserver: 2, ------1579
***kv.doUpdate-: group: 100, kvserver: 0, ------1666
***kv.doUpdate-: group: 100, kvserver: 1, ------1786
 lastConfig: {Num:1 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]},
 config: {Num:2 Shards:[101 101 101 101 101 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2]]},
 moveInShards: map[]
 kv.shardState: map[0:DEL 1:DEL 2:DEL 3:DEL 4:DEL 5:SERVED 6:SERVED 7:SERVED 8:SERVED 9:SERVED]


***callMoveShardData():group: 101, kvserver: 0, reply: {Err:OK ShardData:map[] ClientId2seqId:map[] ConfigNum:2},
 args: {ConfigNum:2 Shard:4}  ------1638
***callMoveShardData():group: 101, kvserver: 0, reply: {Err:OK ShardData:map[] ClientId2seqId:map[] ConfigNum:2},
 args: {ConfigNum:2 Shard:0}  ------1649
***callMoveShardData():group: 101, kvserver: 0, reply: {Err:OK ShardData:map[] ClientId2seqId:map[4509986624377234167:1] ConfigNum:2},
 args: {ConfigNum:2 Shard:2}  ------1752
***callMoveShardData():group: 101, kvserver: 0, reply: {Err:OK ShardData:map[] ClientId2seqId:map[4509986624377234167:1] ConfigNum:2},
 args: {ConfigNum:2 Shard:3}  ------1756
***callMoveShardData():group: 101, kvserver: 0, reply: {Err:OK ShardData:map[] ClientId2seqId:map[4509986624377234167:1] ConfigNum:2},
 args: {ConfigNum:2 Shard:1}  ------1777

***kv.waitApply: group: 101, kvserver: 0, ---------1926 发送了，却没有进domove
 Msg: {CommandValid:true Command:{Method:MoveShard ClientId:0 SeqId:0 Key: Value: Config:{Num:0 Shards:[0 0 0 0 0 0 0 0 0 0] Groups:map[]} Shard:4 ShardData:map[] ClientId2seqId:map[] ConfigNum:2} CommandIndex:3 SnapshotValid:false Snapshot:[] SnapshotTerm:0 SnapshotIndex:0}

***kv.doMoveShard-: group: 101, kvserver: 1, ---2173
***kv.doMoveShard-: group: 101, kvserver: 2, ---2177  kvserver0不见了
 opMsg: {Method:MoveShard ClientId:0 SeqId:0 Key: Value: Config:{Num:0 Shards:[0 0 0 0 0 0 0 0 0 0] Groups:map[]} Shard:1 ShardData:map[] ClientId2seqId:map[4509986624377234167:1] ConfigNum:2},
 kv.dataset: map[]
 kv.moveInShards: map[0:100 1:100 2:100 3:100 4:100],
 kv.shardState:map[0:READY 1:READY 2:READY 3:READY 4:READY 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]


8. TestConcurrent3  bug4B_8

***updateConfig(): group: 100, kvserver: 2, kv.configNum: 5, ------89992
 send new config: {Num:6 Shards:[102 102 102 102 102 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}

***kv.doUpdate: group: 100, kvserver: 2, ------95649
 lastConfig: {Num:5 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]},
 config: {Num:6 Shards:[102 102 102 102 102 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[]
 kv.shardState: map[0:SERVED 1:SERVED 2:SERVED 3:SERVED 4:SERVED 5:SERVED 6:SERVED 7:SERVED 8:SERVED 9:SERVED]

***kv.doUpdate-: group: 100, kvserver: 2, ------95683
 lastConfig: {Num:5 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]},
 config: {Num:6 Shards:[102 102 102 102 102 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[]
 kv.shardState: map[0:DEL 1:DEL 2:DEL 3:DEL 4:DEL 5:SERVED 6:SERVED 7:SERVED 8:SERVED 9:SERVED]

***updateConfig(): group: 100, kvserver: 2, kv.configNum: 6, ------96063
 send new config: {Num:7 Shards:[101 102 102 102 102 101 101 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}

***kv.doUpdate: group: 100, kvserver: 0, ------96865  shardState有问题
***kv.doUpdate: group: 100, kvserver: 1, ------96865
***kv.doUpdate: group: 100, kvserver: 2, ------103863
 lastConfig: {Num:6 Shards:[102 102 102 102 102 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {Num:7 Shards:[101 102 102 102 102 101 101 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[]
 kv.shardState: map[0:DEL 1:DEL 2:DEL 3:DEL 4:DEL 5:SERVED 6:SERVED 7:SERVED 8:SERVED 9:SERVED]

*-----------check------------98420

***moveShard():group: 100, kvserver: 2, is waiting kv.moveInShards: map[],  -----98773  一直重复
 kv.shardState: map[0:DEL 1:DEL 2:DEL 3:DEL 4:DEL 5:SERVED 6:SERVED 7:SERVED 8:SERVED 9:SERVED]

***updateConfig(): group: 102, kvserver: 1, kv.configNum: 6, ------106107
send new config: {Num:7 Shards:[101 102 102 102 102 101 101 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}
***updateConfig(): group: 102, kvserver: 1, kv.configNum: 7, ------106746
 send new config: {Num:8 Shards:[100 102 102 102 102 100 102 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}
***updateConfig(): group: 101, kvserver: 0, kv.configNum: 7, ------106867
send new config: {Num:8 Shards:[100 102 102 102 102 100 102 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}


***ck.Get: num: 25, config.Shards: [100 100 100 100 100 100 100 100 100 100], ----107480
 args: {Key:2 Op:Get ClientId:2187975337139698441 SeqId:13}


***updateConfig(): group: 102, kvserver: 1, kv.configNum: 8, ------107491
send new config: {Num:9 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]}


9. TestConcurrent3  bug4B_9
问题：
group102重启之后，group102无法回溯到最新的状态。因为重启后若没有新的entry加入，raft无法更新commit，从而使raft一直不回溯以前apply的entry。
又因为检测更新config的协程一直无法进行，客户端又恰好没有给group102发送请求，导致了raft一直没有新entry加入。造成group102 停止服务的活锁状态。

解决方案：
每当重启后，在kv层面检测leader是否有当前term的entry，没有的话发送空entry，使得commit进行更新。

***updateConfig(): group: 102, kvserver: 1,  --------54174
get config: {Num:5 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]}
 kv.config :{Num:4 Shards:[100 102 102 102 102 100 102 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}
 kv.shardState: map[0:NOTSERVED 1:SERVED 2:SERVED 3:SERVED 4:SERVED 5:NOTSERVED 6:SERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***updateConfig(): group: 101, kvserver: 2,  --------54843
get config: {Num:6 Shards:[102 102 102 102 102 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}
 kv.config :{Num:5 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]}
 kv.shardState: map[0:NOTSERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***updateConfig(): group: 101, kvserver: 2,  --------55742
get config: {Num:7 Shards:[101 102 102 102 102 101 101 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}
 kv.config :{Num:6 Shards:[102 102 102 102 102 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}
 kv.shardState: map[0:NOTSERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***kv.doDelShard-: group: 102, kvserver: 2, opMsg: {Method:DelShard ClientId:0 SeqId:0 Key: Value:  ---------59792
 Config:{Num:0 Shards:[0 0 0 0 0 0 0 0 0 0] Groups:map[]} Shard:6 ShardData:map[] ClientId2seqId:map[] ConfigNum:5},
 kv.dataset: map[3:bO7ty-bXv3qE-2dIOmg_Ctu7b]
 kv.moveInShards: map[],
 kv.shardState:map[0:NOTSERVED 1:DEL 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***updateConfig(): group: 102, kvserver: 1,   -------60702
get config: {Num:6 Shards:[102 102 102 102 102 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}
 kv.config :{Num:5 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]}
 kv.shardState: map[0:NOTSERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***kv.doUpdate-: group: 102, kvserver: 1, --------------------------60867
***kv.doUpdate-: group: 102, kvserver: 2, --------------------------61135
***kv.doUpdate-: group: 102, kvserver: 0, --------------------------61184
 lastConfig: {Num:5 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]},
 config: {Num:6 Shards:[102 102 102 102 102 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[0:100 1:100 2:100 3:100 4:100]
 kv.shardState: map[0:NOTREADY 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***moveShard():group: 102, kvserver: 1, kv.moveInShards: map[0:100 1:100 2:100 3:100 4:100], ----------61259
 kv.shardState: map[0:NOTREADY 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***updateConfig(): group: 100, kvserver: 0,  -------61519
get config: {Num:6 Shards:[102 102 102 102 102 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]}
 kv.config :{Num:5 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]}
 kv.shardState: map[0:SERVED 1:SERVED 2:SERVED 3:SERVED 4:SERVED 5:SERVED 6:SERVED 7:SERVED 8:SERVED 9:SERVED]

***callMoveShardData():group: 102, kvserver: 1, ok: true, ------------61607
 reply: {Err:ErrNotReady ShardData:map[] ClientId2seqId:map[] ConfigNum:0},
 args: {ConfigNum:6 Shard:0}


***kv.doUpdate-: group: 100, kvserver: 2, ----62466 0,1 完成
 lastConfig: {Num:5 Shards:[100 100 100 100 100 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2]]},
 config: {Num:6 Shards:[102 102 102 102 102 100 100 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 moveInShards: map[]
 kv.shardState: map[0:DEL 1:DEL 2:DEL 3:DEL 4:DEL 5:SERVED 6:SERVED 7:SERVED 8:SERVED 9:SERVED]

***callMoveShardData():group: 102, kvserver: 1, ok: true, -----------62512
 reply: {Err:OK ShardData:map[2:p2PUN7GgV3PZR] ClientId2seqId:map[316561585219227388:6 411526466011830850:12 870666152927832425:11 1661727765518553102:16 1950584313423549176:29 2590779368005342239:28 2618381405654201530:31 3074910346694159043:35 3352056610770962376:10 3702934036555916934:18 4428784096366139686:19] ConfigNum:6},
 args: {ConfigNum:6 Shard:0}
***callMoveShardData():group: 102, kvserver: 1, ok: true, ------------62561
 reply: {Err:OK ShardData:map[5:UZkq_UmbAw0V_sw_6ZjV] ClientId2seqId:map[316561585219227388:6 411526466011830850:12 870666152927832425:11 1661727765518553102:16 1950584313423549176:29 2590779368005342239:28 2618381405654201530:31 3074910346694159043:35 3352056610770962376:10 3702934036555916934:18 4428784096366139686:19] ConfigNum:6},
 args: {ConfigNum:6 Shard:3}
***callMoveShardData():group: 102, kvserver: 1, ok: true, ------------62655
 reply: {Err:OK ShardData:map[3:bO7ty-bXv3qE-2dIOmg_Ctu7b1XM2] ClientId2seqId:map[316561585219227388:6 411526466011830850:12 870666152927832425:11 1661727765518553102:16 1950584313423549176:29 2590779368005342239:28 2618381405654201530:31 3074910346694159043:35 3352056610770962376:10 3702934036555916934:18 4428784096366139686:19] ConfigNum:6},
 args: {ConfigNum:6 Shard:1}
***callMoveShardData():group: 102, kvserver: 1, ok: true, ------------62696
 reply: {Err:OK ShardData:map[4:5dc7sMiURhk7EqaBJSe] ClientId2seqId:map[316561585219227388:6 411526466011830850:12 870666152927832425:11 1661727765518553102:16 1950584313423549176:29 2590779368005342239:28 2618381405654201530:31 3074910346694159043:35 3352056610770962376:10 3702934036555916934:18 4428784096366139686:19] ConfigNum:6},
 args: {ConfigNum:6 Shard:2}
***callMoveShardData():group: 102, kvserver: 1, ok: true, ------------62705
 reply: {Err:OK ShardData:map[6:75ieoHjFpqCRZOzkc] ClientId2seqId:map[316561585219227388:6 411526466011830850:12 870666152927832425:11 1661727765518553102:16 1950584313423549176:29 2590779368005342239:28 2618381405654201530:31 3074910346694159043:35 3352056610770962376:10 3702934036555916934:18 4428784096366139686:19] ConfigNum:6},
 args: {ConfigNum:6 Shard:4}

*-----------shutdown 0 1 2, start 0 1 2------------63026

***kv.doMoveShard-: group: 102, kvserver: 1,  --------63127
opMsg: {Method:MoveShard ClientId:0 SeqId:0 Key: Value: Config:{Num:0 Shards:[0 0 0 0 0 0 0 0 0 0] Groups:map[]} Shard:0 ShardData:map[2:p2PUN7GgV3PZR] ClientId2seqId:map[316561585219227388:6 411526466011830850:12 870666152927832425:11 1661727765518553102:16 1950584313423549176:29 2590779368005342239:28 2618381405654201530:31 3074910346694159043:35 3352056610770962376:10 3702934036555916934:18 4428784096366139686:19] ConfigNum:6},
 kv.dataset: map[2:p2PUN7GgV3PZR]
 kv.moveInShards: map[0:100 1:100 2:100 3:100 4:100],
 kv.shardState:map[0:READY 1:NOTREADY 2:NOTREADY 3:NOTREADY 4:NOTREADY 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]
******************************************************************************************


***kv.readDataset: load group: 102, kvserver: 0, ---------------63305
***kv.readDataset: load group: 102, kvserver: 1, ---------------63322
***kv.readDataset: load group: 102, kvserver: 2, ---------------63337
 dataset: map[3:bO7ty-bXv3qE-2dIOmg_Ctu7b 8:TzBztce],
 clientId2seqId: map[316561585219227388:6 411526466011830850:8 870666152927832425:6 1661727765518553102:16 1950584313423549176:3 2590779368005342239:24 2618381405654201530:8 3074910346694159043:9 3352056610770962376:10 3702934036555916934:18 4428784096366139686:16],
 shardState: map[0:NOTSERVED 1:DEL 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTSERVED 6:DEL 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED],
 kv.moveInShards: map[],
 lastConfig: {4 [100 102 102 102 102 100 102 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {5 [100 100 100 100 100 100 100 100 100 100] map[100:[server-100-0 server-100-1 server-100-2]]}

group102重启之后，group102无法回溯到最新的状态。

***moveShard():group: 102, kvserver: 2, is waiting kv.moveInShards: map[], -----------------67681
 kv.shardState: map[0:NOTSERVED 1:DEL 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTSERVED 6:DEL 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***deleteShard():group: 102, kvserver: 2, kv.moveInShards: map[], -----------------67693
 kv.shardState: map[0:NOTSERVED 1:DEL 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTSERVED 6:DEL 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

****************************检查 group102 shutdown之间是否正常
***kv.doDelShard: group: 102, kvserver: 0, --------------57411
***kv.doDelShard: group: 102, kvserver: 2, opMsg: {Method:DelShard ClientId:0 SeqId:0 Key: Value:  ------57443
 Config:{Num:0 Shards:[0 0 0 0 0 0 0 0 0 0] Groups:map[]} Shard:3 ShardData:map[] ClientId2seqId:map[] ConfigNum:5},
 kv.dataset: map[3:bO7ty-bXv3qE-2dIOmg_Ctu7b 4:5dc7sMiURhk7EqaBJSe 5:UZkq_UmbAw0V_sw_6 6:75ieoHjFpqCRZOzkc 8:TzBztce]
 kv.moveInShards: map[],
 kv.shardState:map[0:NOTSERVED 1:DEL 2:DEL 3:DEL 4:DEL 5:NOTSERVED 6:DEL 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]


***kv.doDelShard: group: 102, kvserver: 1,   -----------59290
***kv.doDelShard: group: 102, kvserver: 0,   -----------59523
***kv.doDelShard: group: 102, kvserver: 2, opMsg: {Method:DelShard ClientId:0 SeqId:0 Key: Value: ------59650
Config:{Num:0 Shards:[0 0 0 0 0 0 0 0 0 0] Groups:map[]} Shard:4 ShardData:map[] ClientId2seqId:map[] ConfigNum:5},
 kv.dataset: map[3:bO7ty-bXv3qE-2dIOmg_Ctu7b 4:5dc7sMiURhk7EqaBJSe 6:75ieoHjFpqCRZOzkc 8:TzBztce]
 kv.moveInShards: map[],
 kv.shardState:map[0:NOTSERVED 1:DEL 2:DEL 3:NOTSERVED 4:DEL 5:NOTSERVED 6:DEL 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***kv.doDelShard: group: 102, kvserver: 1,  -----------59446
***kv.doDelShard: group: 102, kvserver: 0,  -----------59534
***kv.doDelShard: group: 102, kvserver: 2, opMsg: {Method:DelShard ClientId:0 SeqId:0 Key: Value:  -----59675
 Config:{Num:0 Shards:[0 0 0 0 0 0 0 0 0 0] Groups:map[]} Shard:2 ShardData:map[] ClientId2seqId:map[] ConfigNum:5},
 kv.dataset: map[3:bO7ty-bXv3qE-2dIOmg_Ctu7b 4:5dc7sMiURhk7EqaBJSe 8:TzBztce]
 kv.moveInShards: map[],
 kv.shardState:map[0:NOTSERVED 1:DEL 2:DEL 3:NOTSERVED 4:NOTSERVED 5:NOTSERVED 6:DEL 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

***kv.waitApply: Snapshot: group: 102, kvserver: 0, --------------59564
***kv.waitApply: Snapshot: group: 102, kvserver: 1, --------------59657
***kv.waitApply: Snapshot: group: 102, kvserver: 2, --------------59777
 dataset: map[3:bO7ty-bXv3qE-2dIOmg_Ctu7b 8:TzBztce],
 clientId2seqId: map[316561585219227388:6 411526466011830850:8 870666152927832425:6 1661727765518553102:16 1950584313423549176:3 2590779368005342239:24 2618381405654201530:8 3074910346694159043:9 3352056610770962376:10 3702934036555916934:18 4428784096366139686:16],
 shardState: map[0:NOTSERVED 1:DEL 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTSERVED 6:DEL 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED],
 kv.moveInShards map[],
 lastConfig: {4 [100 102 102 102 102 100 102 100 100 100] map[100:[server-100-0 server-100-1 server-100-2] 102:[server-102-0 server-102-1 server-102-2]]},
 config: {5 [100 100 100 100 100 100 100 100 100 100] map[100:[server-100-0 server-100-1 server-100-2]]},
 lastApplied: 99


删除shard6
***kv.doDelShard: group: 102, kvserver: 0, ------59575
***kv.doDelShard: group: 102, kvserver: 1, opMsg: {Method:DelShard ClientId  -----59710
***kv.doDelShard: group: 102, kvserver: 2, opMsg: {Method:DelShard ClientId:0 SeqId:0 Key: Value:  --------59788
Config:{Num:0 Shards:[0 0 0 0 0 0 0 0 0 0] Groups:map[]} Shard:6 ShardData:map[] ClientId2seqId:map[] ConfigNum:5},
 kv.dataset: map[3:bO7ty-bXv3qE-2dIOmg_Ctu7b 8:TzBztce]
 kv.moveInShards: map[],
 kv.shardState:map[0:NOTSERVED 1:DEL 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTSERVED 6:DEL 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

删除shard1
***kv.doDelShard: group: 102, kvserver: 1,  ----------59736
***kv.doDelShard: group: 102, kvserver: 0,  ----------60258
***kv.doDelShard: group: 102, kvserver: 2, opMsg: {Method:DelShard ClientId:0 SeqId:0 Key: Value: ---------60333
 Config:{Num:0 Shards:[0 0 0 0 0 0 0 0 0 0] Groups:map[]} Shard:1 ShardData:map[] ClientId2seqId:map[] ConfigNum:5},
 kv.dataset: map[3:bO7ty-bXv3qE-2dIOmg_Ctu7b]
 kv.moveInShards: map[],
 kv.shardState:map[0:NOTSERVED 1:DEL 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]

*-----------wait done------------117604

10.  TestChallenge1Delete bug4B_10
问题：
可能是有些entry中带有dataset中的数据，即使log的长度小于10，也非常的大
还有一个原因是，snapshot没有来得及更新，即使dataset的数据删除了key0，snapshot的数据仍旧保留了key0的数据

解决方案：
降低snaphot的阈值，改为 每commit就snapshot一次。
或者，每当dataset变化时，snapshot一次，但是这样，若是遇到大量的get操作，就会导致log长度增大，所以采用上面的方法。

1
*------gi: 0, i: 0, raft: 482, snap: 13699------
*------gi: 0, i: 1, raft: 543, snap: 13699------
*------gi: 0, i: 2, raft: 543, snap: 13699------
*------gi: 1, i: 0, raft: 472, snap: 4641------
*------gi: 1, i: 1, raft: 472, snap: 4641------
*------gi: 1, i: 2, raft: 472, snap: 4641------
*------gi: 2, i: 0, raft: 542, snap: 13699------
*------gi: 2, i: 1, raft: 480, snap: 13699------
*------gi: 2, i: 2, raft: 542, snap: 13699------

5.
***updateConfig(): group: 101, kvserver: 2, -------64702
 get config: {Num:15 Shards:[102 101 101 101 101 102 102 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}
 kv.config :{Num:15 Shards:[102 101 101 101 101 102 102 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}
 kv.shardState: map[0:NOTSERVED 1:SERVED 2:SERVED 3:SERVED 4:SERVED 5:NOTSERVED 6:NOTSERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]
 keys: [3 4 6 5]

***updateConfig(): group: 102, kvserver: 0,  -------64750
get config: {Num:15 Shards:[102 101 101 101 101 102 102 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}
 kv.config :{Num:15 Shards:[102 101 101 101 101 102 102 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}
 kv.shardState: map[0:SERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:SERVED 6:SERVED 7:NOTSERVED 8:NOTSERVED 9:NOTSERVED]
 keys: [20 23 24 8 27 22 25 26 2 28 7 29 21]

***updateConfig(): group: 100, kvserver: 0,  -------65112
get config: {Num:15 Shards:[102 101 101 101 101 102 102 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}
 kv.config :{Num:15 Shards:[102 101 101 101 101 102 102 100 100 100] Groups:map[100:[server-100-0 server-100-1 server-100-2] 101:[server-101-0 server-101-1 server-101-2] 102:[server-102-0 server-102-1 server-102-2]]}
 kv.shardState: map[0:NOTSERVED 1:NOTSERVED 2:NOTSERVED 3:NOTSERVED 4:NOTSERVED 5:NOTSERVED 6:NOTSERVED 7:SERVED 8:SERVED 9:SERVED]
 keys: [9 13 17 0 14 15 11 10 19 12 16 18 1]

*------gi: 0, i: 0, raft: 731, snap: 13699------
*------gi: 0, i: 1, raft: 731, snap: 13699------
*------gi: 0, i: 2, raft: 731, snap: 13699------
*------gi: 1, i: 0, raft: 721, snap: 15618------ 只有4个key，snap的大小本应该只有4kb左右
*------gi: 1, i: 1, raft: 721, snap: 15618------
*------gi: 1, i: 2, raft: 721, snap: 15618------
*------gi: 2, i: 0, raft: 659, snap: 13699------
*------gi: 2, i: 1, raft: 597, snap: 13699------
*------gi: 2, i: 2, raft: 659, snap: 13699------

10
*------gi: 0, i: 0, raft: 1095, snap: 15697------
*------gi: 0, i: 1, raft: 1095, snap: 15697------
*------gi: 0, i: 2, raft: 1095, snap: 15697------
*------gi: 1, i: 0, raft: 664, snap: 15700------
*------gi: 1, i: 1, raft: 664, snap: 15700------
*------gi: 1, i: 2, raft: 664, snap: 15700------
*------gi: 2, i: 0, raft: 14064, snap: 1641------
*------gi: 2, i: 1, raft: 14064, snap: 1641------
*------gi: 2, i: 2, raft: 14064, snap: 1641------


