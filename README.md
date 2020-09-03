### mini ETCD
- 一、raft内置的选主协议，用于选出主节点,raft的时钟周期以及超时机制
    - Leader    //领导者 
	- Follower  //跟随者
	- Candidate //候选人 当与leader无法联接时，通过超时机制进行候选人拉票
- 二、日志复制
    leader收到请求后,先向子节点发送操作命令,等待子节点响应后,leader根据响应进行保存数据与异常处理