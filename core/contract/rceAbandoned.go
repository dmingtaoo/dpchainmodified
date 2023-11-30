package contract


// this file is unused!!!!!!!









// import (
// 	"dpchain/common"
// 	"dpchain/core/eles"
// 	"dpchain/core/worldstate"
// 	"dpchain/logger"
// 	"dpchain/logger/glog"
// 	"dpchain/rlp"
// 	"fmt"
// 	"net"
// 	"net/rpc"
// 	"sync"
// )

// type RemoteContractEngine struct {
// 	useMux sync.Mutex

// 	stateManager worldstate.StateManager
// 	valueBox     *sandBox
// 	remote_ip    string
// 	checkIpt     CheckElementInterpreter
// }


// // create a new remote contract engine using the base function
// // compiler.
// func NewRemoteCE(sm worldstate.StateManager, ci CheckElementInterpreter, ip string) *RemoteContractEngine {
// 	sb := &sandBox{
// 		readSet:  make(map[common.WsKey]worldStateValueType),
// 		writeSet: make(map[common.WsKey]worldStateValueType),
// 	}
// 	ce := &RemoteContractEngine{
// 		stateManager: sm,
// 		valueBox:     sb,
// 		checkIpt:     ci,
// 		remote_ip:    ip,
// 	}
// 	return ce
// }

// func (rce *RemoteContractEngine) ExecuteTransactions(txs []eles.Transaction) ([]eles.TransactionReceipt, []eles.WriteEle) {
// 	rce.useMux.Lock()
// 	defer rce.useMux.Unlock()

// 	rce.valueBox.empty()
// 	receipts := make([]eles.TransactionReceipt, 0)

// 	currentVersion := rce.stateManager.CurrentVersion()
// 	client, _ := rpc.Dial("tcp", rce.remote_ip)
// 	for _, tx := range txs {
// 		if rce.stateManager.StaleCheck(tx.Version, currentVersion, tx.LifeTime) {
// 			receipts = append(receipts, eles.CreateInvalidTransactionReceipt())
// 			continue
// 		}
// 		if !rce.ValidateCheckElements(tx.CheckList) {
// 			receipts = append(receipts, eles.CreateInvalidTransactionReceipt())
// 			continue
// 		}
// 		funcWsKey := common.AddressesCatenateWsKey(tx.Contract, tx.Function)
// 		funcName, err := rce.Get(funcWsKey)
// 		if err != nil {
// 			fmt.Print("Here = ", err)
// 			receipts = append(receipts, eles.CreateInvalidTransactionReceipt())
// 			continue
// 		}
// 		contractName, err := rce.Get(common.AddressesCatenateWsKey(tx.Contract, common.Address{}))
// 		if err != nil {
// 			receipts = append(receipts, eles.CreateInvalidTransactionReceipt())
// 			continue
// 		}
// 		tmp := &Function_invoke_type{
// 			Args:         tx.Args,
// 			ContractName: contractName.toBytes(),
// 			FunctionName: funcName.toBytes(),
// 			Self:         tx.Sender,
// 		}
// 		request, _ := rlp.EncodeToBytes(tmp)
// 		var reply string

// 		err = client.Call("ChainCodeEngine.Call_func", string(request), &reply)

// 		reply_data := new(Function_return_type)
// 		rlp.DecodeBytes([]byte(reply), reply_data)

// 		if reply_data.Err != nil || err != nil {
// 			glog.V(logger.Warn).Infof("wskey: %x, fail in execution", funcWsKey)
// 			receipts = append(receipts, eles.CreateInvalidTransactionReceipt())
// 			continue
// 		}
// 		receipts = append(receipts, eles.CreateValidTransactionReceipt(reply_data.Result))
// 	}
// 	writeSet := rce.valueBox.exportWriteSet()
// 	return receipts, writeSet
// }

// func (pce *RemoteContractEngine) CommitWriteSet(writeSet []eles.WriteEle) error {
// 	pce.useMux.Lock()
// 	defer pce.useMux.Unlock()

// 	keys := make([]common.WsKey, len(writeSet))
// 	values := make([][]byte, len(writeSet))
// 	for i := 0; i < len(writeSet); i++ {
// 		keys[i] = writeSet[i].ValueAddress
// 		values[i] = writeSet[i].Value
// 	}
// 	err := pce.stateManager.WriteWorldStateValues(keys, values)
// 	return err
// }

// func (pce *RemoteContractEngine) ValidateCheckElements(checkEle []eles.CheckElement) bool {
// 	for i := 0; i < len(checkEle); i++ {
// 		if !pce.checkIpt.validate_re(pce, checkEle[i]) {
// 			return false
// 		}
// 	}
// 	return true
// }
// func (pce *RemoteContractEngine) GetData(wsKey common.WsKey) (WorldStateValueType, error) {
// 	if val, ok := pce.valueBox.get(wsKey); ok {
// 		tmp := CommonBytes{
// 			Value: val.toBytes(),
// 		}
// 		return &tmp, nil
// 	}
// 	return nil, fmt.Errorf("unknown domain address")
// }
// func (rce *RemoteContractEngine) Get(wsKey common.WsKey) (worldStateValueType, error) {
// 	if val, ok := rce.valueBox.get(wsKey); ok {
// 		return val, nil
// 	}
// func (pce *RemoteContractEngine) Set(wsKey common.WsKey, value worldStateValueType) {
// 	pce.valueBox.set(wsKey, value)
// }
// 	plainTexts, err := rce.stateManager.ReadWorldStateValues(wsKey)
// 	if err != nil {

// 		return nil, err
// 	}
// 	plainText := plainTexts[0]
// 	domainAddr, innerAddr := common.WsKeySplitToAddresses(wsKey)
// 	switch domainAddr[0] {
// 	case domain_ContractCode:
// 		if innerAddr == domainInfoAddress {
// 			value := &domainInfo{
// 				plainValue: plainText,
// 			}
// 			rce.valueBox.add(wsKey, value)
// 			return value, nil
// 		}
// 		switch innerAddr[0] {
// 		case contract_FunctionCode:
// 			value := &commonBytes{
// 				value: plainText,
// 			}
// 			rce.valueBox.add(wsKey, value)
// 			return value, nil
// 		case contract_BytesCode:
// 			value := &commonBytes{
// 				value: plainText,
// 			}
// 			rce.valueBox.add(wsKey, value)
// 			return value, nil
// 		default:
// 			return nil, fmt.Errorf("unknown inner address of a contract")
// 		}

// 	case domain_StorageCode:
// 		if innerAddr == domainInfoAddress {
// 			value := &domainInfo{
// 				plainValue: plainText,
// 			}
// 			rce.valueBox.add(wsKey, value)
// 			return value, nil
// 		}

// 		value := &commonBytes{
// 			value: plainText,
// 		}
// 		rce.valueBox.add(wsKey, value)
// 		return value, nil
// 	default:
// 		return nil, fmt.Errorf("unknown domain address")
// 	}
// }


// type RpcChainCodeAPI struct {
// 	rce *RemoteContractEngine
// }

// //

// func (r *RpcChainCodeAPI) Get(request string, reply *string) error {
// 	tmp := []byte(request)
// 	var wsKey common.WsKey
// 	copy(wsKey[:], tmp[:])

// 	val, ok := r.rce.Get(wsKey)
// 	var re GET_return_type

// 	if ok == nil {
// 		re.Code = GET_OK
// 		re.Payload = string(val.toBytes())
// 	} else {
// 		re.Code = GET_ERROR
// 		re.Payload = string(ok.Error())
// 	}
// 	tmp, _ = rlp.EncodeToBytes(re)
// 	*reply = string(tmp)
// 	return nil
// }

// func (r *RpcChainCodeAPI) Set(request string, reply *string) error {
// 	tmp := []byte(request)
// 	set_args := new(SET_args)
// 	rlp.DecodeBytes(tmp, set_args)
// 	value := &commonBytes{
// 		value: set_args.Value,
// 	}
// 	var wsKey common.WsKey
// 	copy(wsKey[:], set_args.WsKey)
// 	r.rce.valueBox.set(wsKey, value)
// 	*reply = "ok"
// 	return nil
// }
// func (r *RpcChainCodeAPI) SetList(request string, reply *string) error {
// 	tmp := []byte(request)
// 	set_args := new(SET_LIST_args)
// 	rlp.DecodeBytes(tmp, set_args)
// 	for key,val := range set_args.WsKey{
// 		value := &commonBytes{
// 			value: set_args.Value[key],
// 		}
// 		var wsKey common.WsKey
// 		copy(wsKey[:], val)
// 		r.rce.valueBox.set(wsKey, value)
// 	}
// 	*reply = "ok"
// 	return nil
// }
// func (r *RpcChainCodeAPI) InstallContract(request string, reply *string) error {
// 	tmp := []byte(request)
// 	install_args := new(Contract_install_type)
// 	rlp.DecodeBytes(tmp, install_args)
// 	err := r.rce.stateManager.WriteWorldStateValues(install_args.InstallKeys, install_args.InstallValues)
// 	if err != nil {
// 		*reply = "install failed"
// 	}
// 	*reply = "install succeed"
// 	fmt.Println(*reply)
// 	return nil
// }

// func RpcChainCodeSupport(rce *RemoteContractEngine, netMethod string, ip string) error {
// 	RP := new(RpcChainCodeAPI)
// 	RP.rce = rce
// 	rpc.RegisterName("RpcChainCodeAPI"+ip, RP)
// 	listener, err := net.Listen(netMethod, ip)
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Println("RpcChainCodeSupport engine start!! ",ip )
// 	for {
// 		conn, err := listener.Accept()
// 		if err != nil {
// 			return err
// 		}
// 		go rpc.ServeConn(conn)
// 	}

// }