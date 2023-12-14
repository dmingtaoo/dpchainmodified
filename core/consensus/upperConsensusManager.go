package consensus

import (
	"dpchain/core/separator"
)

type UpperConsensusManager interface {
	Install(cp *ConsensusPromoter)
	ProcessMessage(message *separator.Message)
	Start() error
	Stop()
}
