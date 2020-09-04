package main

import (
	"fmt"
	"sync"
	"z.cn/RaftImpl/conf"
	"z.cn/RaftImpl/register"
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
	nodes, err := register.RegisterNode(names, addrs)
	if err != nil {
		fmt.Println("register node ,err", err)
		return
	}
	r, err := register.NewRegister(name, addr, nodes)
	if err != nil {
		fmt.Println("new register err", err)
	}
	register.NodeCommunication(r)

	var wg = &sync.WaitGroup{}

	wg.Add(1)
	wg.Wait()
}
