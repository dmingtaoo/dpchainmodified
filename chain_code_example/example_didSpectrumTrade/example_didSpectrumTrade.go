package main

import (
	cc "dpchain/core/contract/chainCodeSupport"
	"flag"
	"fmt"
	"strings"
	"time"
	"dpchain/common"
	"dpchain/crypto"
	"encoding/json"
)

var (
	ERROR_FUNCTION_ARGS = fmt.Errorf("unmatched arguments")
)

var (
	CONTRACT_NAME = "DID::SPECTRUM::TRADE"
)

var dataList [][]byte

type VC struct {
    Identifier     string `json:"identifier"` // VC ID
    Subject        string `json:"subject"`    // SP DID
    Issuer         string `json:"issuer"`     // UE DID
    Validity       string `json:"validity"`   // Valid time
    Purpose        string `json:"purpose"`    // 用途
	Signature      string `json:"signature"`  // UE Signature
    Reassign 	   string `json:"reassign"`// 是否允许二次转让
    IssuerAddress  string `json:"IssuerAddress"`// issuer address
    SubjectAddress string `json:"SubjectAddress"`// subject address
}



func isDateExpired(dateStr string) (bool, error) {
	// 解析日期字符串
	expiryDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return false, fmt.Errorf("invalid date format: %v", err)
	}

	// 获取当前日期（去除时分秒）
	currentDate := time.Now().Truncate(24 * time.Hour)

	// 如果expiryDate在currentDate之前，则认为已过期
	return expiryDate.Before(currentDate), nil
}

func StringToMap(str string) (map[string]string, error) {
    resultMap := make(map[string]string)
    pairs := strings.Split(str, ",")
    for _, pair := range pairs {
        // 找到第一个冒号的位置
        idx := strings.Index(pair, ":")
        if idx == -1 {
            return nil, fmt.Errorf("invalid pair (no colon found): %s", pair)
        }

        key := pair[:idx]
        value := pair[idx+1:]
        resultMap[key] = value
    }
    return resultMap, nil
}



func VCValid(args [][]byte, ds cc.DperServicePipe) ([][]byte, error) {
	if len(args) != 1 {
		return nil, ERROR_FUNCTION_ARGS
	}
	str1 := string(args[0])
	mymap, _ := StringToMap(str1)

	// 提取并创建 Credential 结构体
	var vc VC
	var exists bool

	if vc.Identifier, exists = mymap["Identifier"]; !exists {
		result := [][]byte{[]byte(fmt.Sprintf("noid"))}
		return result,nil
	}
	if vc.Subject, exists = mymap["Subject"]; !exists {
		result := [][]byte{[]byte(fmt.Sprintf("nosubject"))}
		return result,nil
	}
	if vc.Issuer, exists = mymap["Issuer"]; !exists {
		result := [][]byte{[]byte(fmt.Sprintf("noissuer"))}
		return result,nil
	}
	if vc.Validity, exists = mymap["Validity"]; !exists {
		result := [][]byte{[]byte(fmt.Sprintf("novalidity"))}
		return result,nil
	}
	if vc.Purpose, exists = mymap["Purpose"]; !exists {
		result := [][]byte{[]byte(fmt.Sprintf("nopurpose"))}
		return result,nil
	} 
	if vc.Signature, exists = mymap["Signature"]; !exists {
		result := [][]byte{[]byte(fmt.Sprintf("nosignature"))}
		return result,nil
	} 
	if vc.Reassign, exists = mymap["Reassign"]; !exists {
		result := [][]byte{[]byte(fmt.Sprintf("noreassign"))}
		return result,nil
	} 
	if vc.IssuerAddress, exists = mymap["IssuerAddress"]; !exists {
		result := [][]byte{[]byte(fmt.Sprintf("no issueraddress"))}
		return result,nil
	} 
	if vc.SubjectAddress, exists = mymap["SubjectAddress"]; !exists {
		result := [][]byte{[]byte(fmt.Sprintf("no subjectaddress"))}
		return result,nil
	} 

	// 在这里添加有关 Credential 的验证
	expired, err := isDateExpired(vc.Validity)
	if err != nil {
		result := [][]byte{[]byte(fmt.Sprintf("expired error"))}
		return result,nil
	}
	if expired {
		return nil,nil
	} 
	//再验证签名
	signature := vc.Signature
	address := vc.IssuerAddress
    // 假设地址是返回数组的第一个元素
	msg := crypto.Sha3Hash([]byte(vc.Identifier))
	sig := common.Hex2Bytes(signature)
	add := common.HexToAddress(address)
	ok, err := crypto.SignatureValid(add, sig, msg)
	// 示例：返回 Credential 结构体的字符串表示
	if err!=nil {
		result := [][]byte{[]byte(fmt.Sprintf("signaturevalid wrong", vc))}
		return result, nil
	}else{
		if ok {
						// 序列化 Credential 数据为 JSON
			jsonData, err := json.Marshal(vc)
			if err != nil {
				// 处理序列化错误
				return nil, fmt.Errorf("error marshalling VC: %v", err)
			}

			// 将序列化后的数据保存到区块链
			// 假设我们使用 vc.Identifier 作为存储的键
			if err := ds.UpdateStatus([]byte(vc.Identifier), jsonData); err != nil {
				// 处理 UpdateStatus 的错误
				return nil, fmt.Errorf("failed to update status: %v", err)
			}
		result := [][]byte{[]byte(fmt.Sprintf("Credential: %+v", vc))}
		return result, nil
	}else{
		return nil, nil
	}
	}
}

func GetVC(args [][]byte, ds cc.DperServicePipe) ([][]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("GetVC requires exactly 1 argument, got %d", len(args))
	}

	// 使用 Identifier 作为键来获取 Credential 数据
	storedData, err := ds.GetStatus(args[0])
	if err != nil {
		return nil, fmt.Errorf("error getting VC status: %v", err)
	}

	// 直接返回检索到的 JSON 字符串
	return [][]byte{storedData}, nil
}

func SetDID(args [][]byte, ds cc.DperServicePipe) ([][]byte, error) {
    if len(args) != 2 {
        return nil, fmt.Errorf("SetDID requires exactly 2 arguments, got %d", len(args))
    }

    str1 := string(args[0])
    str2 := string(args[1])

    if len(str1) < 4 || str1[:4] != "DID:" || len(str2) < 8 || str2[:8] != "address:" {
        return nil, fmt.Errorf("invalid argument format")
    }

    if err := ds.UpdateStatus(args[0], args[1]); err != nil {
        // 处理 UpdateStatus 的错误
        return nil, fmt.Errorf("failed to update status: %v", err)
    }

    // 注意：dataList 的使用可能需要考虑线程安全性
    dataList = append(dataList, args[0], args[1])

    return [][]byte{[]byte("setDID succeed")}, nil
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
		"VCValid":	VCValid,
		"GetVC":	GetVC,
		// "VCValidd":	VCValidd,
	}
	err := cc.InstallContractPipe(CONTRACT_NAME, funcMap, *Dper_pipe)
	if err != nil {
		fmt.Print(err)
	} else {
		fmt.Print("install success")
	}
	cc.ContractExecutePipe(*Dper_pipe, *local_pipe, funcMap)
}

// func VCValidd(args [][]byte, ds cc.DperServicePipe) ([][]byte, error) {
// 	if len(args) != 1 {
// 		return nil, ERROR_FUNCTION_ARGS
// 	}
// 	str1 := string(args[0])
// 	mymap, err:= StringToMap(str1)
// 	if err!=nil{
// 		result := [][]byte{[]byte(fmt.Sprintf("damn"))}
// 		return result,nil
// 	}
// 	var vc VC
// 	var exists bool

// 	if vc.Identifier, exists = mymap["Identifier"]; !exists {
// 		result := [][]byte{[]byte(fmt.Sprintf("noid"))}
// 		return result,nil
// 	}
// 	if vc.Subject, exists = mymap["Subject"]; !exists {
// 		result := [][]byte{[]byte(fmt.Sprintf("nosubject"))}
// 		return result,nil
// 	}
// 	if vc.Issuer, exists = mymap["Issuer"]; !exists {
// 		result := [][]byte{[]byte(fmt.Sprintf("noissuer"))}
// 		return result,nil
// 	}
// 	if vc.Validity, exists = mymap["Validity"]; !exists {
// 		result := [][]byte{[]byte(fmt.Sprintf("novalidity"))}
// 		return result,nil
// 	}
// 	if vc.Purpose, exists = mymap["Purpose"]; !exists {
// 		result := [][]byte{[]byte(fmt.Sprintf("nopurpose"))}
// 		return result,nil
// 	} 
// 	if vc.Signature, exists = mymap["Signature"]; !exists {
// 		result := [][]byte{[]byte(fmt.Sprintf("nosignature"))}
// 		return result,nil
// 	} 
// 	if vc.Reassign, exists = mymap["Reassign"]; !exists {
// 		result := [][]byte{[]byte(fmt.Sprintf("noreassign"))}
// 		return result,nil
// 	} 
// 	expired, err := isDateExpired(vc.Validity)
// 	if err != nil {
// 		result := [][]byte{[]byte(fmt.Sprintf("expired error"))}
// 		return result,nil
// 	}
// 	if expired {
// 		result := [][]byte{[]byte(fmt.Sprintf("expired"))}
// 		return result,nil
// 	} 
// 	//再验证签名
// 	signature := vc.Signature
// 	addressArgs := [][]byte{[]byte(vc.Issuer)}
//     addressResult, err := GetAddress(addressArgs, ds)
//     if err != nil {
// 		result := [][]byte{[]byte(fmt.Sprintf("address failed"))}
// 		return result,nil
//     }
//     // 假设地址是返回数组的第一个元素
//     address := string(addressResult[0])
// 	msg := crypto.Sha3Hash([]byte(vc.Identifier))
// 	sig := common.Hex2Bytes(signature)
// 	add := common.HexToAddress(address)
// 	ok, err := crypto.SignatureValid(add, sig, msg)
// 	// 示例：返回 Credential 结构体的字符串表示
// 	if err!=nil {
// 		return nil,err
// 	}else{
// 		if ok {
// 		result := [][]byte{[]byte(fmt.Sprintf("Credential: %+v", vc))}
// 		return result, nil
// 	}else{
// 		return nil,nil
// 	}
// 	}
// 	result := [][]byte{[]byte(fmt.Sprintf("Credential: %+v", vc))}
// 	// result := [][]byte{[]byte("setDID succeed")}
// 		return result, nil
// }