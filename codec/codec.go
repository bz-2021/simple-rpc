/*
 * @Author: bz2021
 * @Date: 2023-12-16 10:24:06
 * @Description: 客户端发送的请求：服务名 Arith，方法名 Multiply，参数 Args。服务端响应：错误 error，返回值 reply
 * 将响应和请求中的参数和返回值抽象为 body，剩余的信息放在 header 中
 */
package codec

import "io"

type Header struct {
	ServiceMethod string // 服务名和方法名，与 Go 中的结构体和方法相映射
	Seq           uint64 // 客户选择的序列号
	Error         string
}

// Codec 对消息进行编解码的接口
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

const (
	GobType Type = "application/gob"
	JsonType Type = "application/json"
)

var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}