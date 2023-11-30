package validator

import (
	"dpchain/common"
	"dpchain/core/eles"
	"dpchain/crypto"
	"fmt"
)

const (
	defaultLowerValidRatio = 0.5
	defaultUpperValidRatio = 0.5
)

// CommonValidator is used to validate the block
type CommonValidator struct {
	SubnetPool           map[string]SubnetVoters
	LeaderVoters         map[common.Address]bool
	LowerValidThresholds map[string]int
	UpperValidThreshold  int
}

func (cv *CommonValidator) SetDefaultThreshold() {
	cv.LowerValidThresholds = make(map[string]int)
	for netID, voters := range cv.SubnetPool {
		threshold := int(float64(len(voters))*defaultLowerValidRatio) + 1
		cv.LowerValidThresholds[netID] = threshold
	}
	cv.UpperValidThreshold = int(float64(len(cv.LeaderVoters))*defaultUpperValidRatio) + 1
}

func (cv *CommonValidator) SetLowerValidThreshold_UsePBFTFactor(factor int) error {
	if factor <= 0 {
		return fmt.Errorf("factor is zero or less")
	}
	cv.LowerValidThresholds = make(map[string]int)
	for netID, voters := range cv.SubnetPool {
		threshold := ((len(voters) - 1) / factor) * (factor - 1)
		cv.LowerValidThresholds[netID] = threshold
	}
	return nil
}

func (cv *CommonValidator) SetLowerValidThreshold_UseRatio(ratio float64) error {
	if ratio < 0 || ratio > 1 {
		return fmt.Errorf("ratio is not between 0 and 1")
	}
	cv.LowerValidThresholds = make(map[string]int)
	for netID, voters := range cv.SubnetPool {
		threshold := int(float64(len(voters)) * ratio)
		cv.LowerValidThresholds[netID] = threshold
	}
	return nil
}

func (cv *CommonValidator) SetUpperValidThreshold_UsePBFTFactor(factor int) error {
	if factor <= 0 {
		return fmt.Errorf("factor is zero or less")
	}

	threshold := ((len(cv.LeaderVoters) - 1) / factor) * (factor - 1)
	cv.UpperValidThreshold = threshold
	return nil
}

func (cv *CommonValidator) SetUpperValidThreshold_UseRatio(ratio float64) error {
	if ratio < 0 || ratio > 1 {
		return fmt.Errorf("ratio is not between 0 and 1")
	}

	threshold := int(float64(len(cv.LeaderVoters)) * ratio)
	cv.UpperValidThreshold = threshold
	return nil
}

type SubnetVoters map[common.Address]bool

func (cv *CommonValidator) LowerValidate(block *eles.Block) bool {
	if cv.LowerValidThresholds == nil {
		return false
	}

	subnetID := string(block.Subnet)
	voters, ok := cv.SubnetPool[subnetID] //1.必须有子网内的节点签名
	if !ok {
		return false
	}

	blockHash, err := block.Hash()
	if err != nil {
		return false
	}
	if blockHash != block.BlockID { //2.区块ID必须正确
		return false
	}

	txIDPool := make(map[common.Hash]bool)
	for _, tx := range block.Transactions {
		txHash := tx.Hash()
		if txHash != tx.TxID { //每笔交易的哈希值必须正确
			return false
		}
		if _, ok := txIDPool[txHash]; ok { // repeat transactions
			return false
		}
		// check whether the transaction is unique (in the past 50 blocks)
		txIDPool[txHash] = true

		valid, err := crypto.SignatureValid(tx.Sender, tx.Signature, tx.TxID)
		if err != nil {
			return false
		}
		if !valid {
			return false
		}
	}

	threshold, ok := cv.LowerValidThresholds[subnetID]
	if !ok {
		return false
	}

	count := 0
	votePool := make(map[common.Address]bool)
	for _, sig := range block.SubnetVotes {
		addr, err := crypto.Signature2Addr(sig, blockHash)
		if err != nil {
			return false
		}
		if _, ok := votePool[addr]; ok { // repeat vote
			return false
		}
		votePool[addr] = true

		if _, ok := voters[addr]; ok {
			count += 1
		}
	}
	if count >= threshold {
		return true
	} else {
		return false
	}
}

func (cv *CommonValidator) UpperValidate(block *eles.Block) bool {
	if !cv.LowerValidate(block) {
		return false
	}

	if len(block.Transactions) != len(block.Receipt.TxReceipt) {
		return false
	}

	blockReceiptID, err := block.ComputeReceiptID()
	if err != nil {
		return false
	}
	if blockReceiptID != block.Receipt.ReceiptID {
		return false
	}

	var threshold int
	threshold = cv.UpperValidThreshold

	count := 0
	votePool := make(map[common.Address]bool)
	for _, sig := range block.LeaderVotes {

		addr, err := crypto.Signature2Addr(sig, blockReceiptID)

		if err != nil {
			return false
		}
		if _, ok := votePool[addr]; ok { // repeat vote
			return false
		}
		votePool[addr] = true

		if _, ok := cv.LeaderVoters[addr]; ok {
			count += 1
		}
	}
	if count >= threshold {
		return true
	} else {
		return false
	}
}
