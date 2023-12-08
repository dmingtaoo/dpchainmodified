package router

import (
	"dpchain/api"
	"dpchain/common"
	"dpchain/core/consensus"
	"dpchain/core/eles"
	"dpchain/crypto"

	// "dpchain/crypto/randentropy"
	"dpchain/dper/transactionCheck"
	"dpchain/ginHttp/pkg/app"
	e "dpchain/ginHttp/pkg/error"
	loglogrus "dpchain/log_logrus"

	// "encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"sync"

	"github.com/astaxie/beego/validation"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var messages []string
var mu sync.Mutex

type VCrequest struct {
	Identifier     string `json:"identifier"` // VC ID
	Subject        string `json:"subject"`    // SP DID
	Issuer         string `json:"issuer"`     // UE DID
	Validity       string `json:"validity"`   // Valid time
	Purpose        string `json:"purpose"`    // 用途
	SubjectAddress string `json:"address"`    // subject address
}

type VC struct {
	Identifier     string `json:"identifier"` // VC ID
	Subject        string `json:"subject"`    // SP DID
	Issuer         string `json:"issuer"`     // UE DID
	Validity       string `json:"validity"`   // Valid time
	Purpose        string `json:"purpose"`    // 用途
	Signature      string `json:"signature"`  // UE Signature
	Reassign       string `json:"reassign"`   // 是否允许二次转让
	IssuerAddress  string `json:"address"`    // issuer address
	SubjectAddress string `json:"address"`    // subject address
}
type DataStruct struct {
	DID    string `json:"did"`
	Name   string `json:"name"`
	Gender string `json:"gender"`
	Age    string `json:"age"`
}

// FindDataInFile 从指定的 JSON 文件中查找特定的 DID 并返回对应的数据
func FindDataInFile(filename, did string) (string, error) {
	// 从文件中加载数据
	var data []DataStruct
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("error reading file: %v", err)
	}
	err = json.Unmarshal(file, &data)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling data: %v", err)
	}

	// 查找特定的 DID
	for _, d := range data {
		if d.DID == did {
			// 将找到的数据转换为 JSON 字符串
			result, err := json.Marshal(d)
			if err != nil {
				return "", fmt.Errorf("error marshalling data: %v", err)
			}
			return string(result), nil
		}
	}
	return "", fmt.Errorf("DID not found")
}
func ResultToSubjectAddress(byteSlices [][]byte) (string, error) {
	var resultString string
	for _, b := range byteSlices {
		resultString += string(b)
	}
	// 解析 JSON 字符串
	var data map[string]interface{}

	err := json.Unmarshal([]byte(resultString), &data)
	if err != nil {
		return "failed", err
	}
	// 提取 SubjectAddress
	subjectAddress, ok := data["SubjectAddress"].(string)
	if !ok {
		return "subjectaddress not found", nil
	}
	resultString = subjectAddress

	return resultString, nil
}
func ResultToIssuer(byteSlices [][]byte) (string, error) {
	var resultString string
	for _, b := range byteSlices {
		resultString += string(b)
	}
	// 解析 JSON 字符串
	var data map[string]interface{}

	err := json.Unmarshal([]byte(resultString), &data)
	if err != nil {
		return "failed", err
	}
	// 提取 SubjectAddress
	Issuer, ok := data["Issuer"].(string)
	if !ok {
		return "Issuer not found", nil
	}
	resultString = Issuer

	return resultString, nil
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

func SendVCrequest(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		destinationPort := c.Param("destinationPort")

		// 从请求中获取数据
		Issuer := c.PostForm("Issuer")
		Subject := c.PostForm("Subject")
		Validity := c.PostForm("Validity")
		Purpose := c.PostForm("Purpose")
		SubjectAddress := c.PostForm("SubjectAddress")

		// 构造VCrequest
		vcrequest := VCrequest{
			Identifier:     uuid.New().String(),
			Subject:        Subject,
			Issuer:         Issuer,
			Validity:       Validity,
			Purpose:        Purpose,
			SubjectAddress: SubjectAddress,
		}
		//VC转为字符串
		vcrequestString := fmt.Sprintf("Identifier:%s,Subject:%s,Issuer:%s,Validity:%s,Purpose:%s,SubjectAddress:%s",
			vcrequest.Identifier, vcrequest.Subject, vcrequest.Issuer, vcrequest.Validity, vcrequest.Purpose, vcrequest.SubjectAddress)
		// 构造目标 URL
		destinationURL := fmt.Sprintf("http://127.0.0.1:%s/dper/vcreceive", destinationPort)

		// 准备要发送的数据
		formData := url.Values{
			"vcrequest": {vcrequestString},
		}
		// 发送 HTTP POST 请求
		resp, err := http.PostForm(destinationURL, formData)
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
			"response": string(responseBody),
		})
	}
}

func VCReceive(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}

		// 从请求中获取VC字符串
		vcrequestString := c.PostForm("vcrequest")
		if vcrequestString == "" {
			appG.Response(http.StatusBadRequest, e.ERROR, "No VCrequest data received")
			return
		}

		// 打印接收到的VC字符串
		fmt.Printf("Received VC String: %s\n", vcrequestString)

		// 返回响应给调用方
		appG.Response(http.StatusOK, e.SUCCESS, fmt.Sprintf("Received VCrequest: %s", vcrequestString))
	}
}

func SendVC(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		destinationPort := c.Param("destinationPort")

		// 从请求中获取数据
		Identifier := c.PostForm("Identifier")
		Issuer := c.PostForm("Issuer")
		Subject := c.PostForm("Subject")
		Validity := c.PostForm("Validity")
		Purpose := c.PostForm("Purpose")
		Signature := c.PostForm("Signature")
		Reassign := c.PostForm("Reassign")
		IssuerAddress := c.PostForm("IssuerAddress")
		SubjectAddress := c.PostForm("SubjectAddress")

		// 构造VC
		vc := VC{
			Identifier:     Identifier,
			Subject:        Subject,
			Issuer:         Issuer,
			Validity:       Validity,
			Purpose:        Purpose,
			Signature:      Signature,
			Reassign:       Reassign,
			IssuerAddress:  IssuerAddress,
			SubjectAddress: SubjectAddress,
		}
		//VC转为字符串
		vcString := fmt.Sprintf("Identifier:%s,Subject:%s,Issuer:%s,Validity:%s,Purpose:%s,Signature:%s,Reassign:%s,IssuerAddress:%s,SubjectAddress:%s",
			vc.Identifier, vc.Subject, vc.Issuer, vc.Validity, vc.Purpose, vc.Signature, vc.Reassign, vc.IssuerAddress, vc.SubjectAddress)
		// 构造目标 URL
		destinationURL := fmt.Sprintf("http://127.0.0.1:%s/dper/vcvalid", destinationPort)

		// 准备要发送的数据
		formData := url.Values{
			"vc": {vcString},
		}
		// 发送 HTTP POST 请求
		resp, err := http.PostForm(destinationURL, formData)
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
			"response": string(responseBody),
		})
	}
}

func VCValid(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}

		// 从请求中获取VC字符串
		vcString := c.PostForm("vc")
		if vcString == "" {
			appG.Response(http.StatusBadRequest, e.ERROR, "No VC data received")
			return
		}
		//调用合约
		contractName := "DID::SPECTRUM::TRADE"
		functionName := "VCValid"
		args := vcString
		commandStr := "invoke " + contractName + " " + functionName + " -args " + args

		receipt, err := ds.SoftInvoke(commandStr)
		if err != nil {
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
	}
}

func SendRandom(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		// 获取目标端口和消息内容
		destinationPort := c.Param("destinationPort")

		// buff := make([]byte, 10) // 创建一个长度为10的缓冲区
		// _, err := randentropy.Reader.Read(buff)
		// if err != nil {
		// 	panic(err)
		// }
		// message := hex.EncodeToString(buff)
		message := c.PostForm("message")
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

func DataRequest(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		// 获取目标端口和消息内容
		destinationPort := c.Param("destinationPort")
		ID := c.PostForm("ID")

		// 签名消息
		signature, err := ds.SignMessage(ID)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}
		// 构造目标 URL
		destinationURL := fmt.Sprintf("http://127.0.0.1:%s/dper/datasend", destinationPort)

		// 发送 HTTP POST 请求
		resp, err := http.PostForm(destinationURL, url.Values{
			"ID":        {ID},
			"signature": {signature},
		})
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
			"message": fmt.Sprintf("ID: %s, response: %s", ID, string(responseBody)),
		})
	}
}
func DataSend(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		ID := c.PostForm("ID")
		signature := c.PostForm("signature")

		// 合约调用逻辑...
		contractName := "DID::SPECTRUM::TRADE"
		functionName := "GetVC"
		args := ID
		commandStr := "invoke " + contractName + " " + functionName + " -args " + args
		receipt, err := ds.SoftInvoke(commandStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
			return
		}

		subjectAddress, _ := ResultToSubjectAddress(receipt.Result)
		issuer, _ := ResultToIssuer(receipt.Result)
		msg := crypto.Sha3Hash([]byte(ID))
		sig := common.Hex2Bytes(signature)
		add := common.HexToAddress(subjectAddress)
		ok, err := crypto.SignatureValid(add, sig, msg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
			return
		}

		if ok {
			data, err := FindDataInFile("data.json", issuer)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
				return
			}
			// 直接返回结果给调用方
			c.JSON(http.StatusOK, gin.H{
				"status": "success",
				"data":   data,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{"status": "error", "error": "Signature validation failed"})
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
// func SendVC(ds *api.DperService) func(*gin.Context) {
// 	return func(c *gin.Context) {
// 		appG := app.Gin{c}

// 		// 从请求中获取数据
// 		Issuer := c.PostForm("Issuer")
// 		Subject := c.PostForm("Subject")
// 		validity := c.PostForm("validity")

// 		// 签名消息
// 		signature, err := ds.SignMessage("GETDATA")
// 		if err != nil {
// 			appG.Response(http.StatusInternalServerError, e.ERROR, err)
// 			return
// 		}

// 		// 构造VC
// 		vc := Credential{
// 			Identifier: "hahaha",
// 			Subject:    Subject,
// 			Issuer:     Issuer,
// 			Validity:   validity,
// 			Message:    "GETDATA",
// 			Signature:  signature,

// 		}

// 		// 将VC转换为字符串
// 		vcString := fmt.Sprintf("Identifier:%s,Subject:%s,Issuer:%s,Validity:%s,Signature:%s,Message:%s",
// 			vc.Identifier, vc.Subject, vc.Issuer, vc.Validity, vc.Signature, vc.Message)

// 		// 返回响应给调用方
//         c.JSON(http.StatusOK, gin.H{
//             "response": vcString, // 或者是处理结果
//         })
// 	}
// }
// func DataSendtest(ds *api.DperService) func(*gin.Context) {
// 	return func(c *gin.Context) {
// 		// 解析消息内容
// 		appG := app.Gin{c}
// 		// 获取目标端口和消息内容
// 		ID := c.PostForm("ID")
// 		//合约调用
// 		contractName := "DID::SPECTRUM::TRADE"
// 		functionName := "GetVC"
// 		args := ID
// 		commandStr := "invoke " + contractName + " " + functionName + " -args " + args
// 		receipt, err := ds.SoftInvoke(commandStr)
// 		if err != nil {
// 			fmt.Println("Error parsing JSON:", err)
// 			return
// 		}
// 		// 将结果字节切片转换为字符串切片，然后合并为单个字符串
// 		var resultString string

// 		// 返回 SubjectAddress
// 		appG.Response(http.StatusOK, e.SUCCESS, subjectAddress)
// 	}
// }
