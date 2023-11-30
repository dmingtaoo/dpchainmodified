package router

import (
	"dpchain/api"
	"dpchain/common"
	"dpchain/core/consensus"
	"dpchain/core/eles"
	"dpchain/crypto"
	"dpchain/crypto/randentropy"
	"dpchain/dper/transactionCheck"
	"dpchain/ginHttp/pkg/app"
	e "dpchain/ginHttp/pkg/error"
	loglogrus "dpchain/log_logrus"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"sync"

	"github.com/astaxie/beego/validation"
	"github.com/gin-gonic/gin"
)

var messages []string
var mu sync.Mutex

type VerifiedCredential struct {
	UeDID       string `json:"ueDID"`
	SpDID       string `json:"spDID"`
	UeSignature string `json:"ueSignature"`
	Lifetime    uint8  `json:"lifetime"`
	Role        string `json:"role"` // User or Supervisor
}

func BackAccountList(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- BackAccountList\n")
		appG := app.Gin{c}
		data, err := ds.BackListAccounts()
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- BackAccountList  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, data)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- BackAccountList  succeed!\n")
	}
}

func CreateNewAccount(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- CreateNewAccount\n")
		appG := app.Gin{c}
		password := c.PostForm("password")

		commandStr := "createNewAccount"

		if password != "" {
			commandStr += (" -p " + password)
		}

		account, err := ds.CreateNewAccount(commandStr)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- CreateNewAccount  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, fmt.Sprintf("%x", account))
		}
		loglogrus.Log.Infof("Http: Reply to user request -- CreateNewAccount  succeed!\n")
	}

}

// 十六进制字符串格式的account
func UseAccount(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		account := c.PostForm("account")
		password := c.PostForm("password")
		valid := validation.Validation{}
		valid.Required(account, "account").Message("账户Hash不能为空")
		if valid.HasErrors() {
			app.MarkErrors("", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- UseAccount  account:%s\n", account)

		commandStr := "useAccount " + account

		if password != "" {
			commandStr += (" -p " + password)
		}

		err := ds.UseAccount(commandStr)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- UseAccount  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, nil)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- UseAccount  succeed!\n")
	}

}

func CurrentAccount(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- CurrentAccount\n")
		appG := app.Gin{c}
		infoStr := ds.CurrentAccount()

		appG.Response(http.StatusOK, e.SUCCESS, infoStr)
		loglogrus.Log.Infof("Http: Reply to user request -- CurrentAccount  succeed!\n")
	}

}

// on:开启  off:关闭
func OpenTxCheckMode(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		mode := c.PostForm("mode")
		valid := validation.Validation{}
		valid.Required(mode, "commandStr").Message("on:开启交易上链检查 off:关闭交易上链检查")
		if valid.HasErrors() {
			app.MarkErrors("OpenTxCheckMode", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- OpenTxCheckMode  mode:%s\n", mode)
		commandStr := "txCheckMode " + mode

		err := ds.OpenTxCheckMode(commandStr)
		if err != nil {
			loglogrus.Log.Infof("Http: Reply to user request -- OpenTxCheckMode  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, nil)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- OpenTxCheckMode  succeed!\n")
	}

}

// contractAddr:合约地址
// functionAddr:参数地址
// args: 参数列表,以空格分割
func SolidInvoke(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		contractAddr := c.PostForm("contractAddr")
		functionAddr := c.PostForm("functionAddr")
		args := c.PostForm("args")
		valid := validation.Validation{}
		valid.Required(contractAddr, "contractAddr").Message("合约地址不能为空")
		valid.Required(functionAddr, "functionAddr").Message("函数地址不能为空")
		valid.Required(args, "args").Message("参数不能为空")
		if valid.HasErrors() {
			app.MarkErrors("SolidInvoke", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- SolidInvoke  contractAddr:%s , functionAddr:%s , args:%s\n", contractAddr, functionAddr, args)
		commandStr := "solidInvoke " + contractAddr + " " + functionAddr + " -args " + args

		receipt, err := ds.SolidInvoke(commandStr)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- SolidInvoke  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {

			if reflect.DeepEqual(receipt, transactionCheck.CheckResult{}) {
				appG.Response(http.StatusOK, e.SUCCESS, nil)
			} else {
				type ResponseReceipt struct {
					TxID   string `json:"Transaction ID"`
					Valid  bool   `json:"Valid"`
					Result string `json:"Transaction Results"`
					Delay  string `json:"Consensus Delay"`
				}
				var r ResponseReceipt = ResponseReceipt{
					TxID:   fmt.Sprintf("%x", receipt.TransactionID),
					Valid:  receipt.Valid,
					Result: fmt.Sprintf("%s", receipt.Result),
					Delay:  fmt.Sprintf("%d ms", receipt.Interval),
				}
				appG.Response(http.StatusOK, e.SUCCESS, r)
			}

		}
		loglogrus.Log.Infof("Http: Reply to user request -- SolidInvoke  succeed!\n")
	}

}

// contractAddr:合约地址
// functionAddr:参数地址
// args: 参数列表,以空格分割
func SolidCall(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		contractAddr := c.PostForm("contractAddr")
		functionAddr := c.PostForm("functionAddr")
		args := c.PostForm("args")
		valid := validation.Validation{}
		valid.Required(contractAddr, "contractAddr").Message("合约地址不能为空")
		valid.Required(functionAddr, "functionAddr").Message("函数地址不能为空")
		valid.Required(args, "args").Message("参数不能为空")

		if valid.HasErrors() {
			app.MarkErrors("SolidCall", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- SolidCall  contractAddr:%s , functionAddr:%s , args:%s\n", contractAddr, functionAddr, args)
		commandStr := "solidCall " + contractAddr + " " + functionAddr + " -args " + args

		data, err := ds.SolidCall(commandStr)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- SolidCall  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, data)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- SolidCall  succeed!\n")
	}

}

// contractName:合约名
// functionName:参数名
// args: 参数列表,以空格分割
func SoftInvoke(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		contractName := c.PostForm("contractName")
		functionName := c.PostForm("functionName")
		args := c.PostForm("args")
		valid := validation.Validation{}
		valid.Required(contractName, "contractName").Message("合约名不能为空")
		valid.Required(functionName, "functionName").Message("函数名不能为空")
		valid.Required(args, "args").Message("参数不能为空")

		if valid.HasErrors() {
			app.MarkErrors("SoftInvoke", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}

		loglogrus.Log.Infof("Http: Get user request -- SoftInvoke  contractName:%s , functionName:%s , args:%s\n", contractName, functionName, args)
		fmt.Printf("Http: Get user request -- SoftInvoke  contractName:%s , functionName:%s , args:%s\n", contractName, functionName, args)

		commandStr := "invoke " + contractName + " " + functionName + " -args " + args

		receipt, err := ds.SoftInvoke(commandStr)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- SoftInvoke  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			if reflect.DeepEqual(receipt, transactionCheck.CheckResult{}) {
				appG.Response(http.StatusOK, e.SUCCESS, nil)
			} else {
				type ResponseReceipt struct {
					TxID   string `json:"Transaction ID"`
					Valid  bool   `json:"Valid"`
					Result string `json:"Transaction Results"`
					Delay  string `json:"Consensus Delay"`
				}
				var r ResponseReceipt = ResponseReceipt{
					TxID:   fmt.Sprintf("%x", receipt.TransactionID),
					Valid:  receipt.Valid,
					Result: fmt.Sprintf("%s", receipt.Result),
					Delay:  fmt.Sprintf("%d ms", receipt.Interval.Milliseconds()),
				}
				appG.Response(http.StatusOK, e.SUCCESS, r)
			}
		}
		loglogrus.Log.Infof("Http: Reply to user request -- SoftInvoke  succeed!\n")
		fmt.Printf("Http: Reply to user request -- SoftInvoke  succeed!\n")
	}

}

func SoftInvokeQuery(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		contractName := c.PostForm("contractName")
		functionName := c.PostForm("functionName")
		args := c.PostForm("args")
		valid := validation.Validation{}
		valid.Required(contractName, "contractName").Message("合约名不能为空")
		valid.Required(functionName, "functionName").Message("函数名不能为空")
		valid.Required(args, "args").Message("参数不能为空")

		if valid.HasErrors() {
			app.MarkErrors("SoftInvoke", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- SoftInvokeQuery contractName:%s , functionName:%s , args:%s\n", contractName, functionName, args)
		fmt.Printf("Http: Get user request -- SoftInvokeQuery contractName:%s , functionName:%s , args:%s\n", contractName, functionName, args)

		commandStr := "invoke " + contractName + " " + functionName + " -args " + args

		var wg sync.WaitGroup
		ds.SoftInvokeQuery(commandStr, wg)
		fmt.Printf("Http: Reply to user request -- SoftInvokeQuery  succeed!\n")
	}
}

func PublishTx(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		// 1.解析 []tx 编码得到的byte流
		appG := app.Gin{c}
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			loglogrus.Log.Warnf("Http: PublishTx函数无法成功读取http请求,err:%v\n", err)
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		} else {
			loglogrus.Log.Warnf("Http: PublishTx函数成功读取http请求,body比特流长度:%v\n", len(body))
		}
		// Parse函数
		txList, err := consensus.DeserializeTransactionsSeries(body)
		if err != nil {
			loglogrus.Log.Warnf("Http: PublishTx函数无法解析http请求内容,err:%v\n", err)
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		} else {
			loglogrus.Log.Warnf("Http: PublishTx函数解析获得的交易数量:%v\n", len(txList))
		}

		// 2. 直接上传给consensusPromoter
		txListPtr := make([]*eles.Transaction, 0)
		for _, tx := range txList {
			txPtr := new(eles.Transaction)
			copyTransaction(tx, txPtr)
			txListPtr = append(txListPtr, txPtr)
		}

		err = ds.PublishTx(txListPtr)
		if err != nil {
			loglogrus.Log.Warnf("Http: PublishTx函数无法上传交易,err:%v\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		}
	}
}

func copyTransaction(src eles.Transaction, dst *eles.Transaction) {
	dst.TxID = src.TxID
	dst.Sender = src.Sender
	dst.Nonce = src.Nonce
	dst.Version = src.Version
	dst.LifeTime = src.LifeTime
	dst.Signature = append(dst.Signature, src.Signature...)
	dst.Contract = src.Contract
	dst.Function = src.Function
	dst.Args = append(dst.Args, src.Args...)
	dst.CheckList = append(dst.CheckList, src.CheckList...)
}

// contractName:合约名
// functionName:参数名
// args: 参数列表,以空格分割
func SoftCall(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		contractName := c.PostForm("contractName")
		functionName := c.PostForm("functionName")
		args := c.PostForm("args")
		valid := validation.Validation{}
		valid.Required(contractName, "contractName").Message("合约名不能为空")
		valid.Required(functionName, "functionName").Message("函数名不能为空")
		valid.Required(args, "args").Message("参数不能为空")

		if valid.HasErrors() {
			app.MarkErrors("SoftCall", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- SoftCall  contractName:%s , functionName:%s , args:%s\n", contractName, functionName, args)

		commandStr := "call " + contractName + " " + functionName + " -args " + args

		data, err := ds.SoftCall(commandStr)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- SoftCall  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, data)
		}

		loglogrus.Log.Infof("Http: Reply to user request -- SoftCall  succeed!\n")
	}

}

func BecomeBooter(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- BecomeBooter")
		appG := app.Gin{c}
		err := ds.BecomeBooter()
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- BecomeBooter  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, nil)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- BecomeBooter  succeed!\n")
	}

}

func BackViewNet(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- BackViewNet")
		appG := app.Gin{c}
		data, err := ds.BackViewNet()
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- BackViewNet  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, data)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- BackViewNet  succeed!\n")
	}

}

func HelpMenu(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		data := ds.HelpMenu()
		appG.Response(http.StatusOK, e.SUCCESS, data)
	}
}

func Exit(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- Exit\n")
		appG := app.Gin{c}
		appG.Response(http.StatusOK, e.SUCCESS, "ByeBye")
		defer ds.Exit()
	}
}

func VcReturn(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		lifetime := c.PostForm("lifetime")
		role := c.PostForm("role")
		signature, err := ds.SignMessage("fakemessage")
		uedid := "DID:1231231231231"
		spdid := "DID:4654654654656"

		num, _ := strconv.Atoi(lifetime)
		vc := VerifiedCredential{
			UeDID:       uedid,
			SpDID:       spdid,
			UeSignature: signature,
			Lifetime:    uint8(num), // Replace with the actual value
			Role:        role,
		}
		responseData := map[string]interface{}{
			"signature": signature,
			"lifetime":  lifetime,
			"role":      role,
			"vc":        vc,
		}
		if err != nil {
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, responseData)
		}

	}
}

func Send(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		// 获取目标端口和消息内容
		destinationPort := c.Param("destinationPort")
		message := c.PostForm("message")

		// 构造目标 URL
		destinationURL := fmt.Sprintf("http://127.0.0.1:%s/dper/receive", destinationPort)

		// 发送 HTTP POST 请求
		resp, err := http.PostForm(destinationURL, url.Values{"message": {message}})
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}

		defer resp.Body.Close()

		// 读取响应
		responseBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}

		// 返回响应给调用方
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Message sent to port %s, response: %s", destinationPort, string(responseBody)),
		})
	}
}
func SendRandom(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		// 获取目标端口和消息内容
		destinationPort := c.Param("destinationPort")

		buff := make([]byte, 10) // 创建一个长度为10的缓冲区
		_, err := randentropy.Reader.Read(buff)
		if err != nil {
			panic(err)
		}

		// 打印生成的随机字节
		fmt.Printf("Generated Random Bytes: %s", hex.EncodeToString(buff))

		message := hex.EncodeToString(buff)

		// 构造目标 URL
		destinationURL := fmt.Sprintf("http://127.0.0.1:%s/dper/signaturereturn", destinationPort)

		// 发送 HTTP POST 请求
		resp, err := http.PostForm(destinationURL, url.Values{"message": {message}})
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}

		defer resp.Body.Close()

		// 读取响应
		responseBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}

		// 返回响应给调用方
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Message: %s,response: %s", message, string(responseBody)),
		})
	}
}

func SignatureReturn(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		// 解析消息内容
		message := c.PostForm("message")
		// 获取签名和事务ID
		signature, err := ds.SignMessage(message)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status": "error",
				"error":  err.Error(),
			})
			return
		} else {
			c.JSON(http.StatusOK, gin.H{
				"status":    "success",
				"signature": fmt.Sprintf("%s", signature),
			})
		}

	}
}
func SignValid(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		// 解析消息内容
		message := c.PostForm("message")
		signature := c.PostForm("signature")
		//address相当于pk
		address := c.PostForm("address")
		// 获取签名和事务ID
		msg := crypto.Sha3Hash([]byte(message))
		sig := common.Hex2Bytes(signature)
		add := common.HexToAddress(address)
		ok, err := crypto.SignatureValid(add, sig, msg)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status": "error",
				"error":  err.Error(),
			})
			return
		} else {
			if ok == true {
				c.JSON(http.StatusOK, gin.H{
					"status": "success",
					"valid":  "The signature is valid!",
				})
			} else if ok == false {
				c.JSON(http.StatusOK, gin.H{
					"status": "success",
					"valid":  "The signature is not valid!",
				})
			}
		}

	}

}


// func Receive(ds *api.DperService) func(*gin.Context) {
// 	return func(c *gin.Context) {
// 		destinationPort := c.Param("destinationPort")

// 		// 解析消息内容
// 		message := c.PostForm("message")

// 		// 获取签名和事务ID
// 		signature, TxID, err := ds.SignatureReturn(message)
// 		if err != nil {
// 			c.JSON(http.StatusOK, gin.H{
// 				"status": "error",
// 				"error":  err.Error(),
// 			})
// 			return
// 		} else {
// 			c.JSON(http.StatusOK, gin.H{
// 				"status":    "success",
// 				"message":   message,
// 				"TxID":      fmt.Sprintf("%x", TxID),
// 				"signature": fmt.Sprintf("%x", signature),
// 			})
// 		}

// 		// 打印收到的消息和目标端口
// 		fmt.Printf("Received message: %s, destination port: %s\n", message, destinationPort)
// 	}
// }
