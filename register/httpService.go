package register

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"z.cn/RaftImpl/model"
)

func putHandler(writer http.ResponseWriter, request *http.Request) {
	msg := model.ResponseBody{
		Code: 400,
		Msg:  "request method error",
	}

	if request.Method == http.MethodPut {
		defer request.Body.Close()
		data, err := ioutil.ReadAll(request.Body)
		if err != nil {
			msg.Code = 400
			msg.Msg = err.Error()
		}
		var rb model.RequestBody
		if err := json.Unmarshal(data, &rb); err != nil {
			msg.Code = 400
			msg.Msg = err.Error()
		}
		rb.Method = http.MethodPut
		rb.IsPutLog = true
		register.tmpLog <- &rb
		go register.sendLogReplication(rb)
		msg.Code = 200
		msg.Msg = "success"
		result, err := json.Marshal(&msg)
		if err != nil {
			msg.Msg = err.Error()
		}
		writer.Write(result)
	} else {
		writer.WriteHeader(400)
		data, err := json.Marshal(&msg)
		if err != nil {
			msg.Msg = err.Error()
		}
		writer.Write(data)
	}
}

func getHandler(writer http.ResponseWriter, request *http.Request) {
	msg := model.ResponseBody{
		Code: 400,
		Msg:  "request method error",
	}
	if request.Method == http.MethodGet {
		values := request.URL.Query()
		key := values.Get("key")
		v, err := register.store.Get(model.RequestBody{Method: http.MethodGet, Key: key})
		if err != nil {
			msg.Msg = err.Error()
			result, _ := json.Marshal(&msg)
			writer.Write(result)
		} else {
			msg.Code = 200
			msg.Msg = v
			result, _ := json.Marshal(&msg)
			writer.Write(result)
		}
	} else {
		writer.WriteHeader(400)
		data, err := json.Marshal(&msg)
		if err != nil {
			msg.Msg = err.Error()
		}
		writer.Write(data)
	}
}
