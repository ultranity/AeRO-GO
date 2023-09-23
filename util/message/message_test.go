package message

import (
	"bytes"
	"testing"
)

var pipe []byte

func TestSend(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	req := LoginRequest{
		Version:  1,
		AuthCode: "12345",
		HostName: "ling",
		OS:       "macOs",
	}
	resp := LoginResponse{
		ClientId: "223",
		Result:   "success",
	}
	err := Send(&req, buf)
	if err != nil {
		t.Fatal(err)
	}
	err = Send(&resp, buf)
	if err != nil {
		t.Fatal(err)
	}
	pipe = buf.Bytes()
	if len(pipe) < 1 {
		t.Error("buf write error")
	}
}

func TestGet(t *testing.T) {
	buf := bytes.NewReader(pipe)
	msg, err := Get(buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(msg)
	req, ok := msg.(*LoginRequest)
	if !ok {
		t.Error("get message error")
	}
	t.Log(req)
	msg, err = Get(buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(msg)
}
