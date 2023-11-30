package app

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

type Communication struct {
}

// Send 发送HTTP请求
func (c *Communication) Send(sourcePort, targetPort string, data interface{}) ([]byte, error) {
	url := fmt.Sprintf("http://127.0.0.1:%s/send", targetPort)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Receive 接收HTTP请求
func (c *Communication) Receive(port string) error {
	http.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Received data at port %s", port)
	})

	fmt.Printf("Server started at port %s\n", port)
	return http.ListenAndServe(":"+port, nil)
}
