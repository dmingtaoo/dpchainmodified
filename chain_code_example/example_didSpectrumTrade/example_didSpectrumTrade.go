package main

import (
	cc "dpchain/core/contract/chainCodeSupport"
	"flag"
	"fmt"
	"strings"
)

var (
	ERROR_FUNCTION_ARGS = fmt.Errorf("unmatched arguments")
)

var (
	CONTRACT_NAME = "DID::SPECTRUM::TRADE"
)

var dataList [][]byte

type VerifiedCredential struct {
	UeDID       string `json:"ueDID"`
	SpDID       string `json:"spDID"`
	UeSignature []byte `json:"ueSignature"`
	Lifetime    uint8  `json:"lifetime"`
	Role        string `json:"role"` // User or Supervisor
}

func StringToMap(str string) (map[string]string, error) {
	resultMap := make(map[string]string)
	// 使用逗号分隔字符串，获取键值对
	pairs := strings.Split(str, ",")
	for _, pair := range pairs {
		kv := strings.Split(pair, ":")
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid pair: %s", pair)
		}
		key := kv[0]
		value := kv[1]
		resultMap[key] = value
	}
	return resultMap, nil
}

func SetDID(args [][]byte, ds cc.DperServicePipe) ([][]byte, error) {
	if len(args) != 2 {
		return nil, ERROR_FUNCTION_ARGS
	}
	str1 := string(args[0])
	str2 := string(args[1])
	if (len(str1) >= 4 && str1[:4] == "DID:") && (len(str2) >= 10 && str2[:10] == "address:") {
		ds.UpdateStatus(args[0], args[1])
		result := [][]byte{[]byte("setDID succeed")}
		dataList = append(dataList, args[0], args[1])
		return result, nil
	} else {
		return nil, ERROR_FUNCTION_ARGS
	}
}

func GetAddress(args [][]byte, ds cc.DperServicePipe) ([][]byte, error) {
	if len(args) != 1 {
		return nil, ERROR_FUNCTION_ARGS
	}
	value, err := ds.GetStatus(args[0])
	if err != nil {
		return nil, err
	}
	result := [][]byte{value}
	return result, nil
}

func GetDIDList(args [][]byte, ds cc.DperServicePipe) ([][]byte, error) {
	data := dataList
	return data, nil
}

func main() {
	local_pipe := flag.String("local_pipe", "", "")
	Dper_pipe := flag.String("dper_pipe", "", "")
	flag.Parse()
	funcMap := map[string]cc.ContractFuncPipe{
		"SetDID":     SetDID,
		"GetAddress": GetAddress,
		"GetDIDList": GetDIDList,
	}
	err := cc.InstallContractPipe(CONTRACT_NAME, funcMap, *Dper_pipe)
	if err != nil {
		fmt.Print(err)
	} else {
		fmt.Print("install success")
	}
	cc.ContractExecutePipe(*Dper_pipe, *local_pipe, funcMap)
}
