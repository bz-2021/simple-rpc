/*
 * @Author: bz2021
 * @Date: 2023-12-20 11:09:35
 * @Description:
 */
package xclient

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type RpcRegistryDiscovery struct {
	*MultiServerDiscovery // 复用其中的功能
	registry string // 注册中心的地址
	timeout time.Duration // 服务列表的过期时间
	lastUpdate time.Time // 从注册中心更新服务列表的时间，默认 10s 过期，10s 后需要从注册中心更新新的列表
}

const defaultUpdateTimeout = time.Second * 10

func NewRpcRegistryDiscovery(registerAddr string, timeout time.Duration) *RpcRegistryDiscovery {
	if timeout == 0 {
		timeout = defaultUpdateTimeout
	}
	d := &RpcRegistryDiscovery{
		MultiServerDiscovery: NewMultiServerDiscovery(make([]string, 0)),
		registry: registerAddr,
		timeout: timeout,
	}
	return d
}

func (d *RpcRegistryDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	d.lastUpdate = time.Now()
	return nil
}

func (d *RpcRegistryDiscovery) Refresh() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.lastUpdate.Add(d.timeout).After(time.Now()) {
		return nil
	}
	log.Println("rpc registry: refresh servers from registry", d.registry)
	resp, err := http.Get(d.registry)
	if err != nil { 
		log.Println("rpc registry refresh err:", err)
		return err
	}
	servers := strings.Split(resp.Header.Get("X-RPC-Servers"), ",")
	d.servers = make([]string, 0, len(servers))
	for _, server := range servers {
		if strings.TrimSpace(server) != "" {
			d.servers = append(d.servers, strings.TrimSpace(server))
		}
	}
	d.lastUpdate = time.Now()
	return nil
}

func (d *RpcRegistryDiscovery) Get(mode SelectMode) (string, error) {
	if err := d.Refresh(); err != nil {
		return "", err
	}
	return d.MultiServerDiscovery.Get(mode)
}

func (d *RpcRegistryDiscovery) GetAll() ([]string, error) {
	if err := d.Refresh(); err != nil {
		return nil, err
	}
	return d.MultiServerDiscovery.GetAll()
}