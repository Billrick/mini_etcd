package main

import (
	"fmt"
	"sync"
	"z.cn/RaftImpl/Register"
	"z.cn/RaftImpl/conf"
)

func main() {
	name, addr, err := conf.GetCurrentConfig()
	if err != nil {
		fmt.Println("fialed open server.ini ,err", err)
		return
	}
	names, addrs, err := conf.GetClusterConfig()
	if err != nil {
		fmt.Println("fialed open server.ini ,err", err)
		return
	}
	nodes, err := Register.RegisterNode(names, addrs)
	fmt.Println(nodes)
	if err != nil {
		fmt.Println("register node ,err", err)
		return
	}
	register, err := Register.NewRegister(name, addr, nodes)
	if err != nil {
		fmt.Println("new register err", err)
	}
	Register.NodeCommunication(register)

	var wg = &sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
