package arbitrage

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	log2 "github.com/ethereum/go-ethereum/log"
	"math/big"
)

/*
交易到来时才往图中添加节点， 并将其他交易所的pair也添加进来(通过getPair或者getPool方法，如果存在的话)
目前ethereum上只考虑 uniSwap v2/3 sushi, pancake

维护流动性图，图的节点是pair， 有公共token的pair之间存在边。
额外维护所有节点的2个索引， key为token 以及 key 为pair address， 方便通过address和token获取pair
在交易到达时在图中遍历该pair的相邻pair（有最大深度限制）， 直到形成环， 然后在这些环中， 找出有利润空间的路线

无向图， 在搜索的时候会按路径搜索, 比如 AB 节点， 进入的时候是A， 就只找公共token是B的
>>>>>>>>>>>>>>>>>>>
如何初始化图， 以及如何提前计算
>>>>>>>>>>>>>>
估算是否可以搬砖： 价格存在差异， 比如 > 0.1 %, 价格以最后一次swap的price为准
通过估算， 可以过滤掉部分无效信号
*/
var (
	v2SwapTopic0 = "0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822"
	v3SwapTopic0 = "0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67"
)

type PairGraphNode struct {
	Addr      common.Address
	Token0    common.Address
	Token1    common.Address
	Price     *big.Int
	Neighbors map[common.Address][]*PairGraphNode // 有token相交的都是邻居, key为token0或者token1
}

func (p *PairGraphNode) HasToken(tokenA common.Address) bool {
	if tokenA == p.Token0 {
		return true
	} else if tokenA == p.Token1 {
		return true
	} else {
		return false
	}
}
func (p *PairGraphNode) AnotherToken(tokenA common.Address) common.Address {
	if tokenA == p.Token0 {
		return p.Token1
	} else if tokenA == p.Token1 {
		return p.Token0
	} else {
		log2.Crit("token not in pair", "tokenA", tokenA, "pair", p.Addr)
		return common.Address{}
	}
}

// SearchCycles FindCycle 根据tokenIn，找到所有含有tokenOut的邻居， 并递归调用邻居
// 如果endToken == tokenOut, 且depth == 0, 则结束递归,
func (p *PairGraphNode) SearchCycles(path []*PairGraphNode, tokenIn, endToken common.Address, depth int) [][]*PairGraphNode {
	path = append(path, p)
	// 遍历邻居， 不包括自己， 因为开始节点不能包括自己， 要统一处理
	neighborsWithTokenOut := p.Neighbors[p.AnotherToken(tokenIn)]
	var result [][]*PairGraphNode
	for _, node := range neighborsWithTokenOut {
		if node.HasToken(endToken) {
			result = append(result, append(path, node))
		} else {
			if depth > 0 {
				childResult := node.SearchCycles(path, p.AnotherToken(tokenIn), endToken, depth-1)
				for _, nodes := range childResult {
					result = append(result, nodes)
				}
			}
		}
	}
	return result
}

type PairGraph struct {
	ctx        context.Context
	cli        *ethclient.Client
	backend    ethapi.Backend
	pairByAddr map[common.Address]*PairGraphNode // 按合约地址组织图
}

// SearchCycle SearchPath 搜索跟从addr出发， 可以再次回到addr的路径, 可能存在多个
func (p *PairGraph) SearchCycles(protocol SwapProtocol, addr common.Address, tokenIn common.Address, maxDepth int) ([][]*PairGraphNode, error) {
	if _, ok := p.pairByAddr[addr]; !ok {
		err := p.CreatePair(protocol, addr)
		if err != nil {
			return nil, err
		}
	}
	pair := p.pairByAddr[addr]
	if !pair.HasToken(tokenIn) {
		return nil, fmt.Errorf("token %s not in pair %s", tokenIn, pair.Addr)
	}
	return pair.SearchCycles(nil, tokenIn, pair.AnotherToken(tokenIn), maxDepth-1), nil
}

//func (p *PairGraph) Start() {
//	ch := make(chan []*types.Log)
//	p.backend.SubscribeLogsEvent(ch)
//	go func() {
//		for logs := range ch {
//			for _, log := range logs {
//				if len(log.Topics) > 0 {
//					if log.Topics[0].Hex() == v2SwapTopic0 {
//						p.processV2Log(log)
//					} else if log.Topics[0].Hex() == v3SwapTopic0 {
//						p.processV3Log(log)
//					}
//				}
//			}
//		}
//	}()
//}

// CreatePair 从链上读取pair信息， 以及其他交易所的该pair信息， 并写入图中
func (p *PairGraph) CreatePair(protocol SwapProtocol, addr common.Address) error {
	factory := protocol.Factory()
	// 读取pair信息， 包括factory， token0， token1
	p.cli.CallContract(p.ctx, ethereum.CallMsg{
		From: addr,
		To:   addr,
		Data: addr,
	}, nil)
	// 创建pair并保存到graph
	// 根据token0， token1， 获取其他几个交易所的pair， 也保存到graph
	return nil
}

//func (p *PairGraph) processV2Log(log *types.Log) {
//	parsed, err := parseUniv2SwapEvent(log)
//	if err != nil {
//		log2.Warn("bad univ2 swap", "log", log.TxHash.Hex(), "err", err)
//		return
//	}
//}
//
//func (p *PairGraph) processV3Log(log *types.Log) {
//	parsed, err := parseUniv3SwapEvent(log)
//	if err != nil {
//		log2.Warn("bad univ3 swap", "log", log.TxHash.Hex(), "err", err)
//		return
//	}
//}
