/*
 * @Author: bz2021
 * @Date: 2023-12-16 10:41:12
 * @Description:
 */
package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

// GobCodec 通过 gob 进行编解码
type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	// 创建一个带有缓冲区的 Writer，可以减少实际的 IO 次数，提高效率
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		// 在执行 enc.Encode() 时，会先将数据写入到缓冲区，直到缓冲区满或者调用 Flush 或者 Close 才会一次性写入到 conn
		enc: gob.NewEncoder(buf),
	}
}

func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *GobCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		// 刷新输出流缓冲区
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()
	if err := c.enc.Encode(h); err != nil {
		log.Println("rpc codec: gob error encoding header:", err)
		return err
	}
	if err := c.enc.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding body:", err)
		return err
	}
	return nil
}

func (c *GobCodec) Close() error {
	return c.conn.Close()
}
