package conf

import (
	"fmt"
	ini "gopkg.in/ini.v1"
)

func GetCurrentConfig() (name, addr string, err error) {
	cfg, openerr := ini.Load("./conf/server.ini")
	if openerr != nil {
		fmt.Println("fialed open server.ini ,err", openerr)
	}
	name = cfg.Section("currentNode").Key("name").String()
	addr = cfg.Section("currentNode").Key("addr").String()
	return name, addr, openerr
}

func GetClusterConfig() (name, addr []string, err error) {
	cfg, openerr := ini.Load("./conf/server.ini")
	if openerr != nil {
		fmt.Println("fialed open server.ini ,err", openerr)
	}
	name = cfg.Section("cluster").Key("name").Strings(",")
	addr = cfg.Section("cluster").Key("addr").Strings(",")
	return name, addr, openerr
}
