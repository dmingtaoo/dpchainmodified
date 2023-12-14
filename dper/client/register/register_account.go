package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strings"
)

// AccountsData 和 Response 用于解析 GET 请求的 JSON 响应
type AccountsData struct {
	Accounts []string `json:"accounts"`
}

type Response struct {
	Code int          `json:"code"`
	Data AccountsData `json:"data"`
	Msg  string       `json:"msg"`
}

// FetchAccounts 发送 HTTP GET 请求到给定的 URL 并提取 accounts 字段
// FetchAccounts 发送 HTTP GET 请求到给定的 URL 并提取第一个账户
func FetchAccounts(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	if len(response.Data.Accounts) > 0 {
		// 移除账户字符串中的大括号
		cleanedAccount := strings.Trim(response.Data.Accounts[0], "{}")
		return cleanedAccount, nil
	}

	return "", nil // 如果没有账户，则返回空字符串
}

func PostMultipartRequest(url, account, password string) (string, error) {
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	// 添加表单字段
	_ = writer.WriteField("account", account)
	_ = writer.WriteField("password", password)

	// 关闭writer，这将终止multipart消息
	err := writer.Close()
	if err != nil {
		return "", err
	}

	// 构造请求
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 发送请求
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// 读取响应
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
func SendTxCheck(url string) (string, error) {
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	if err := writer.WriteField("mode", "on"); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
func main() {
	getURLs := []string{
		"http://127.0.0.1:8001/dper/accountsList",
		"http://127.0.0.1:8002/dper/accountsList",
		"http://127.0.0.1:8003/dper/accountsList",
		"http://127.0.0.1:8004/dper/accountsList",
	}
	registerURLs := []string{
		"http://127.0.0.1:8001/dper/useAccount",
		"http://127.0.0.1:8002/dper/useAccount",
		"http://127.0.0.1:8003/dper/useAccount",
		"http://127.0.0.1:8004/dper/useAccount",
	}
	txCheckURLs := []string{
		"http://127.0.0.1:8001/dper/txCheck",
		"http://127.0.0.1:8002/dper/txCheck",
		"http://127.0.0.1:8003/dper/txCheck",
		"http://127.0.0.1:8004/dper/txCheck",
	}
	password := "" // 使用同一个密码，或根据需要修改

	for i, getURL := range getURLs {
		account, err := FetchAccounts(getURL)
		if err != nil {
			fmt.Printf("Error fetching account from %s: %v\n", getURL, err)
			continue
		}

		if account != "" {
			_, err := PostMultipartRequest(registerURLs[i], account, password)
			if err != nil {
				fmt.Printf("Error registering account %s at %s: %v\n", account, registerURLs[i], err)
				continue
			}
			fmt.Printf("Successfully registered account %s at %s\n", account, registerURLs[i])
		}
	}
	for _, url := range txCheckURLs {
		response, err := SendTxCheck(url)
		if err != nil {
			fmt.Printf("Error sending txCheck to %s: %v\n", url, err)
			continue
		}
		fmt.Printf("Successfully sent txCheck to %s: %s\n", url, response)
	}
}
