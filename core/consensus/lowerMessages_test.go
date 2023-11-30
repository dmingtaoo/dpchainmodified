package consensus

import (
	"dpchain/common"
	"reflect"
	"testing"
)

func TestWrappedLowerConsensusMessage(t *testing.T) {
	ppm := &PrePrepareMsg{
		head: CommonHead{
			Consensus: []byte("test consensus"),
			TypeCode:  StatePrePrepare,
			Sender:    common.NodeID{},
		},
		Version: common.StringToHash("TestVersion"),
		Nonce:   0,
		TxOrder: []common.Hash{common.StringToHash("tx1"), common.StringToHash("tx2"), common.StringToHash("tx3")},
	}
	roundID := ppm.ComputeRoundID()
	if roundID != ppm.head.RoundID {
		t.Fatalf("fail in set roundID")
	}
	wlcm := ppm.Wrap()
	reconstructppm, err := wlcm.UnWrap()
	if err != nil {
		t.Fatalf("fail in unwrap: %v", err)
	}
	if !reflect.DeepEqual(reconstructppm, ppm) {
		t.Fatalf("PrePrepareMsg changed after deserialized")
	}

}
