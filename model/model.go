package model

type RequestBody struct {
	Method string `json:"method"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

type ResponseBody struct {
	Code int
	Msg  string
}
