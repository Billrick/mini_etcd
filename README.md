### mini ETCD
- 一、raft内置的选主协议，用于选出主节点,raft的时钟周期以及超时机制
    - Leader    //领导者 
	- Follower  //跟随者
	- Candidate //候选人 当与leader无法联接时，通过超时机制进行候选人拉票
    