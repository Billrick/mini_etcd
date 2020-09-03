package register

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/rpc"
	"sync"
	"time"
	"z.cn/RaftImpl/model"
	"z.cn/RaftImpl/store"
)

const (
	Leader    = iota //领导者
	Follower         //跟随者
	Candidate        //候选人

	//command
	Canvass        //选举
	Heart          //心跳
	LogReplication //日志复制

)

var register *Register

type Register struct {
	*Node
	othersNode    map[string]*Node //其他节点
	failedNode    chan *Node
	lastHeartTime time.Time //最后心跳时间
	mx            sync.RWMutex
	pool          map[string]*rpc.Client //节点链接
}

func NewRegister(name, addr string, otherNode map[string]*Node) (*Register, error) {
	store, err := store.NewStore(name + ".log")
	if err != nil {
		return nil, err
	}
	r := &Register{
		Node: &Node{
			Name:         name,
			Address:      addr,
			Role:         Follower,
			State:        0,
			Time:         0,
			CanvassNum:   0,
			CanvassFlag:  true,
			heartTimeout: make(chan time.Time, 1),
			store:        store,
			tmpLog:       make(chan *model.RequestBody, 100),
		},
		othersNode: otherNode,
		failedNode: make(chan *Node, 2),
		pool:       make(map[string]*rpc.Client, 5),
	}
	register = r
	go r.registerService()
	return r, nil
}

func (r *Register) registerService() error {
	//添加put方法
	http.HandleFunc("/put", putHandler)
	//注册服务
	rpc.Register(new(Register))
	//把服务处理绑定到http协议上
	rpc.HandleHTTP()
	fmt.Println("start rpc server at", r.Address)
	err := http.ListenAndServe(r.Address, nil)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func RegisterNode(name, addr []string) (otherNode map[string]*Node, err error) {
	otherNode = make(map[string]*Node, 5)
	if len(name) == len(addr) {
		for i, _ := range addr {
			node := &Node{
				Name:        name[i],
				Address:     addr[i],
				Role:        Follower,
				State:       0,
				Time:        0,
				CanvassNum:  0,
				CanvassFlag: true,
			}
			otherNode[node.Name] = node
		}

	} else {
		return nil, errors.New("cluster address and name no equal")
	}

	return
}

type Node struct {
	Name         string //节点名称
	Address      string //本节点IP端口
	Role         int    //当前角色
	State        int    //节点状态
	Leader       string
	Time         int  //选举次数
	CanvassFlag  bool //选票状态  true 还有未投选票, false 已投选票
	CanvassNum   int
	heartTimeout chan time.Time
	tmpLog       chan *model.RequestBody
	store        *store.Store //日志数据的存储
}

type CommandMsg struct {
	Command     int
	Msg         string
	LogCommand  model.RequestBody
	Node        Node
	CanvassFlag bool
	Err         error
}

func NewCommandMsg(Command int, Msg string, node *Node, canvassFlag bool) *CommandMsg {
	return &CommandMsg{
		Command:     Command,
		Msg:         Msg,
		Node:        *node,
		CanvassFlag: canvassFlag,
	}
}

//NodeCommunication 节点间通讯
func NodeCommunication(r *Register) {
	for _, node := range r.othersNode {
		go r.connectOthderNode(node)
	}
	//尝试重新链接掉线的节点
	go r.tryFailedNode()
	//监听与leader超时时间
	go listenerTimeOut()
	//leader发送心跳自动任务
	go heartTask()
	// 竞选自动任务,与leader超时连接,开始竞选
	go canvassTask()
}

// canvassTask 竞选自动任务
func canvassTask() {
	for {
		select {
		case <-time.Tick(register.randomTimeOut(Canvass)):
			if register.Role != Leader {
				register.requestCanvass()
			}
		}
	}
}

//heartTask 发送心跳自动任务
func heartTask() {
	for {
		select {
		case <-time.Tick(register.randomTimeOut(Heart)):
			register.sendHeart()
		}
	}
}

// listenerTimeOut 监听与leader的超时时间
func listenerTimeOut() {
	for {
		select { //监控超时
		case lastTime := <-register.heartTimeout:
			register.lastHeartTime = lastTime
			if register.Role == Follower && time.Now().Sub(lastTime) > register.randomTimeOut(Candidate) {
				register.Role = Candidate
			}
		default: //与leader最后连接时间超时时,成为候选者
			if register.Role == Follower && time.Now().Sub(register.lastHeartTime) > register.randomTimeOut(Candidate) {
				register.Role = Candidate
			}
			time.Sleep(time.Millisecond * 300)
		}
	}
}

//节点连接
func (r *Register) connectOthderNode(node *Node) {
	client, err := rpc.DialHTTP("tcp", node.Address)
	//fmt.Println("start connect other node :", node.Address)
	if err != nil {
		//fmt.Println("listener create error", err)
		r.failedNode <- node //把失败的节点放入map,启动后置任务,尝试重新连接
	} else {
		fmt.Println("listener create success at", node.Address)
		r.pool[node.Name] = client
	}
}

//处理连接失败的节点
func (r *Register) tryFailedNode() {
	for {
		select {
		case failedNode := <-r.failedNode:
			//fmt.Println("try reconnect listener")
			r.connectOthderNode(failedNode)
			time.Sleep(time.Millisecond * 500)
		default:
			time.Sleep(time.Second * 3)
		}

	}
}

//update 修改本节点信息
func (r *Register) update(node *Node) {
	r.mx.Lock()
	defer r.mx.Unlock()
	if onode, ok := r.othersNode[node.Name]; ok {
		onode.Role = node.Role
		onode.State = node.State
		onode.Leader = node.Leader
		r.Node.CanvassNum = 0
		r.Node.CanvassFlag = true
	}
}

//randomTimeOut 获取随机超时时间
func (r *Register) randomTimeOut(command int) time.Duration {
	switch command {
	case Candidate:
		return 2 * time.Second
	case Heart:
		return time.Duration(r.randInt(100, 150)) * time.Millisecond
	default:
		return time.Duration(r.randInt(1500, 3000)) * time.Millisecond
	}
}

func (r *Register) randInt(min, max int64) int64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return rand.Int63n(max-min) + min
}

//主节点向其他子节点发送心跳
func (r *Register) sendHeart() error {
	if r.Leader == r.Name {
		for _, node := range r.othersNode {
			if client, ok := r.pool[node.Name]; ok { //leader是本节点时,向其他节点发心跳包
				res := NewCommandMsg(Heart, "", node, false)
				err := client.Call("Register.Heart", CommandMsg{Command: Heart, Node: *r.Node}, res)
				if err != nil {
					fmt.Println("failed rpc service register.Heart", err)
					delete(r.pool, node.Name)
					r.failedNode <- node
					return err
				}
				fmt.Printf("%s leader %s send heart to childNode %s \n", time.Now().Format("2009-01-02 03:04:05"), r.Name, node.Name)
			}
		}
	}
	return nil
}

//请求竞选
func (r *Register) requestCanvass() error {
	if (len(r.pool)) < 1 {
		r.Leader = r.Name
		r.Role = Leader
		fmt.Println("auto leader")
	} else {
		if r.Role == Candidate { //如果超时
			for _, node := range r.othersNode {
				if client, ok := r.pool[node.Name]; ok { //如果本届点状态为候选人，向其他节点发送请求选票
					fmt.Printf("Candidate %s request Canvass from othersNode %s\n", r.Name, node.Name)
					res := CommandMsg{}
					err := client.Call("Register.Canvass", CommandMsg{Command: Canvass, Node: *r.Node}, &res)
					if err != nil {
						fmt.Println("failed rpc service register.Canvass", err)
						delete(r.pool, node.Name)
						r.failedNode <- node
						return err
					}
					if res.CanvassFlag {
						r.CanvassNum++
					}
					fmt.Println(res.Node.Name, " give ", r.Name, res.CanvassFlag, ",计数：", r.CanvassNum)
					if r.CanvassNum > 0 && (r.CanvassNum > (len(r.othersNode)-len(r.failedNode)) || r.CanvassNum >= (len(r.othersNode))/2) { //获取当前存活节点的数量
						r.Leader = r.Name
						r.Role = Leader
						fmt.Println(">2/1 leader")
					}
				}
			}
		}
	}
	return nil
}

//主节点收到 数据结果时将命令传输到其他节点
func (r *Register) sendLogReplication(body model.RequestBody) error {
	if r.Leader == r.Name {
		for _, node := range r.othersNode {
			if client, ok := r.pool[node.Name]; ok { //leader是本节点时,向其他节点发心跳包
				res := NewCommandMsg(Heart, "", node, false)
				err := client.Call("Register.LogReplication", CommandMsg{Command: LogReplication, LogCommand: body, Node: *r.Node}, res)
				if err != nil {
					fmt.Println("failed rpc service LogReplication", err)
					delete(r.pool, node.Name)
					r.failedNode <- node
					return err
				} else {
					register.store.Resolve(body)
				}
				fmt.Printf("%s leader %s send LogReplication to childNode %s \n", time.Now().Format("2009-01-02 03:04:05"), r.Name, node.Name)
			}
		}
	}
	return nil
}

func (r *Register) Heart(req CommandMsg, res *CommandMsg) error {
	fmt.Println(time.Now().Format("2009-01-02 03:04:05"), "收到心跳...")
	//同一任期时 更新本地节点信息
	register.update(&req.Node)
	register.Leader = req.Node.Name
	register.Role = Follower
	if register.Time == req.Node.Time {
		res.Node = *register.Node
	} else if register.Time < req.Node.Time { // 如果leader节点任期大于子节点 更新节点任期
		register.Time = req.Node.Time
	}
	register.heartTimeout <- time.Now()
	return nil
}

func (r *Register) Canvass(req CommandMsg, res *CommandMsg) error {
	res.Command = Canvass
	res.CanvassFlag = register.Node.CanvassFlag
	res.Node = *register.Node
	if register.CanvassFlag {
		register.Node.CanvassFlag = !register.Node.CanvassFlag
	}
	return nil
}

func (r *Register) LogReplication(req CommandMsg, res *CommandMsg) error {
	res.Command = LogReplication
	log := req.LogCommand
	value, err := register.store.Resolve(log)
	if err != nil {
		res.Msg = value
		res.Err = err
		return nil
	}
	fmt.Println(register.Name, "收到", req.Node.Name, "的日志命令:", log)
	return nil
}
