package blockOrdering

import (
	"crypto/ecdsa"
	"dpchain/coefficient"
	"dpchain/common"
	"dpchain/core/consensus"
	"dpchain/core/contract"
	"dpchain/core/dpnet"
	"dpchain/core/eles"
	"dpchain/core/separator"
	"dpchain/core/validator"
	"dpchain/core/worldstate"
	"dpchain/crypto"
	loglogrus "dpchain/log_logrus"
	"dpchain/rlp"
	"fmt"
	"sync"
	"time"
)

var block_expiration = coefficient.BlockOrder_BlockExpireDuration

var blockUploadCycle = coefficient.BlockOrder_BlockUploadScanCycle

const (
	state_finish uint8 = 0x00
	state_signed uint8 = 0x01
	state_wait   uint8 = 0x02 // have known the block server is malicious, wait for recovery
)

type BlockOrderAgent struct {
	msgChannel consensus.UpperChannel

	selfNode *dpnet.Node
	prvKey   *ecdsa.PrivateKey

	serverInTerm common.NodeID
	sc           *stateContainer

	stateManager    worldstate.StateManager
	blockCache      eles.BlockCache
	validateManager *validator.ValidateManager
	contractEngine  contract.ContractEngine
	syncProtocal    SyncProto

	rc *roundContainer

	pc *protectContainer

	quit chan struct{}
}

func NewBlockOrderAgent(orderServer common.NodeID, syn SyncProto) *BlockOrderAgent {
	sc := &stateContainer{
		changeMux:      sync.RWMutex{},
		currentVersion: common.Hash{},
		currentHeight:  0,
	}

	rc := &roundContainer{
		changeMux: sync.RWMutex{},
		state:     state_finish,
		version:   common.Hash{},
		height:    0,
		stuff:     nil,
		sigMsg:    nil,
		donMsg:    nil,
	}

	pc := &protectContainer{
		changeMux: sync.RWMutex{},
		isActive:  false,
	}

	boa := &BlockOrderAgent{
		serverInTerm: orderServer,
		sc:           sc,
		rc:           rc,
		pc:           pc,
		syncProtocal: syn,
		quit:         make(chan struct{}),
	}
	return boa
}

func (b *BlockOrderAgent) Install(cp *consensus.ConsensusPromoter) {
	cp.SetUpperConsensusManager(b)
	b.msgChannel = cp.BackUpperChannel()
	b.selfNode = cp.SelfNode
	b.prvKey = cp.BackPrvKey()
	b.stateManager = cp.StateManager
	b.blockCache = cp.BackBlockCache()
	b.validateManager = cp.BackValidateManager()
	b.contractEngine = cp.BackContractEngine()

	b.flushState()

}

type stateContainer struct {
	changeMux sync.RWMutex

	currentVersion common.Hash
	currentHeight  uint64
}

type roundContainer struct {
	changeMux sync.RWMutex

	state   uint8
	version common.Hash // the previous block hash
	height  uint64      // the previous block height

	stuff  *eles.Block
	sigMsg *SignMessage
	donMsg *DoneMessage
}

type protectContainer struct {
	changeMux sync.RWMutex
	isActive  bool
}

func (pc *protectContainer) isProtect() bool {
	pc.changeMux.RLock()
	defer pc.changeMux.RUnlock()
	return pc.isActive
}

func (pc *protectContainer) setVale(val bool) {
	pc.changeMux.Lock()
	defer pc.changeMux.Unlock()
	pc.isActive = val
}

func (sc *stateContainer) getState() (version common.Hash, height uint64) {
	sc.changeMux.RLock()
	defer sc.changeMux.RUnlock()
	return sc.currentVersion, sc.currentHeight
}

func (sc *stateContainer) updateState(version common.Hash, height uint64) {
	sc.changeMux.Lock()
	defer sc.changeMux.Unlock()
	sc.currentVersion = version
	sc.currentHeight = height
}

func (b *BlockOrderAgent) flushState() (currentVersion common.Hash, currentHeight uint64) {
	version, height := b.stateManager.GetCurrentState()
	b.sc.updateState(version, height)
	return version, height
}

func (b *BlockOrderAgent) ProcessMessage(message *separator.Message) {
	if !message.IsUpperConsensus() {
		return
	}
	bom, err := DeserializeBlockOrderMsg(message.PayLoad)
	if err != nil {
		loglogrus.Log.Warnf("[Block Ordering] Upper Consensus failed: Couldn't Deserialize separator.Message into Block Order Msg, err:%v\n", err)
		return
	}
	if message.From != bom.Sender {
		loglogrus.Log.Warnf("[Block Ordering] Upper Consensus failed: Deserialized message -- message.From (%x) could not match the sender (%x)\n", message.From, bom.Sender)
		return
	}

	msg, err := RetrievePayload(bom)
	if err != nil {
		loglogrus.Log.Warnf("[Block Ordering] fail in ProcessMessage from %x: %v\n", bom.Sender, err)
		return
	}

	switch bom.TypeCode {
	case CODE_NEXT:
		nexMsg, ok := msg.(*NextMessage)
		if !ok {
			loglogrus.Log.Errorf("[Block Ordering] fail in type assert from %x\n", bom.Sender)
			return
		}
		b.handleNEXT(nexMsg)

	case CODE_CMIT:
		cmiMsg, ok := msg.(*CommitMessage)
		if !ok {
			loglogrus.Log.Errorf("[Block Ordering] fail in type assert from %x\n", bom.Sender)
			return
		}
		loglogrus.Log.Infof("[Block Ordering] Upper Channel收到来自节点(%x)的 CODE_CMIT Msg, Msg.height:%d  local.Height:%d \n", message.From, cmiMsg.Height, b.stateManager.GetBlockChainHeight())
		b.handleCMIT(cmiMsg)
		loglogrus.Log.Infof("[Block Ordering] Upper Channel完成对来自节点(%x)的 CODE_CMIT Msg的处理\n", message.From)

	case CODE_RCVY:
		loglogrus.Log.Infof("[Block Ordering] Upper Channel收到来自节点(%x)的 CODE_RCVY Msg\n", message.From)
		rcvMsg, ok := msg.(*RecoveryMessage)
		if !ok {
			loglogrus.Log.Errorf("[Block Ordering] fail in type assert from %x\n", bom.Sender)
			return
		}
		b.handleRCVY(rcvMsg)
		loglogrus.Log.Infof("[Block Ordering] Upper Channel完成对来自节点(%x)的 CODE_RCVY Msg的处理\n", message.From)

	case CODE_INIT:
		loglogrus.Log.Infof("[Block Ordering] Upper Channel收到来自节点(%x)的 CODE_INIT Msg\n", message.From)
		iniMsg, ok := msg.(*InitMessage)
		if !ok {
			loglogrus.Log.Errorf("[Block Ordering] fail in type assert from %x\n", bom.Sender)
			return
		}
		b.handleINIT(iniMsg)
		loglogrus.Log.Infof("[Block Ordering] Upper Channel完成对来自节点(%x)的 CODE_INIT Msg的处理\n", message.From)

	case CODE_SYNC:
		loglogrus.Log.Infof("[Block Ordering] Upper Channel收到来自节点(%x)的 CODE_SYNC Msg\n", message.From)
		synMsg, ok := msg.(*SynchronizeMessage)
		if !ok {
			loglogrus.Log.Errorf("[Block Ordering] fail in type assert from %x\n", bom.Sender)
			return
		}
		b.handleSYNC(synMsg)
		loglogrus.Log.Infof("[Block Ordering] Upper Channel完成对来自节点(%x)的 CODE_SYNC Msg的处理\n", message.From)

	default:
		loglogrus.Log.Warnf("[Block Ordering] Unknown block order message type : %d is received from %x\n", bom.TypeCode, bom.Sender)
		return
	}
}

func (b *BlockOrderAgent) recovery() {
	b.syncProtocal.UseNormalMode()
	b.pc.setVale(true)
	loglogrus.Log.Infof("[Block Ordering] Protect mode is on, now is recovering\n")
}

func (b *BlockOrderAgent) respawn() {
	b.syncProtocal.UseLeaderMode()
	b.pc.setVale(false)
	loglogrus.Log.Infof("[Block Ordering] Protect mode is closed, recovery has finished\n")
}

// at this time, when the next block sent by the booter is invalid, no action is taken
// TODO: Notice other agents to do new booter election when the previous booter is no longer rliable.
func (b *BlockOrderAgent) handleNEXT(msg *NextMessage) {
	b.rc.changeMux.Lock()
	defer b.rc.changeMux.Unlock()

	if b.pc.isActive {
		currentVersion, currentHeight := b.flushState()
		if currentHeight == msg.Height && currentVersion == msg.Version {
			b.respawn()
		} else {
			loglogrus.Log.Warnf("[Block Ordering] Node is still in recovery mode and cannot do the next round of consensus\n")
			return
		}
	}

	currentVersion, currentHeight := b.sc.getState()

	if currentHeight == msg.Height && currentVersion == msg.Version {
		if b.rc.state == state_finish {
			sigMsg := &SignMessage{
				Valid:     BlockInvalid,
				BlockID:   msg.Block.BlockID,
				ReceiptID: common.Hash{},
				Signature: make([]byte, 0),
			}

			if !b.validateManager.LowerValidate(&msg.Block) {
				loglogrus.Log.Warnf("[Block Ordering] the next block %x is invalid, the booter in term may be malicious!", msg.Block.BlockID)
				// at this time, no other action is taken but just ignore this round of block and return an invalid message
				bom, err := CreateBlockOrderMsg(sigMsg, b.selfNode.NodeID, CODE_SIGN)
				if err != nil {
					loglogrus.Log.Warnf("[Block Ordering] fail in create block order message: %v", err)
					return
				}
				b.sendMessage(bom, b.serverInTerm)
				return
			}

			if b.validateManager.HasRepeatTransaction(&msg.Block) {
				loglogrus.Log.Warnf("[Block Ordering] the next block %x is invalid, the booter in term may be malicious!", msg.Block.BlockID)
				// at this time, no other action is taken but just ignore this round of block and return an invalid message
				bom, err := CreateBlockOrderMsg(sigMsg, b.selfNode.NodeID, CODE_SIGN)
				if err != nil {
					loglogrus.Log.Warnf("[Block Ordering] fail in create block order message: %v", err)
					return
				}
				b.sendMessage(bom, b.serverInTerm)
				return
			}

			// a new round starts
			b.rc.state = state_signed
			b.rc.height = currentHeight
			b.rc.version = currentVersion
			b.rc.stuff = &msg.Block
			b.rc.sigMsg = nil
			b.rc.donMsg = nil

			txRcp, wrtSet := b.contractEngine.ExecuteTransactions(b.rc.stuff.Transactions)
			receiptID, err := b.rc.stuff.SetReceipt(b.rc.version, txRcp, wrtSet)
			if err != nil {
				loglogrus.Log.Errorf("[Block Ordering] Upper Consensus failed: The NextBlock (%x) couldn't get transaction receipt, err:%v\n", b.rc.stuff.BlockID, err)
				return
			}
			signature, err := crypto.SignHash(receiptID, b.prvKey)
			if err != nil {
				loglogrus.Log.Errorf("[Block Ordering] Upper Consensus failed: The NextBlock (%x) with recipt %x couldn't be signed, err:%v\n", b.rc.stuff.BlockID, receiptID, err)
				return
			}
			sigMsg.Valid = BlockValid
			sigMsg.ReceiptID = receiptID
			sigMsg.Signature = signature

			bom, err := CreateBlockOrderMsg(sigMsg, b.selfNode.NodeID, CODE_SIGN)
			if err != nil {
				loglogrus.Log.Warnf("[Block Ordering] fail in create block order message: %v\n", err)
				return
			}
			b.sendMessage(bom, b.serverInTerm)
			b.rc.sigMsg = sigMsg
			return

		} else if b.rc.state == state_signed { // round is running, but duplicate Next message is received
			loglogrus.Log.Warnf("[Block Ordering] unexpect error in handleNEXT: dulplicate next message to launch same round of consensus\n")
			// resend the signed message
			bom, err := CreateBlockOrderMsg(b.rc.sigMsg, b.selfNode.NodeID, CODE_SIGN)
			if err != nil {
				loglogrus.Log.Warnf("[Block Ordering] fail in create block order message: %v\n", err)
				return
			}
			b.sendMessage(bom, b.serverInTerm)
			return
		}
	} else if msg.Height > currentHeight {
		loglogrus.Log.Warnf("[Block Ordering] the agent (height: %d) falls behind others (height: %d) and should use protect mode", currentHeight, msg.Height)
		b.recovery()
		return
	} else {
		loglogrus.Log.Errorf("[Block Ordering] unknowmn next message is received, the server is malicious!")
		return
	}

}

func (b *BlockOrderAgent) handleCMIT(msg *CommitMessage) {
	b.rc.changeMux.Lock()
	defer b.rc.changeMux.Unlock()

	// When agent is in protect, the roundContainer would not be updated, so handleCMIT would be nonsense
	if b.pc.isProtect() {
		return
	}

	if b.rc.state == state_finish {
		if b.rc.stuff.BlockID == msg.BlockID && b.rc.stuff.Receipt.ReceiptID == msg.ReceiptID {
			// repeat commit message sent by server, probably it has not collect enough done messages
			// resend the done message
			loglogrus.Log.Warnf("[Block Ordering] Repeat commit message of block %x is received\n", msg.BlockID)
			bom, err := CreateBlockOrderMsg(b.rc.donMsg, b.selfNode.NodeID, CODE_DONE)
			if err != nil {
				loglogrus.Log.Warnf("[Block Ordering] fail in create block order message: %v\n", err)
				return
			}
			b.sendMessage(bom, b.serverInTerm)
		} else {
			// the next message of this round is missing, should do recovery
			loglogrus.Log.Warnf("[Block Ordering] new round finish before commit, may be the NextMessage is missing\n")
			b.recovery()
			return
		}
	}

	//// this case is impossible to happen
	//currentVersion, currentHeight := b.sc.getState()
	//
	//if b.rc.height != currentHeight || b.rc.version != currentVersion {
	//	loglogrus.Log.Warnf("[Block Ordering] The commit message is disorder\n")
	//	b.recovery()
	//	return
	//}

	if b.rc.stuff.BlockID == msg.BlockID && b.rc.height+1 == msg.Height && b.rc.stuff.Receipt.ReceiptID == msg.ReceiptID {
		donMsg := &DoneMessage{
			Valid:   BlockInvalid,
			BlockID: msg.BlockID,
			Height:  msg.Height,
		}

		b.rc.stuff.SetLeaderVotes(msg.Signatures)

		if !b.validateManager.UpperValidate(b.rc.stuff) {
			loglogrus.Log.Warnf("[Block Ordering] The commit block is not valid, the booter maybe malicious!\n")
			bom, err := CreateBlockOrderMsg(donMsg, b.selfNode.NodeID, CODE_DONE)
			if err != nil {
				loglogrus.Log.Warnf("[Block Ordering] fail in create block order message: %v\n", err)
				return
			}
			b.sendMessage(bom, b.serverInTerm)
			b.rc.state = state_wait
			b.rc.donMsg = donMsg
			return
		}

		donMsg.Valid = BlockValid
		bom, err := CreateBlockOrderMsg(donMsg, b.selfNode.NodeID, CODE_DONE)
		if err != nil {
			loglogrus.Log.Warnf("[Block Ordering] fail in create block order message: %v\n", err)
			return
		}
		b.sendMessage(bom, b.serverInTerm)

		loglogrus.Log.Infof("[Block Ordering] 准备提交区块到数据库, current Block Hash:%x \n", b.rc.stuff.BlockID)

		err = b.stateManager.CommitBlock(b.rc.stuff)
		if err != nil {
			loglogrus.Log.Errorf("[Block Ordering] fatal error: fail in block commit: %v\n", err)
		}

		loglogrus.Log.Infof("[Block Ordering] 完成区块提交, current Block Hash:%x \n", b.rc.stuff.BlockID)

		b.flushState()

		b.rc.state = state_finish
		b.rc.donMsg = donMsg
		return

	} else {
		// the commit message is not match the server state before, either the server is malicious or
		// the agent misses a commit message and a next message, should do recovery
		loglogrus.Log.Warnf("[Block Ordering] the block committed is not matched in this round\n")
		b.recovery()
		return
	}
}

// Recovery is happened when the next message should be replaced
func (b *BlockOrderAgent) handleRCVY(msg *RecoveryMessage) {
	b.rc.changeMux.Lock()
	defer b.rc.changeMux.Unlock()

	loglogrus.Log.Warnf("[Block Ordering] Recovery message is received, the repeat count is %d, version %x, height %d, blockID %x\n", msg.RepeatCount, msg.Version, msg.Height, msg.Block.BlockID)

	if b.pc.isActive {
		currentVersion, currentHeight := b.flushState()
		if currentHeight == msg.Height && currentVersion == msg.Version {
			b.respawn()
		} else {
			loglogrus.Log.Warnf("[Block Ordering] Node is still in recovery mode and cannot do the next round of consensus\n")
			return
		}
	}

	currentVersion, currentHeight := b.sc.getState()
	if currentVersion == msg.Version && currentHeight == msg.Height { // this recovery message is valid
		sigMsg := &SignMessage{
			Valid:     BlockInvalid,
			BlockID:   msg.Block.BlockID,
			ReceiptID: common.Hash{},
			Signature: make([]byte, 0),
		}

		if !b.validateManager.LowerValidate(&msg.Block) {
			loglogrus.Log.Warnf("[Block Ordering] the next block %x is invalid, the booter in term may be malicious!", msg.Block.BlockID)
			// at this time, no other action is taken but just ignore this round of block and return an invalid message
			bom, err := CreateBlockOrderMsg(sigMsg, b.selfNode.NodeID, CODE_SIGN)
			if err != nil {
				loglogrus.Log.Warnf("[Block Ordering] fail in create block order message: %v", err)
				return
			}
			b.sendMessage(bom, b.serverInTerm)
			return
		}

		if b.validateManager.HasRepeatTransaction(&msg.Block) {
			loglogrus.Log.Warnf("[Block Ordering] the next block %x is invalid, the booter in term may be malicious!", msg.Block.BlockID)
			// at this time, no other action is taken but just ignore this round of block and return an invalid message
			bom, err := CreateBlockOrderMsg(sigMsg, b.selfNode.NodeID, CODE_SIGN)
			if err != nil {
				loglogrus.Log.Warnf("[Block Ordering] fail in create block order message: %v", err)
				return
			}
			b.sendMessage(bom, b.serverInTerm)
			return
		}

		// a new round starts
		b.rc.state = state_signed
		b.rc.height = currentHeight
		b.rc.version = currentVersion
		b.rc.stuff = &msg.Block
		b.rc.sigMsg = nil
		b.rc.donMsg = nil

		txRcp, wrtSet := b.contractEngine.ExecuteTransactions(b.rc.stuff.Transactions)
		receiptID, err := b.rc.stuff.SetReceipt(b.rc.version, txRcp, wrtSet)
		if err != nil {
			loglogrus.Log.Errorf("[Block Ordering] Upper Consensus failed: The NextBlock (%x) couldn't get transaction receipt, err:%v\n", b.rc.stuff.BlockID, err)
			return
		}
		signature, err := crypto.SignHash(receiptID, b.prvKey)
		if err != nil {
			loglogrus.Log.Errorf("[Block Ordering] Upper Consensus failed: The NextBlock (%x) with recipt %x couldn't be signed, err:%v\n", b.rc.stuff.BlockID, receiptID, err)
			return
		}
		sigMsg.Valid = BlockValid
		sigMsg.ReceiptID = receiptID
		sigMsg.Signature = signature

		bom, err := CreateBlockOrderMsg(sigMsg, b.selfNode.NodeID, CODE_SIGN)
		if err != nil {
			loglogrus.Log.Warnf("[Block Ordering] fail in create block order message: %v\n", err)
			return
		}
		b.sendMessage(bom, b.serverInTerm)
		b.rc.sigMsg = sigMsg
		return
	} else if msg.Height > currentHeight {
		loglogrus.Log.Warnf("[Block Ordering] the agent (height: %d) falls behind others (height: %d) and should use protect mode", currentHeight, msg.Height)
		b.recovery()
		return
	} else {
		loglogrus.Log.Errorf("[Block Ordering] stale recovery message is received, the server is malicious!")
		return
	}
}

func (b *BlockOrderAgent) handleINIT(msg *InitMessage) {
	currentVersion, currentHeight := b.flushState()
	staMsg := &StateMessage{
		Version: currentVersion,
		Height:  currentHeight,
	}
	bom, err := CreateBlockOrderMsg(staMsg, b.selfNode.NodeID, CODE_STAT)
	if err != nil {
		loglogrus.Log.Warnf("[Block Ordering] fail in create block order message: %v\n", err)
		return
	}
	b.sendMessage(bom, b.serverInTerm)

	if msg.Height > currentHeight {
		loglogrus.Log.Warnf("[Block Ordering] The block chain (height %d) is not the latest compared with the booter (height %d), do recovery\n", currentHeight, msg.Height)
		b.recovery()
	}
}

func (b *BlockOrderAgent) handleSYNC(msg *SynchronizeMessage) {
	_, currentHeight := b.flushState()
	if currentHeight >= msg.Height {
		return
	} else {
		loglogrus.Log.Warnf("[Block Ordering] Got sync message, self height: %d, remote height: %d, do recovery\n", currentHeight, msg.Height)
		b.recovery()
	}
}

func (b *BlockOrderAgent) checkSelf() bool {
	if b.msgChannel == nil {
		return false
	}
	if b.selfNode == nil {
		return false
	}
	if b.prvKey == nil {
		return false
	}
	if b.serverInTerm == (common.NodeID{}) {
		return false
	}
	if b.stateManager == nil {
		return false
	}
	if b.blockCache == nil {
		return false
	}
	if b.validateManager == nil {
		return false
	}
	if b.contractEngine == nil {
		return false
	}
	if b.rc == nil {
		return false
	}
	if b.pc == nil {
		return false
	}
	if b.sc == nil {
		return false
	}
	localVersion, localHeight := b.stateManager.GetCurrentState()
	currentVersion, currentHeight := b.sc.getState()
	if localVersion != currentVersion || localHeight != currentHeight {
		return false
	}
	if b.syncProtocal == nil {
		return false
	}
	return true
}

func (b *BlockOrderAgent) Start() error {
	if !b.checkSelf() {
		loglogrus.Log.Warnf("[Block Ordering] Upper Consensus failed: Leader (%x) could not start BlockOrderAgent!\n", b.selfNode.NodeID)
		return fmt.Errorf("fail in start OrderServiceAgent")
	}

	// NOTE: use the leader mode!!!!
	b.respawn()

	// not need
	//b.flushState()

	go b.blockUploadLoop()

	loglogrus.Log.Infof("[Block Ordering] Upper Consensus: Leader (%x) start to run BlockOrderAgent successfully\n", b.selfNode.NodeID)
	return nil
}

func (b *BlockOrderAgent) Stop() {
	close(b.quit)
}

func (b *BlockOrderAgent) blockUploadLoop() {
	tk := time.NewTicker(blockUploadCycle)
	for {
		select {
		case <-tk.C:
			b.uploadBlocks()
		case <-b.quit:
			tk.Stop()
			return
		}
	}
}

// at this time, nothing has been done to make sure a block is really received by the block order server
// TODO: Resend block if a block uploaded is not seen on the chain for a long time.
func (b *BlockOrderAgent) uploadBlocks() {
	blocks := b.blockCache.BackUnmarkedLocalBlock()
	for _, block := range blocks {
		b.blockCache.MarkLocalBlock(block.BlockID)
		expireTime := time.Now().Add(block_expiration)
		uplMsg := &UploadMessage{
			ExpireTime: uint64(expireTime.Unix()),
			Block:      *block,
		}
		bom, err := CreateBlockOrderMsg(uplMsg, b.selfNode.NodeID, CODE_UPLD)
		if err != nil {
			loglogrus.Log.Warnf("[Block Ordering] Consensus: fail in uploadBlocks, %v", err)
			continue
		}
		b.sendMessage(bom, b.serverInTerm)
	}
}

func (b *BlockOrderAgent) sendMessage(bom *BlockOrderMessage, receiver common.NodeID) {
	payload, err := rlp.EncodeToBytes(bom)
	if err != nil {
		loglogrus.Log.Warnf("[Block Ordering] Consensus: fail in sendMessage, receiver is: %x : %v", receiver, err)
		return
	}
	msg := b.msgChannel.NewUpperConsensusMessage(b.selfNode.NodeID, payload)
	err = b.msgChannel.MsgSend(msg, receiver)
	if err != nil {
		loglogrus.Log.Warnf("[Block Ordering] Consensus: fail in sendMessage, receiver is: %x : %v", receiver, err)
		return
	}
}
