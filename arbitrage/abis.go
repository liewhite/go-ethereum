package arbitrage

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"strings"
)

var (
	// todo 可以codegen， 更方便一点
	ABIV2, _ = abi.JSON(strings.NewReader(``))
	ABIV3, _ = abi.JSON(strings.NewReader(``))
)
