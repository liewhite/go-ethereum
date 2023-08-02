package arbitrage

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type SwapProtocol string

var (
	SwapProtocolV2 SwapProtocol = "V2"
	SwapProtocolV3 SwapProtocol = "V3"
)

func (s SwapProtocol) Factory() common.Address {
	if s == SwapProtocolV2 {
		return common.HexToAddress("0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f")
	} else if s == SwapProtocolV3 {
		return common.HexToAddress("0x1F98431c8aD98523631AE4a59f267346ea31F984")
	} else {
		log.Crit("unknown protocol", "protocol", s)
		return [20]byte{}
	}
}
func (s SwapProtocol) ABI() *abi.ABI {
	if s == SwapProtocolV2 {
		return &ABIV2
	} else if s == SwapProtocolV3 {
		return &ABIV3
	} else {
		log.Crit("unknown protocol", "protocol", s)
		return nil
	}
}
