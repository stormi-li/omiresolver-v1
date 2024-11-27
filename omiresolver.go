package omiresolver

import (
	"fmt"
	"math/rand/v2"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stormi-li/omiserd-v1"
)

type Router struct {
	Discover        *omiserd.Discover
	addressMap      map[string]map[string]map[string]string
	addressPool     map[string][]string
	mutex           sync.RWMutex
	RefreshInterval time.Duration
}

func NewRouter(opts *redis.Options, nodeType omiserd.NodeType) *Router {
	router := &Router{
		Discover:        omiserd.NewClient(opts, nodeType).NewDiscover(),
		addressMap:      map[string]map[string]map[string]string{},
		addressPool:     map[string][]string{},
		mutex:           sync.RWMutex{},
		RefreshInterval: 10 * time.Second,
	}
	router.Update()
	go router.Refresh()
	return router
}

func (router *Router) Update() {
	addrs := router.Discover.GetAll()
	addrPool := map[string][]string{}
	for name, addrs := range addrs {
		for _, addr := range addrs {
			data := router.Discover.GetData(name, addr)
			weight, _ := strconv.Atoi(data["weight"])
			for i := 0; i < weight; i++ {
				addrPool[name] = append(addrPool[name], addr)
			}
		}
	}
	addrMap := map[string]map[string]map[string]string{}
	for name, addrs := range addrs {
		if addrMap[name] == nil {
			addrMap[name] = map[string]map[string]string{}
		}
		for _, addr := range addrs {
			data := router.Discover.GetData(name, addr)
			addrMap[name][addr] = data
		}
	}
	router.mutex.Lock()
	router.addressMap = addrMap
	router.addressPool = addrPool
	router.mutex.Unlock()
}

func (router *Router) Refresh() {
	for {
		router.Update()
		time.Sleep(router.RefreshInterval)
	}
}

func (router *Router) GetAddress(serverName string) string {
	router.mutex.RLock()
	defer router.mutex.RUnlock()
	if len(router.addressMap[serverName]) == 0 {
		return ""
	}
	return router.addressPool[serverName][rand.IntN(len(router.addressMap[serverName]))]
}

func (router *Router) Has(serverName string) bool {
	router.mutex.RLock()
	defer router.mutex.RUnlock()
	return len(router.addressMap[serverName]) != 0
}

// 定义接口用于服务地址解析
type Resolver struct {
	router   *Router
	nodeType omiserd.NodeType
}

func NewResolver(opts *redis.Options, nodeType omiserd.NodeType) *Resolver {
	return &Resolver{
		router:   NewRouter(opts, nodeType),
		nodeType: nodeType,
	}
}

func (resolver *Resolver) Resolve(url url.URL) (*url.URL, error) {
	serverName := strings.Split(url.Path, "/")[1]
	domainName := url.Host
	if resolver.router.Has(serverName) {
		url.Path = strings.TrimPrefix(url.Path, "/"+serverName)
		url.Host = resolver.router.GetAddress(serverName)
		url.Scheme = resolver.router.addressMap[serverName][url.Host]["protocal"]
	} else if resolver.router.Has(domainName) {
		url.Host = resolver.router.GetAddress(domainName)
		url.Scheme = resolver.router.addressMap[domainName][url.Host]["protocal"]
	} else {
		return nil, fmt.Errorf("解析失败: %s", url.String())
	}
	return &url, nil
}
