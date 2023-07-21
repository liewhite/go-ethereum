package arbitrage

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
)

/*
*****************
***** 先做信号端， 通过产生的信号分析是否真的有利润且没有被别人抢走， 然后再做合约， 酌情控制gas
*****************



维护流动性图，图的节点是pair， 有公共token的pair之间存在边。
额外维护所有节点的2个索引， key为token 以及 key 为pair address， 方便通过address和token获取pair
在交易到达时在图中遍历该pair的相邻pair（有最大深度限制）， 直到形成环， 然后在这些环中， 找出有利润空间的路线

如果是汉堡， 先考虑简单逻辑
	买： 同样的路径买卖
	卖： 拿到除了当前pair外最近的环，然后买过来卖出， 最后在买入还回去。

如果考虑复杂逻辑， 则可以结合搬砖汉堡， 汉堡买卖的时候可以找更远的环. 比如买入时可以从其他池子借eth， 然后买入后再还回去， 这样单笔就形成了一笔搬砖
这样整体成本升高了， 但是好处是可以不消耗自有资金， 而且信号范围更大， 能拿到别人看不到的利润

接收到新的交易， 模拟执行拿到swap logs， 然后分别构造交易尝试几个已知的pool（sushi， v2，v3）中对应的pair， 看是否有利可图
如果发现有利润， 则以用户购买量为最大值进行二分查找， 然后以利润最大值的参数发送mev交易
测试： 直接调用doArbitrage， 传入目标交易， mock掉backend和sender， 只让其输出交易内容， 然后本地hardhat广播交易检查利润是否符合预期

## 以下信息需要缓存, 提升性能， 避免每次都要调用合约获取信息
内置factory地址，通过log找到pair， 通过pair获取两个symbol和factory
然后通过俩symbol获取其他几个pair的地址, 再进行利润搜索


## 合约设计
为了通用性， 合约最好的方法就是接受swap路径， 需要的信息包括：
* pair协议
* 每个pair的swap方向
* pair地址
* 每个pair的amountIn, amountOut
32个字节要编码这么多东西还是比较困难的， 因为pair地址就占了20字节, 剩下12字节， 我们只能抠bit了，一共96bit,协议一个bit路径一个bit
还剩94bit，amountIn和amountOut各47
很明显用uint表示不够看，2 ** 47 / 10 ** 18, 一算连decimal都不够
那么就只能用浮点数表示了， 底数决定了精度， 40bit的底数，精度达到了1099511627776， 万亿级别的精度，完全够了
至于指数， 7bit，按照无符号数来算 10 ** (2**7) 也完全超过18的decimal了
比如底数100， 指数18， 就能表示100个ETH
比如底数100， 指数15， 就能表示0.1个ETH

还剩11个字节用来表示买入数量， 所以前面4者可以编码到一个byte32
买入数量编码方式： 浮点数， 即底数 << N, 由于uint256最大256位， N是uint8, 所以N用一个字节代表就够了; 然后剩下10个字节保存底数，size完全够了
10个字节最大 2 ** 80, 这个精度也够了
>>>>> 所以合约的参数就是 byte32[] , 每个元素代表一次swap

合约入口接收到参数， 直接转发到doSwap函数

doSwap函数执行 byte32[] 的第一个swap， 然后通过flash swap把剩余的byte32[]传递给下一个callback
callback 也是直接调用doSwap
*/

// Move 搬砖是闪电贷， 不需要自有资金
type Move struct {
	ctx     context.Context
	backend ethapi.Backend
	evm     *vm.EVM
}

func NewMove(ctx context.Context, backend ethapi.Backend) *Move {
	return &Move{
		ctx:     ctx,
		backend: backend,
	}
}

func (r *Move) Run() {
	txCh := make(chan core.NewTxsEvent)
	sub := r.backend.SubscribeNewTxsEvent(txCh)
	go func() {
		<-r.ctx.Done()
		sub.Unsubscribe()
	}()

	blockCh := make(chan core.ChainHeadEvent)
	blockSub := r.backend.SubscribeChainHeadEvent(blockCh)
	go func() {
		<-r.ctx.Done()
		blockSub.Unsubscribe()
	}()

	go func() {
		for {
			select {
			case txs := <-txCh:
				{
					for _, tx := range txs.Txs {
						fmt.Println(tx)
						err := r.doMove(tx)
						if err != nil {
							log.Error("failed do arbitrage: ", "err", err)
						}
					}
				}
			case block := <-blockCh:
				{
					// todo 重置交易池, 更新evm实例
					fmt.Println(block)
					//r.evm, _ = r.backend.GetEVM(r.ctx)
					//blockNum := block.Block.NumberU64()
					//r.backend.CurrentBlock()
				}
			case err := <-sub.Err():
				{
					log.Error("tx subscription err: " + err.Error())
					panic(err)
				}
			case err := <-blockSub.Err():
				{
					log.Error("block subscription err: " + err.Error())
					panic(err)
				}
			}
		}
	}()
}

func (r *Move) doMove(tx *types.Transaction) error {
	// 搬砖搬多笔交易， 有新的mempool交易到来就累加计算
	// 每笔有利润的交易都要搬， 然后再所有同路径的放一个bundle里再发一次
	panic("not impl")
}
