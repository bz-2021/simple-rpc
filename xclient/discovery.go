/*
 * @Author: bz2021
 * @Date: 2023-12-19 15:46:42
 * @Description:
 */
package xclient

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"time"
)



type SelectMode int

const (
	RandomSelect SelectMode = iota
	RoundRobinSelect
)

type Discovery interface {
	Refresh() error
	Update(servers []string) error
	Get(mode SelectMode) (string, error)
	GetAll() ([]string, error)
}

// MultiServerDiscovery 是一个不需要注册中心的服务发现中心
// 需要用户显式提供服务器地址
type MultiServerDiscovery struct {
	r *rand.Rand // 生成随机数字
	mu sync.RWMutex
	servers []string 
	index int // 记录 Robin 算法选择的位置
}

func NewMultiServerDiscovery(servers []string) *MultiServerDiscovery {
	d := &MultiServerDiscovery{
		servers: servers,
		r: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	d.index = d.r.Intn(math.MaxInt32 - 1)
	return d
}

var _ Discovery = (*MultiServerDiscovery)(nil)

func (d *MultiServerDiscovery) Refresh() error {
	return nil
}

// Update 对服务发现进行动态更新
func (d *MultiServerDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	return nil
}

func (d *MultiServerDiscovery) Get(mode SelectMode) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := len(d.servers)
	if n == 0 {
		return "", errors.New("rpc discovery: no available servers")
	}
	switch mode {
	case RandomSelect:
		return d.servers[d.r.Intn(n)], nil
	case RoundRobinSelect:
		s := d.servers[d.index % n]
		d.index = (d.index + 1) % 1
		return s, nil
	default:
		return "", errors.New("rpc discovery: not supported select mode")
	}
}

func (d *MultiServerDiscovery) GetAll() ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	// 返回 d.servers 的拷贝
	servers := make([]string, len(d.servers), len(d.servers))
	copy(servers, d.servers)
	return servers, nil
}

// func (xc *XClient) Broadcast(ctx context.Context, serviceMethod string, args, reply interface{}) error {
// 	servers, err := xc.d.GetAll()
// 	if err != nil {
// 		return err
// 	}
// 	var wg sync.WaitGroup
// 	var mu sync.Mutex // 保护 e 和 replyDone
// 	var e error
// 	replyDone := reply == nil
// 	ctx, cancel := context.WithCancel(ctx)
// 	for _, rpcAddr := range servers {
// 		wg.Add(1)
// 		go func(rpcAddr string) {
// 			defer wg.Done()
// 			var clonedReply interface{}
// 			if reply != nil {
// 				clonedReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
// 			}
// 			err := xc.call(rpcAddr, ctx, serviceMethod, args, clonedReply)
// 			mu.Lock()
// 			if err != nil && e == nil {
// 				e = err
// 				cancel() // 某个 Call 失败了，将其 cancel 掉
// 			}
// 			if err == nil && !replyDone {
// 				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(clonedReply).Elem())
// 				replyDone = true
// 			}
// 			mu.Unlock()
// 		}(rpcAddr)
// 	}
// 	wg.Wait()
// 	return e
// }

func (xc *XClient) Broadcast(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	servers, err := xc.d.GetAll()
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var mu sync.Mutex // 保护 e 和 replyDone
	var e error
	replyDone := reply == nil
	var cancel context.CancelFunc // 声明 cancel 变量
	ctx, cancel = context.WithCancel(ctx) // 为 ctx 分配 cancel 函数
	defer cancel() // 确保 cancel 函数被调用以避免上下文泄漏
	for _, rpcAddr := range servers {
		wg.Add(1)
		go func(rpcAddr string) {
			defer wg.Done()
			var clonedReply interface{}
			if reply != nil {
				clonedReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
			}
			err := xc.call(rpcAddr, ctx, serviceMethod, args, clonedReply)
			mu.Lock()
			if err != nil && e == nil {
				e = err
				cancel() // 某个 Call 失败了，将其 cancel 掉
			}
			if err == nil && !replyDone {
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(clonedReply).Elem())
				replyDone = true
			}
			mu.Unlock()
		}(rpcAddr)
	}
	wg.Wait()
	return e
}