package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"z.cn/RaftImpl/model"
	"z.cn/RaftImpl/util"
)

const key = "killllllllllllll" //16位

/*
数据存储
*/
type Data map[string]string

type Store struct {
	commandFile *os.File
	Data        Data
	Index       int64
	mx          sync.RWMutex
}

//初始化数据存储，打开日志命令
func NewStore(filename string) (*Store, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	return &Store{
		Data:        make(map[string]string, 30),
		commandFile: file,
	}, nil
}

func (s *Store) ReadLogCommand(index int) {
	reader := bufio.NewReader(s.commandFile)
	for {
		//读取一行
		command, err := reader.ReadString('\n') //根据换行符读取命令
		if err == io.EOF {                      // io:EOF 文件读取完毕:文件结束错误
			break
		}
		if err != nil {
			fmt.Println("read file err:", err)
			break
		}
		command = util.AesDecrypt(command, key) //解析
		tmp := model.RequestBody{}
		err = json.Unmarshal([]byte(command), &tmp)
		fmt.Println("解析命令:", tmp)
		if err != nil {
			fmt.Println("json Unmarshal err:", err)
			break
		}
		s.Resolve(tmp) //将日志中的command执行
	}
}

func (s *Store) Put(body model.RequestBody) error {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.Data[body.Key] = body.Value

	if body.IsPutLog { //是否存入日志
		body.IsPutLog = false //还原数据时,不再放入日志中.
		data, err := json.Marshal(&body)
		if err != nil {
			return err
		}
		command := string(data)
		command = util.AesEncrypt(command, key) + "\n"
		_, err = s.commandFile.WriteString(command)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Get(body model.RequestBody) (string, error) {
	s.mx.RLock()
	defer s.mx.RUnlock()
	if v, ok := s.Data[body.Key]; ok {
		return v, nil
	} else {
		return "", errors.New("no found")
	}
}

func (s *Store) Resolve(body model.RequestBody) (value string, err error) {
	if body.Method == http.MethodPut {
		err = s.Put(body)
	} else if body.Method == http.MethodGet {
		value, err = s.Get(body)
	} else {
		return "", errors.New("no command")
	}
	return
}
