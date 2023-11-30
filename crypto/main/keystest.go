package main

import (
	"dpchain/crypto"
	"dpchain/crypto/keys"
	"dpchain/crypto/randentropy"
	"encoding/base64"
	"fmt"
	"os"
)

func main() {
	// 1. 创建一个新的 KeyStorePassphrase 实例
	keyStore := keys.NewKeyStorePassphrase("./mykeys") // "./mykeys" 是存储密钥的目录

	// 2. 生成新的密钥
	password := "" // 用于加密私钥的密码
	key, err := keyStore.GenerateNewKey(randentropy.Reader, password)
	if err != nil {
		fmt.Println("Failed to generate new key:", err)
		os.Exit(1)
	}

	// 3. 获取公钥和私钥
	privateKeyBytes:= crypto.FromECDSA(key.PrivateKey)
	// publicKeyBytes := key.PrivateKey.PublicKey
	privateKey := base64.StdEncoding.EncodeToString(privateKeyBytes)
    // publicKey := base64.StdEncoding.EncodeToString(publicKeyBytes)
	// 打印公钥和私钥信息（可选）
	fmt.Println("Private Key:", privateKey)
	// fmt.Println("Public Key:", publicKey)
}
