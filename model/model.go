package model

type RequestBody struct {
	Method   string `json:"method"`
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsPutLog bool   `json:"IsPutLog"` //是否存入日志
}

type ResponseBody struct {
	Code int
	Msg  string
}
