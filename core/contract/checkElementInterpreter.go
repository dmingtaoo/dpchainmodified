package contract

import (
	"bytes"
	"dpchain/core/eles"
)

type CheckElementInterpreter interface {
	validate(pce *PrecompileContractEngine, checkEle eles.CheckElement) bool
	// validate_re(pce *RemoteContractEngine, checkEle eles.CheckElement) bool
	validate_pipe(pce *PipeContractEngine, checkEle eles.CheckElement) bool
}

const (
	CHECK_TYPE_COMMON_EXIST uint8 = iota
	CHECK_TYPE_COMMON_EXIST_NOT

	CHECK_TYPE_BYTES_EQUAL
	CHECK_TYPE_BYTES_EQUAL_NOT

	CHECK_TYPE_BYTES_LESS
	CHECK_TYPE_BYTES_LESS_NOT

	CHECK_TYPE_BYTES_GREATER
	CHECK_TYPE_BYTES_GREATER_NOT
)

type commonCheckElementInterpreter struct{}

func (cei *commonCheckElementInterpreter) validate(pce *PrecompileContractEngine, checkEle eles.CheckElement) bool {
	checkValue := checkEle.Value
	ele, err := pce.Get(checkEle.ValueAddress)
	if err != nil {
		return false
	}
	eleValue := ele.toBytes()
	switch checkEle.CheckType {
	case CHECK_TYPE_COMMON_EXIST:
		return err == nil
	case CHECK_TYPE_COMMON_EXIST_NOT:
		return err != nil
	case CHECK_TYPE_BYTES_EQUAL:
		return bytes.Equal(eleValue, checkValue)
	case CHECK_TYPE_BYTES_EQUAL_NOT:
		return !bytes.Equal(eleValue, checkValue)
	case CHECK_TYPE_BYTES_LESS:
		return bytes.Compare(eleValue, checkValue) == -1
	case CHECK_TYPE_BYTES_LESS_NOT:
		return bytes.Compare(eleValue, checkValue) != -1
	case CHECK_TYPE_BYTES_GREATER:
		return bytes.Compare(eleValue, checkValue) == 1
	case CHECK_TYPE_BYTES_GREATER_NOT:
		return bytes.Compare(eleValue, checkValue) != 1
	default:
		return false
	}
}

// func (cei *commonCheckElementInterpreter) validate_re(pce *RemoteContractEngine, checkEle eles.CheckElement) bool {
// 	checkValue := checkEle.Value
// 	ele, err := pce.Get(checkEle.ValueAddress)
// 	if err != nil {
// 		return false
// 	}
// 	eleValue := ele.toBytes()
// 	switch checkEle.CheckType {
// 	case CHECK_TYPE_COMMON_EXIST:
// 		return err == nil
// 	case CHECK_TYPE_COMMON_EXIST_NOT:
// 		return err != nil
// 	case CHECK_TYPE_BYTES_EQUAL:
// 		return bytes.Equal(eleValue, checkValue)
// 	case CHECK_TYPE_BYTES_EQUAL_NOT:
// 		return !bytes.Equal(eleValue, checkValue)
// 	case CHECK_TYPE_BYTES_LESS:
// 		return bytes.Compare(eleValue, checkValue) == -1
// 	case CHECK_TYPE_BYTES_LESS_NOT:
// 		return bytes.Compare(eleValue, checkValue) != -1
// 	case CHECK_TYPE_BYTES_GREATER:
// 		return bytes.Compare(eleValue, checkValue) == 1
// 	case CHECK_TYPE_BYTES_GREATER_NOT:
// 		return bytes.Compare(eleValue, checkValue) != 1
// 	default:
// 		return false
// 	}
// }
func (cei *commonCheckElementInterpreter) validate_pipe(pce *PipeContractEngine, checkEle eles.CheckElement) bool {
	checkValue := checkEle.Value
	ele, err := pce.Get(checkEle.ValueAddress)
	if err != nil {
		return false
	}
	eleValue := ele.toBytes()
	switch checkEle.CheckType {
	case CHECK_TYPE_COMMON_EXIST:
		return err == nil
	case CHECK_TYPE_COMMON_EXIST_NOT:
		return err != nil
	case CHECK_TYPE_BYTES_EQUAL:
		return bytes.Equal(eleValue, checkValue)
	case CHECK_TYPE_BYTES_EQUAL_NOT:
		return !bytes.Equal(eleValue, checkValue)
	case CHECK_TYPE_BYTES_LESS:
		return bytes.Compare(eleValue, checkValue) == -1
	case CHECK_TYPE_BYTES_LESS_NOT:
		return bytes.Compare(eleValue, checkValue) != -1
	case CHECK_TYPE_BYTES_GREATER:
		return bytes.Compare(eleValue, checkValue) == 1
	case CHECK_TYPE_BYTES_GREATER_NOT:
		return bytes.Compare(eleValue, checkValue) != 1
	default:
		return false
	}
}
