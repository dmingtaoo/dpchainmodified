package router

import (
	"dpchain/common"
	"fmt"
	"testing"
	// "path/filepath"
)

// 测试 SignContent 函数
func TestSignContent(t *testing.T) {
	// 测试使用的输入数据
	// relativePath := "../../../../dper/client/auto/dper_dper1/accounts"
	// keyDirPath, err := filepath.Abs(relativePath)
	// if err != nil {
	// 	fmt.Println("Error converting to absolute path:", err)
	// 	return
	// }
	// fmt.Println("Absolute path:", keyDirPath)
	keyDirPath := "/Users/dengmingtao/Documents/GitHub/dpchain/dper/client/auto/dper_dper1/accounts"
	keyAddr := common.HexToAddress("03b0758c578187cfb0763e20118635d4fd2e704b") // 替换为实际的密钥地址
	testContent := "1234"
	
	// 调用 SignContent 函数进行签名
	signature, err := SignContent(keyDirPath, keyAddr, testContent)
	if err != nil {
		t.Fatalf("SignContent failed: %v", err)
	}
	fmt.Println(signature)
	fmt.Printf("%s",signature)
	// 验证签名 - 这里的验证取决于你的具体要求
	// 例如，可以检查签名是否为非空字符串
	if signature == "" {
		t.Errorf("Expected a signature, but got an empty string")
	}

	// 更多的验证可以根据需要添加
	// 例如，如果可能的话，可以验证签名是否能被用来验证原始内容
}
