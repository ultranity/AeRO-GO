package message

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
)

var ErrInvalidMsg = errors.New("invalid Message")
var ErrUnknownMsg = errors.New("unknown Message Type")

const (
	RequestFlag   = '<'
	ResponseFlag  = '>'
	TypeLogin     = 'l'
	TypeProxy     = 'p'
	TypeHeartbeat = 'h'
	TypePipe      = 'q'
	TypeConn      = 'w'
)

var TypeReqMap = map[byte]interface{}{
	TypeHeartbeat: HeartbeatRequest{},
	TypeLogin:     LoginRequest{},
	TypeProxy:     ProxyRequest{},
	TypePipe:      PipeRequest{},
	TypeConn:      ConnRequest{},
}

var TypeRespMap = map[byte]interface{}{
	TypeHeartbeat: HeartbeatResponse{},
	TypeLogin:     LoginResponse{},
	TypeProxy:     ProxyResponse{},
}

type Message interface{}

// login server
type LoginRequest struct {
	Version  uint8  `json:"ver,omitempty"`
	AuthCode string `json:"xcode,omitempty"`
	PoolSize int    `json:"pool,omitempty"`
	HostName string `json:"host,omitempty"`
	OS       string `json:"os,omitempty"`
	Tag      string `json:"tag,omitempty"`
}

type LoginResponse struct {
	ClientId string `json:"cid,omitempty"`
	Result   string `json:"result,omitempty"`
}

// register proxy
type ProxyRequest struct {
	Name   string `json:"name,omitempty"`
	Type   string `json:"type,omitempty"`
	Port   string `json:"port,omitempty"` //0 由server分配
	Target string `json:"trgt,omitempty"`
	Enable bool   `json:"enable,omitempty"`
}

type ProxyResponse struct {
	Name   string `json:"name,omitempty"`
	Port   string `json:"port,omitempty"` //由server分配的port
	Enable bool   `json:"enable,omitempty"`
	Result string `json:"result,omitempty"`
}

// heartbeat
type HeartbeatRequest struct {
	Timestamp int64 `json:"t,omitempty"`
}

type HeartbeatResponse struct {
	TimeSend int64 `json:"t_send,omitempty"`
	TimeRecv int64 `json:"t_recv,omitempty"`
}

type PipeRequest struct {
	ClientId string `json:"cid,omitempty"`
}

type ConnRequest struct {
	ProxyName string `json:"name,omitempty"`
	SrcIp     string `json:"src_ip,omitempty"`
	SrcPort   string `json:"src_port,omitempty"`
	DstIp     string `json:"dst_ip,omitempty"`
	DstPort   string `json:"dst_port,omitempty"`
}

/*
message: [<|>:1 byte] + [type code:1 byte] + [length:4 byte] + [data:(length)]
"<" : request
">" : response
*/

func Write(flag, typ byte, data []byte, writer io.Writer) error {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(flag)
	buf.WriteByte(typ)
	binary.Write(buf, binary.BigEndian, int32(len(data)))
	buf.Write(data)
	_, err := writer.Write(buf.Bytes())
	return err
}

func Read(reader io.Reader) (flag byte, typ byte, data []byte, err error) {
	buf := make([]byte, 2)
	_, err = reader.Read(buf)
	if err != nil {
		return
	}
	flag = buf[0]
	typ = buf[1]
	var length int32
	if err = binary.Read(reader, binary.BigEndian, &length); err != nil {
		return
	}
	if length < 0 {
		err = ErrInvalidMsg
		return
	}
	data = make([]byte, length)
	n, err := io.ReadFull(reader, data)
	if err != nil {
		err = ErrInvalidMsg
	} else if int32(n) != length {
		err = ErrInvalidMsg
	}
	return
}

func Get(reader io.Reader) (Message, error) {
	flag, typ, data, err := Read(reader)
	if err != nil {
		return nil, err
	}
	var mtype interface{}
	var ok bool
	if flag == RequestFlag {
		mtype, ok = TypeReqMap[typ]
	} else if flag == ResponseFlag {
		mtype, ok = TypeRespMap[typ]
	}
	if !ok {
		return nil, fmt.Errorf("unknown Message Type %q %q %s", flag, typ, data)
	}
	msg := reflect.New(reflect.TypeOf(mtype)).Interface()
	err = json.Unmarshal(data, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func Send(msg Message, writer io.Writer) error {
	flag, typ, err := getMsgInfo(msg)
	if err != nil {
		return err
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return Write(flag, typ, data, writer)
}

func getMsgInfo(msg Message) (flag byte, typ byte, err error) {
	switch msg.(type) {
	case *LoginRequest:
		flag = RequestFlag
		typ = TypeLogin
	case *LoginResponse:
		flag = ResponseFlag
		typ = TypeLogin
	case *HeartbeatRequest:
		flag = RequestFlag
		typ = TypeHeartbeat
	case *HeartbeatResponse:
		flag = ResponseFlag
		typ = TypeHeartbeat
	case *ProxyRequest:
		flag = RequestFlag
		typ = TypeProxy
	case *ProxyResponse:
		flag = ResponseFlag
		typ = TypeProxy
	case *PipeRequest:
		flag = RequestFlag
		typ = TypePipe
	case *ConnRequest:
		flag = RequestFlag
		typ = TypeConn
	default:
		err = ErrUnknownMsg
	}
	return
}
