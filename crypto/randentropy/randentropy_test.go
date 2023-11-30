package randentropy

import (
	"encoding/hex"
	"fmt"
	"testing"
)

// TestRandEntropy 测试 randEntropy 类型是否正确生成随机字节
func TestRandEntropy(t *testing.T) {
	var re randEntropy
	length := 10 // 我们想要生成的随机字节的长度
	bytes := make([]byte, length)

	n, err := re.Read(bytes)
	if err != nil {
		t.Fatalf("Read() error = %v, wantErr %v", err, false)
	}
	if n != length {
		t.Errorf("Read() returned %v bytes, want %v", n, length)
	}

	// 打印生成的随机字节
	fmt.Printf("Generated Random Bytes: %s", hex.EncodeToString(bytes))
}
