<!--
 * @Author: bz2021
 * @Date: 2023-12-18 17:20:11
 * @Description:  
-->

# 概述

- 从零实现了 `net/rpc` 包，具有基本调用功能
- 支持通过**不同的编码格式**进行序列化和反序列化
- 高性能客户端，支持**并发**和**异步**请求
- 支持客户端和服务端的**超时处理**
- 具有**注册中心**，通过心跳机制进行健康检查
- 具有**服务发现**功能，支持多种**负载均衡**算法

# 流程

![流程](./docs/process.png)

# 项目类图

![UML](./docs/uml.png)

# 遇到的问题和解决方法

## 传输 JSON 格式数据时 TCP Socket 粘包问题

![报文格式](docs/encode.png)

客户端在与服务端建立连接时执行 Dial 函数，会先将 Option 发送过去，这是协议交换阶段

``` go
type Option struct {
	MagicNumber    int
	CodecType      codec.Type
	ConnectTimeout time.Duration
	HandleTimeout  time.Duration
}

json.NewEncoder(conn).Encode(opt);
```

服务端执行 Accept 时会通过以下代码将其解析出来
``` go
json.NewDecoder(conn).Decode(&opt);
```

而后会进入 RPC 消息阶段，执行读取 Header 和 Body 的操作
``` go
err := cc.ReadHeader(&h);
```

在此过程中，如果客户端发送的 Option 数据没有被服务端及时接收，会和 Header 连在一起

json.Decoder 在执行时会将数据加载到缓存区，可能会影响 gob.Decoder 的执行

为了解决这个问题，我们将 Option 也通过 gob 的形式进行编码

实践验证，没有再出现相同的错误

## Goroutine 泄露问题

以下代码执行时，如果 `<-time.After(timeout)` 超时，则会导致 Goroutine 泄露

``` go
func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()
	called := make(chan struct{})
	sent := make(chan struct{})
	go func() {
		err := req.svc.call(req.mtype, req.argv, req.replyv)
		called <- struct{}{}
		if err != nil {
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			sent <- struct{}{}
			return
		}
		server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
		sent <- struct{}{}
	}()

	if timeout == 0 {
		<-called
		<-sent
		return
	}
	select {
	case <-time.After(timeout):
		req.h.Error = fmt.Sprintf("rpc server: request handle timeout: expect within %s", timeout)
		server.sendResponse(cc, req.h, invalidRequest, sending)
	case <-called:
		<-sent
	}
}
```

解决方法是再通过一个通道来通知协程关闭

``` Go
finish := make(chan struct{})

go func() {
	err := req.svc.call(req.mtype, req.argv, req.replyv)
	select {
	case <-finish:
		close(called)
		close(sent)
		return

// 省略部分代码

select {
case <-time.After(timeout):
	req.h.Error = fmt.Sprintf("rpc server: request handle timeout: expect within %s", timeout)
	server.sendResponse(cc, req.h, invaildRequest, sending)
	finish <- struct{}{}
case <-called:
	<-sent
}

```

# 参考资料

https://geektutu.com/