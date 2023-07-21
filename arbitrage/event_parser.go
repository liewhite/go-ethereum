package arbitrage

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"strings"
)

var (
	ZERO = big.NewInt(0)
)

type UniV2SwapEvent struct {
	Sender     string   `json:"sender"`
	Amount0In  *big.Int `json:"amount0In"`
	Amount1In  *big.Int `json:"amount1In"`
	Amount0Out *big.Int `json:"amount0Out"`
	Amount1Out *big.Int `json:"amount1Out"`
	To         string   `json:"to"`
	LogIndex   string   `json:"logIndex"`
}

// Price amount0 / amount1
func (v *UniV2SwapEvent) Price() *big.Int {
	if !isZero(v.Amount0In) {
		return new(big.Int).Quo(v.Amount0In, v.Amount1Out)
	} else {
		return new(big.Int).Quo(v.Amount0Out, v.Amount1In)
	}
}

type UniV3SwapEvent struct {
	Sender    string   `json:"sender"`
	Amount0   *big.Int `json:"amount0In"`
	Amount1   *big.Int `json:"amount1In"`
	Recipient string   `json:"to"`
	LogIndex  string   `json:"logIndex"`
}

func (v *UniV3SwapEvent) Price() *big.Int {
	return new(big.Int).Abs(new(big.Int).Quo(v.Amount0, v.Amount1))
}

func parseUniv2SwapEvent(log *types.Log) (*UniV2SwapEvent, error) {
	event := log
	data := event.Data
	if len(event.Topics) != 3 {
		return nil, fmt.Errorf("univ2 swap topic not match")
	}

	a0i := big.NewInt(0).SetBytes(data[:32])
	a1i := big.NewInt(0).SetBytes(data[32:64])
	a0o := big.NewInt(0).SetBytes(data[64:96])
	a1o := big.NewInt(0).SetBytes(data[96:128])

	parsed := &UniV2SwapEvent{
		Sender:     hash2Addr(event.Topics[1]),
		Amount0In:  a0i,
		Amount1In:  a1i,
		Amount0Out: a0o,
		Amount1Out: a1o,
		To:         hash2Addr(event.Topics[2]),
	}
	if isZero(parsed.Amount0In) && isZero(parsed.Amount0Out) && isZero(parsed.Amount1In) && isZero(parsed.Amount1In) {
		return nil, fmt.Errorf("swap amount is 0: %s", log.TxHash)
	}
	return parsed, nil
}

func parseUniv3SwapEvent(log *types.Log) (*UniV3SwapEvent, error) {
	event := log
	data := event.Data
	if len(event.Topics) != 3 {
		return nil, fmt.Errorf("univ3 swap topic not match")
	}
	int256, _ := abi.NewType("int256", "", nil)
	a0, _ := abi.ReadInteger(int256, data[0:32])

	amount0, ok := a0.(*big.Int)
	if !ok {
		return nil, fmt.Errorf("univ3 parse failed %s", log.TxHash.Hex())
	}

	a1, _ := abi.ReadInteger(int256, data[32:32*2])

	amount1, ok := a1.(*big.Int)
	if !ok {
		return nil, fmt.Errorf("univ3 parse failed %s", log.TxHash.Hex())
	}

	parsed := &UniV3SwapEvent{
		Amount0: amount0,
		Amount1: amount1,
	}
	if isZero(parsed.Amount0) && isZero(parsed.Amount1) {
		return nil, fmt.Errorf("swap amoun is 0: %s", log.TxHash)
	}
	return parsed, nil
}

func isZero(i *big.Int) bool {
	return i.Cmp(ZERO) == 0
}
func hash2Addr(hs common.Hash) string {
	return strings.ToLower(common.BytesToAddress(hs[12:]).Hex())

}
