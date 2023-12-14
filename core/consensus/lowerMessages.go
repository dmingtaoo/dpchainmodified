package consensus

import (
	"dpchain/common"
	"dpchain/crypto"
	"dpchain/logger"
	"dpchain/logger/glog"
	"dpchain/rlp"
	"fmt"
)

type CommonHead struct {
	Consensus []byte        //标注当前共识协议
	RoundID   common.Hash   //当前共识round的ID
	TypeCode  uint8         //阶段类型(tpbft有三个阶段)
	Sender    common.NodeID //发送者的NodeID
}

// Outer wrapper of pre-prepare, prepare, commit messages
// tpbft的三类共识消息(三个阶段各自使用不同的共识消息)
type WrappedLowerConsensusMessage struct {
	Head    CommonHead //消息头(共识消息的common消息头)
	Content []byte     //消息实体(整条共识消息的rlp编码)
}

//打包接口，负责将对应的共识消息(PrePrepareMsg/PrepareMsg/CommitMsg)统一打包成WrappedLowerConsensusMessage
type LowerConsensusMessage interface {
	Wrap() WrappedLowerConsensusMessage
}

//Pre-Prepare阶段使用的共识消息(注意:只有子网组的Leader节点可以发送pre-prepare阶段共识消息)
type PrePrepareMsg struct {
	// common
	head CommonHead //消息头

	// message
	Version common.Hash   //当前区块哈希
	Nonce   uint8         //暂不了解。。。
	TxOrder []common.Hash //当前节点所记录的区块内交易的顺序
}

func (ppm *PrePrepareMsg) Wrap() WrappedLowerConsensusMessage {
	serialized, err := rlp.EncodeToBytes(ppm)
	if err != nil {
		glog.V(logger.Error).Infoln("fail in PrePrepareMsg wrap")
	}
	wlcm := WrappedLowerConsensusMessage{
		Head:    ppm.head,
		Content: serialized,
	}
	return wlcm
}

type PrepareMsg struct {
	// common
	head CommonHead

	// message
	ValidOrder []byte // vote the valid orders, true is 1, false is 0  一个交易对应 1bit, 注意在prepare阶段时的交易列表就是有序的了
	Version    common.Hash
}

//将 PrepareMsg 打包成 WrappedLowerConsensusMessage统一格式消息
func (pm *PrepareMsg) Wrap() WrappedLowerConsensusMessage {
	serialized, err := rlp.EncodeToBytes(pm)
	if err != nil {
		glog.V(logger.Error).Infoln("fail in PrepareMsg wrap")
	}
	wlcm := WrappedLowerConsensusMessage{
		Head:    pm.head,
		Content: serialized,
	}
	return wlcm
}

type CommitMsg struct {
	// common
	head CommonHead

	// message
	Result []byte // the result after tidy  完成整理的待发送区块的区块ID

	// vote
	Signature []byte // to support the result  发送此commit共识消息的节点的数字签名
}

//将 CommitMsg 打包成 WrappedLowerConsensusMessage统一格式消息
func (cm *CommitMsg) Wrap() WrappedLowerConsensusMessage {
	serialized, err := rlp.EncodeToBytes(cm)
	if err != nil {
		glog.V(logger.Error).Infoln("fail in CommitMsg wrap")
	}
	wlcm := WrappedLowerConsensusMessage{
		Head:    cm.head,
		Content: serialized,
	}
	return wlcm
}

//计算PrePrepare消息的哈希值
func (ppm *PrePrepareMsg) Hash() common.Hash {
	temp := *ppm
	serialized, err := rlp.EncodeToBytes(&temp)
	if err != nil {
		glog.V(logger.Error).Infoln("fail in encode prepreparemsg")
	}
	return crypto.Sha3Hash(serialized)
}

// PrePrepare消息的哈希值作为本次交易round的ID值
// compute the round ID and set it
func (ppm *PrePrepareMsg) ComputeRoundID() common.Hash {
	if (ppm.head.RoundID == common.Hash{}) {
		ppm.head.RoundID = ppm.Hash()
	}
	return ppm.head.RoundID
}

//将统一格式的WrappedLowerConsensusMessage消息解包成对应共识阶段类型的消息
//(Pre-Prepare阶段/Prepare阶段/Commit阶段)分别解包成(PrePrepareMsg/PrepareMsg/CommitMsg)消息
func (wlcm *WrappedLowerConsensusMessage) UnWrap() (LowerConsensusMessage, error) {
	switch wlcm.Head.TypeCode {
	case StatePrePrepare:
		ppm := new(PrePrepareMsg)
		if err := rlp.DecodeBytes(wlcm.Content, ppm); err != nil {
			return nil, fmt.Errorf("fail in WrappedLowerConsensusMessage unwrap: %v", err)
		}
		ppm.head = wlcm.Head
		return ppm, nil
	case StatePrepare:
		pm := new(PrepareMsg)
		if err := rlp.DecodeBytes(wlcm.Content, pm); err != nil {
			return nil, fmt.Errorf("fail in WrappedLowerConsensusMessage unwrap: %v", err)
		}
		pm.head = wlcm.Head
		return pm, nil
	case StateCommit:
		cm := new(CommitMsg)
		if err := rlp.DecodeBytes(wlcm.Content, cm); err != nil {
			return nil, fmt.Errorf("fail in WrappedLowerConsensusMessage unwrap: %v", err)
		}
		cm.head = wlcm.Head
		return cm, nil
	default:
		return nil, fmt.Errorf("fail in WrappedLowerConsensusMessage unwrap: unknown code: %v", wlcm.Head.TypeCode)
	}
}

// Validate the integrity of a pre-prepare message by compare the
// round ID and the in-time computed hash.
// 验证PrePrepareMsg消息的正确性
func (ppm *PrePrepareMsg) ValidateIntegrity() bool {
	return ppm.Hash() == ppm.head.RoundID
}
