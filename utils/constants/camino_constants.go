package constants

import (
	"math/big"
)

// CaminoLocalChainID Used in the local network. Necessary to set the chainID to a value other than the default,
// otherwise the local network will give deployment permissions to every address no matter its role value.
var CaminoLocalChainID = big.NewInt(503)

const APIAdminKey = "runner"
