package core

import (
	"frbg/codec"
	"sync"
)

var rpcMgr = NewRpcMgr()

type RpcMgr struct {
	// RPC响应管理
	rpcResponses map[uint16]chan *codec.Message
	rpcCallbacks map[uint16]RpcCallback
	rpcMu        sync.RWMutex
}

func NewRpcMgr() *RpcMgr {
	return &RpcMgr{
		rpcResponses: make(map[uint16]chan *codec.Message),
		rpcCallbacks: make(map[uint16]RpcCallback),
	}
}

// RegisterRpc 注册RPC响应等待
func (p *RpcMgr) RegisterRpc(seq uint16, respChan chan *codec.Message) {
	p.rpcMu.Lock()
	defer p.rpcMu.Unlock()
	p.rpcResponses[seq] = respChan
}

// UnregisterRpc 取消注册RPC响应等待
func (p *RpcMgr) UnregisterRpc(seq uint16) {
	p.rpcMu.Lock()
	defer p.rpcMu.Unlock()
	delete(p.rpcResponses, seq)
}

// RegisterRpcCallback 注册RPC异步回调
func (p *RpcMgr) RegisterRpcCallback(seq uint16, callback RpcCallback) {
	p.rpcMu.Lock()
	defer p.rpcMu.Unlock()
	p.rpcCallbacks[seq] = callback
}

func (p *RpcMgr) UnregisterRpcCallback(seq uint16) {
	p.rpcMu.Lock()
	defer p.rpcMu.Unlock()
	delete(p.rpcCallbacks, seq)
}

// HandleRpcResponse 处理RPC响应
func (p *RpcMgr) HandleRpcResponse(msg *codec.Message) bool {
	seq := msg.Seq

	// fmt.Println("HandleRpcResponse seq:", seq, p.rpcResponses)

	p.rpcMu.RLock()
	// 优先处理同步等待
	if respChan, ok := p.rpcResponses[seq]; ok {
		p.rpcMu.RUnlock()
		respChan <- msg
		// select {
		// case respChan <- msg:
		// default:
		// 通道已关闭或满，丢弃
		// }
		return true
	}

	// 处理异步回调
	if callback, ok := p.rpcCallbacks[seq]; ok {
		p.rpcMu.RUnlock()
		go callback(msg, nil)
		return true
	}
	p.rpcMu.RUnlock()
	return false
}
