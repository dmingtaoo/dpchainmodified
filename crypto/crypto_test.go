package crypto

import (
	"dpchain/common"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestSignatureValid(t *testing.T) {
	// 你需要创建一个测试用的地址、签名和哈希
	// 用于测试 SignatureValid 函数

	// 示例地址
	address := common.HexToAddress("03b0758c578187cfb0763e20118635d4fd2e704b")

	// 示例哈希
	hash := Sha3Hash([]byte("1234"))
	fmt.Println(hash)
	hexSignature := "03d5f293d3e7920d78ca962a719d873bca609f7bf8d61dccfff1bd4dc84c7fd92a8cf5625da975f15e4bf1f5a2c599563f9bca3faf8a7cf46fb1872175b19a0101"

	// 将十六进制字符串转换为字节切片

	signature, err := hex.DecodeString(hexSignature)
	valid, err := SignatureValid(address, signature, hash)

	if err != nil {
		t.Errorf("Error while verifying signature: %v", err)
	}

	if valid {
		t.Logf("Signature is valid for the provided address.")
	} else {
		t.Errorf("Signature is not valid for the provided address.")
	}
}

// func TestValid(t *testing.T) {
// 	// 3a0633... is signature of "1234"
// 	sig, err := hex.DecodeString("3a0633e4ab8326e0641914174aa91600816456945f42227ab63ab825698de90f4db53b0454f55c45c50b7acc514f0477dc8976109ff00fd889fb5a36b656ee3201")
// 	if err != nil {
// 		fmt.Println("hex decode err!")
// 	}
// 	fmt.Println("sig: ", sig)
// 	valid, err := SignatureValid(common.HexToAddress("3036edb382285d6c933881c9c67df40adc360d94"), sig, Sha3Hash([]byte("1234")))
// 	if err != nil {
// 		fmt.Println("err")
// 	}
// 	fmt.Println("result: ", valid)
// }
