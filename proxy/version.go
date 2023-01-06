package proxy

import (
	tmabci "github.com/tendermint/tendermint/abci/types"

	"github.com/line/ostracon/version"
)

// RequestInfo contains all the information for sending
// the abci.RequestInfo message during handshake with the app.
// It contains only compile-time version information.
var RequestInfo = tmabci.RequestInfo{
	Version:      version.OCCoreSemVer,
	BlockVersion: version.BlockProtocol,
	P2PVersion:   version.P2PProtocol,
}
