package dper

import (
	"crypto/ecdsa"
	"dpchain/accounts"
	"dpchain/blockSync"
	"dpchain/common"
	"dpchain/core/blockOrdering"
	"dpchain/core/consensus"
	"dpchain/core/contract"
	"dpchain/core/dpnet"
	"dpchain/core/eles"
	"dpchain/core/netconfig"
	"dpchain/core/separator"
	"dpchain/core/validator"
	"dpchain/core/worldstate"
	"dpchain/crypto"
	"dpchain/crypto/keys"
	"dpchain/crypto/randentropy"
	"dpchain/database"
	"dpchain/dper/transactionCheck"
	loglogrus "dpchain/log_logrus"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/go-ini/ini"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"

	"dpchain/p2p/discover"
	"dpchain/p2p/nat"
	"dpchain/p2p/server"
	"dpchain/p2p/whisper"
)

const (
	BlockDatabaseName      = "BlockDB"
	WorldStateDatabaseName = "WSDB"

	BenchMarkFetch = 1000

	// archiveScanInterval = 10 * time.Second // 归档协程扫描archiveThreshold间隔
	// archiveThreshold    = 60               // 本地存储区块数量达到该阈值后，触发归档(必须要大于50 --- 满足交易过滤器的区块数量需求) 适当配置，不能太低，否则区块同步将失效
)

var (
	archiveMode bool = false
)

var (
	INSTALL_TPBFT       = []string{"CTPBFT"}
	REPEAT_CHECK_ENABLE = false
)

// 区块归档服务
type DataAchive struct {
	ArchiveMode         bool
	ArchiveScanInterval time.Duration
	ArchiveThreshold    int
}

var DataAchiveSetting = &DataAchive{}

// Dper is the actual implementation of DP-Chain. It contains all the databases, managers, networks and so on.
type Dper struct {
	selfKey        keys.Key
	selfNode       *dpnet.Node
	blockDB        database.Database //本地维护一个区块数据库
	storageDB      database.Database //本地维护一个存储数据库
	accountManager *accounts.Manager

	netManager     *netconfig.NetManager //本地维护一个dp-chain网络
	p2pServer      *server.Server
	stateManager   worldstate.StateManager
	contractEngine contract.ContractEngine

	validateManager   *validator.ValidateManager
	consensusPromoter *consensus.ConsensusPromoter

	txCheckPool  *transactionCheck.TxCheckPool
	txCheckChan  chan common.Hash
	txCheckClose chan bool

	transactionFilter *eles.TransactionFilter
}

// Config configures the Dper.
type DperConfig struct {
	// basic information
	NewAddressMode bool //whether to create a new account when start
	DperKeyAddress common.Address
	DperPrivateKey *ecdsa.PrivateKey
	KeyStoreDir    string

	// connect configuration
	ServerName     string
	ListenAddr     string
	NAT            nat.Interface
	BootstrapNodes []*discover.Node //整个dp-chain中已知的引导节点
	MaxPeers       int

	// about database
	MemoryDataBaseMode  bool // whether to use memory mode
	BlockDataBasePath   string
	StorageDataBasePath string

	// about dpnet configuration
	Role              dpnet.RoleType // the role in subnet
	NetID             string         //所属的网络ID
	CentralConfigMode bool           // whether to use central protocol to config dpnet

	// about contract engine
	ContractEngine        string
	RemoteSupportPipeName string
	RemoteEnginePipeName  string
}

type DPNetwork struct {
	SubnetCount int `json:"subnetCount"`
	LeaderCount int `json:"leaderCount"`
	BooterCount int `json:"booterCount"`

	NetGroups []NetGroup `json:"netGroups"`
}

type NetGroup struct {
	NetID       string `json:"netID"`
	MemberCount int    `json:"memberCount"`
}

// 获取dper所属的网络的ID
func (dp *Dper) GetNetID() string {
	return dp.netManager.GetSelfNodeNetID()
}

// 获取depr在网络中的身份
func (dp *Dper) GetNodeRole() dpnet.RoleType {
	return dp.netManager.GetSelfNodeRole()
}

func (dp *Dper) BackStateManager() worldstate.StateManager {
	return dp.stateManager
}

func (dp *Dper) BackNetManager() *netconfig.NetManager {
	return dp.netManager
}

func (dp *Dper) CloseDiskDB() {
	dp.blockDB.Close()
	dp.storageDB.Close()
}

func saveAddress(path string, address common.Address) {
	byteValue, err := os.ReadFile(path) //读取json文件
	if err != nil {
		loglogrus.Log.Errorf("NewDper failed: Unable to open Dper configuration file at the specified path:%s ,err:%v\n", path, err)
		panic("NewDper failed: Unable to open Dper configuration file at the specified path!")
	}
	result := make(map[string]interface{}, 0)
	err = json.Unmarshal(byteValue, &result) //解析json k-v对
	if err != nil {
		loglogrus.Log.Errorf("NewDper failed: Unable to Unmarshal Dper configuration file, err:%v\n", err)
		panic("NewDper failed: Unable to Unmarshal Dper configuration file!")

	}
	result["DperKeyAddress"] = fmt.Sprintf("%x", address)

	temp, _ := json.MarshalIndent(result, "", "")
	err = os.WriteFile(path, temp, 0644)
	if err != nil {
		loglogrus.Log.Errorf("NewDper failed: Unable to modify Dper configuration file, err:%v\n", err)
		panic("NewDper failed: Unable to modify Dper configuration file!")
	}
}

// 初始化一个Dper对象
func NewDper(cfg *DperConfig) *Dper {
	err := os.Mkdir(cfg.KeyStoreDir, 0750)
	if err != nil && !os.IsExist(err) {
		loglogrus.Log.Errorf("NewDper failed: can't to create key store according to the given path:%s\n", cfg.KeyStoreDir)
		panic("NewDper failed: Failed to create key store according to the given path!")
	}
	// keyStore := keys.NewKeyStorePlain(cfg.KeyStoreDir) //创建一个key管理接口(负责key的增删改查)
	keyStore := keys.NewKeyStorePassphrase(cfg.KeyStoreDir) //创建一个key管理接口(负责key的增删改查)
	loglogrus.Log.Infof("NewDper: Initialize key manager...")
	accountManager := accounts.NewManager(keyStore) //创建一个账户管理对象
	var selfKey *keys.Key
	if cfg.NewAddressMode { //"NewAddressMode":"true"则生成新的Address   "NewAddressMode":"false"则读取"SelfNodeAddress"指定的Address
		var err error
		selfKey, err = keyStore.GenerateNewKey(randentropy.Reader, "")
		if err != nil {
			loglogrus.Log.Errorf("NewDper failed: can't to generate new key for current Dper ,err:%v\n", err)
			panic("NewDper failed: Generate New Key is failed!")
		}
		loglogrus.Log.Infof("NewDper: Complete generating key manager!")

	} else {
		var err error
		rootPath, _ := os.Getwd()

		dirPath := rootPath + string(os.PathSeparator) + strings.TrimLeft(cfg.KeyStoreDir, "./")
		files, err := ioutil.ReadDir(dirPath)
		if err != nil {
			loglogrus.Log.Errorf("NewDper failed: Unable to find the account directory of the current node in the specified path:%s\n", dirPath)
			panic("NewDper failed: Unable to reach the account directory of the current node!")
		}
		if len(files) == 0 {
			selfKey, err = keyStore.GenerateNewKey(randentropy.Reader, "")
			if err != nil {
				loglogrus.Log.Warnf("NewDper Warn: Can't generate new Key for current Dper,err:%v\n", err)
				selfKey, _ = keyStore.GenerateNewKey(randentropy.Reader, "") //TODO: 是否会出现反复创建不成功的情况？
			}
			jsonConfigPath := rootPath + string(os.PathSeparator) + "settings" + string(os.PathSeparator) + "dperConfig.json"
			saveAddress(jsonConfigPath, selfKey.Address)
			loglogrus.Log.Infof("NewDper: Welcome to Dper for the first time, the first account has been automatically generated for you!\n")

		} else {
			selfKey, err = keyStore.GetKey(cfg.DperKeyAddress, "")
			if err != nil {
				loglogrus.Log.Errorf("NewDper failed: The account you specified cannot be found in the account directory, err:%v\n", err)
				panic("NewDper failed: The specified account does not exist, login failed!")
			}
		}
		loglogrus.Log.Infof("NewDper: Complete loading key manager!\n")
	}

	publicKey := &selfKey.PrivateKey.PublicKey //从selfKey获取公钥

	selfNode := dpnet.NewNode(crypto.KeytoNodeID(publicKey), int(cfg.Role), cfg.NetID) //根据已知配置信息,生成dpnet.Node对象
	loglogrus.Log.Infof("NewDper failed: Complete creating dpnet.Node for self: NodeID: %x ,NetID: %s , Role: %v\n", selfNode.NodeID, selfNode.NetID, selfNode.Role)

	loglogrus.Log.Infof("NewDper: Initialize database...")
	var blockDB, storageDB database.Database
	if cfg.MemoryDataBaseMode {
		memDB1, err := leveldb.Open(storage.NewMemStorage(), nil)
		if err != nil {
			loglogrus.Log.Warnf("NewDper Warn: Can't Create New memDB for BlockChain,err:%v\n", err)
			memDB1, _ = leveldb.Open(storage.NewMemStorage(), nil) // TODO: 具体判断该错误的类型，尝试再次创建?
		}
		memDB2, err := leveldb.Open(storage.NewMemStorage(), nil)
		if err != nil {
			loglogrus.Log.Warnf("NewDper Warn: Can't Create New memDB for Storage,err:%v", err)
			memDB2, _ = leveldb.Open(storage.NewMemStorage(), nil) // TODO: 具体判断该错误的类型，尝试再次创建?
		}
		blockDB = database.NewSimpleLDB(BlockDatabaseName, memDB1)
		storageDB = database.NewSimpleLDB(WorldStateDatabaseName, memDB2)
	} else {
		DB1, err := database.NewLDBDatabase(cfg.BlockDataBasePath)
		if err != nil {
			loglogrus.Log.Errorf("NewDper failed: Can't Load Persistent data from Specified path:%s , err:%v\n", cfg.BlockDataBasePath, err)
			panic("NewDper failed: Can't Load Persistent data from Specified path")

		}
		DB2, err := database.NewLDBDatabase(cfg.StorageDataBasePath)
		if err != nil {
			loglogrus.Log.Errorf("NewDper failed: Can't Load Persistent data from Specified path:%s , err:%v\n", cfg.StorageDataBasePath, err)
			panic("NewDper failed: Can't Load Persistent data from Specified path")
		}
		blockDB = DB1
		storageDB = DB2
	}
	loglogrus.Log.Infof("NewDper: Complete loading database!\n")

	loglogrus.Log.Infof("NewDper: Initialize blockchain...\n")
	var transactionFilter *eles.TransactionFilter
	if REPEAT_CHECK_ENABLE {
		transactionFilter = eles.NewTransactionFilter()
	}

	TxCheckPool := transactionCheck.NewTaskPool() //创建一个交易检查事件池
	blockchain := eles.InitBlockChain(blockDB, TxCheckPool, transactionFilter, DataAchiveSetting.ArchiveMode)

	loglogrus.Log.Infof("NewDper: Complete initializing blockchain!\n")

	loglogrus.Log.Infof("NewDper: Initialize state manager...\n")
	stateManager := worldstate.NewDpStateManager(selfNode, storageDB, blockchain)
	loglogrus.Log.Infof("NewDper: Complete initializing state manager\n")

	loglogrus.Log.Infof("NewDper: Launch contract engine...\n")
	var contractEngine contract.ContractEngine
	switch cfg.ContractEngine {
	case contract.DEMO_PIPE_CONTRACT_NAME:
		tmp := contract.CreatePipeContract(stateManager, cfg.RemoteEnginePipeName)
		contractEngine = tmp

		go contract.PipeChainCodeSupport(tmp, cfg.RemoteSupportPipeName)
	case contract.DEMO_CONTRACT_1_NAME:
		contractEngine, err = contract.CreateDemo_Contract_1(stateManager)
		if err != nil {
			panic("NewDper failed: Can't install Contract DEMO1!")
		}
	case contract.DEMO_CONTRACT_2_NAME:
		contractEngine, err = contract.CreateDemo_Contract_2(stateManager)
		if err != nil {
			panic("NewDper failed: Can't install Contract DEMO2!")
		}
	case contract.DEMO_CONTRACT_3_NAME:
		contractEngine, err = contract.CreateDemo_Contract_3(stateManager)
		if err != nil {
			panic("NewDper failed: Can't install Contract DEMO3!")
		}
	case contract.DEMO_CONTRACT_MIX123_NAME:
		contractEngine, err = contract.CreateDemo_Contract_MIX123(stateManager)
		if err != nil {
			panic("NewDper failed: Can't install Contract DEMO_MAX123!")
		}
	case contract.DEMO_CONTRACT_FL_NAME:
		contractEngine, err = contract.CreateDemo_Contract_FL(stateManager)
		if err != nil {
			panic("NewDper failed: Can't install Contract DEMO_FL!")
		}
	default:
		loglogrus.Log.Errorf("NewDper failed: Unknown contract engine: %s\n", cfg.ContractEngine)
		panic("NewDper failed: Unknown contract engine")
	}
	blockchain.InstallExecuteBlockFunc(contractEngine.CommitWriteSet)
	loglogrus.Log.Infof("NewDper: Contract engine launched\n")

	loglogrus.Log.Infof("NewDper: Now start net manager...\n")
	protocols := make([]server.Protocol, 0)
	shh := whisper.New() //本节点的whisper客户端对象
	protocols = append(protocols, shh.Protocol())
	spr := separator.New() //本节点的separator客户端对象
	protocols = append(protocols, spr.Protocol())
	syn := blockSync.New(blockchain)
	protocols = append(protocols, syn.Protocol())
	var cen *netconfig.Centre
	if cfg.CentralConfigMode { //若cfg.CentralConfigMode == true ,则采用中心化配置方式完成dpnet组网
		cen = netconfig.CentralNew()
		protocols = append(protocols, cen.Protocol())
	}
	srv := &server.Server{ //本节点的服务器对象
		PrivateKey:     selfKey.PrivateKey, //自身私钥
		Discovery:      true,               //允许被其他节点discover
		BootstrapNodes: cfg.BootstrapNodes, //引导节点
		MaxPeers:       cfg.MaxPeers,       // TODO: be customized  最大连接的节点数
		Name:           cfg.ServerName,
		Protocols:      protocols,      //使用的协议
		ListenAddr:     cfg.ListenAddr, //监听的地址
		NAT:            cfg.NAT,        // TODO: should be customized  NAT映射器
	}

	srv.Start() //启动节点的服务器模块
	// shh.Start() starts a whisper message expiration thread. As the net config
	// messages are not massive, it could be ignored.
	shh.Start()

	netManager := netconfig.NewNetManager(selfKey.PrivateKey, srv, shh, spr, cen, syn, 0, selfNode) //让本地节点维护一个本地的dp-chain网络
	err = netManager.ViewNetAddSelf()
	if err != nil {
		loglogrus.Log.Errorf("NewDper failed: Unable to join the current node to the corresponding partition network, err:%v\n", err)
		panic("NewDper failed: Unable to join the current node to the corresponding partition network!")
	}
	err = netManager.StartConfigChannel()
	if err != nil {
		loglogrus.Log.Errorf("NewDper failed: Failed to config lowerchannel consensus message processor for the whisper client module of the current node, err:%v\n", err)
		panic("NewDper failed: Failed to config lowerchannel consensus message processor for the whisper client module of the current node!")
	}
	loglogrus.Log.Infof("NewDper: Net manager hsa started\n")

	loglogrus.Log.Infof("NewDper: Now construct dper...\n")
	dper := &Dper{
		selfKey:           *selfKey,
		selfNode:          selfNode,
		blockDB:           blockDB,
		storageDB:         storageDB,
		accountManager:    accountManager,
		netManager:        netManager,
		p2pServer:         srv,
		stateManager:      stateManager,
		contractEngine:    contractEngine,
		txCheckPool:       TxCheckPool,
		txCheckChan:       make(chan common.Hash),
		txCheckClose:      make(chan bool),
		transactionFilter: transactionFilter,
	}
	loglogrus.Log.Infof("NewDper: Succeed in constructing dper\n")

	go dper.ArchiveHistoryBlock(cfg.BlockDataBasePath, blockchain, contractEngine.CommitWriteSet)

	return dper
}

func (dp *Dper) StateSelf(reconnection bool) error {

	if !reconnection {
		if err := dp.netManager.SendInitSelfNodeState(); err != nil {
			return err
		}
	} else { // 只有当正常连接方式错误时，才会发送断连请求消息
		if err := dp.netManager.SendReconnectState(); err != nil {
			return err
		}
	}
	return nil
}

func (dp *Dper) ConstructDpnet() error {
	err := dp.netManager.ConfigLowerChannel()
	if err != nil {
		return err
	}
	if dp.selfNode.Role == dpnet.Leader {
		err = dp.netManager.ConfigUpperChannel()
		if err != nil {
			return err
		}
	}
	return nil
}

func (dp *Dper) Start() error {
	err := dp.netManager.StartSeparateChannels()
	if err != nil {
		return err
	}
	var upperChannel consensus.UpperChannel
	var lowerChannel consensus.LowerChannel
	lowerChannel, err = dp.netManager.BackLowerChannel()
	if err != nil {
		return err
	}
	if dp.selfNode.Role == dpnet.Leader {
		upperChannel, err = dp.netManager.BackUpperChannel()
		if err != nil {
			return err
		}
	}

	validateManager := new(validator.ValidateManager)
	exportedValidator, err := dp.netManager.ExportValidator()
	if err != nil {
		return err
	}
	validateManager.Update(exportedValidator)
	validateManager.InjectTransactionFilter(dp.transactionFilter)
	validateManager.SetLowerValidFactor(3)
	validateManager.SetUpperValidRate(0.8)

	dp.netManager.Syn.SetValidateManager(validateManager)
	go dp.netManager.Syn.Start()

	commonBlockCache := new(eles.CommonBlockCache)
	commonBlockCache.Start()

	consensusPromoter := consensus.NewConsensusPromoter(dp.selfNode, dp.selfKey.PrivateKey, dp.stateManager)
	consensusPromoter.SetLowerChannel(lowerChannel)
	consensusPromoter.SetUpperChannel(upperChannel)
	consensusPromoter.SetValidateManager(validateManager)
	consensusPromoter.SetBlockCache(commonBlockCache)
	consensusPromoter.SetContractEngine(dp.contractEngine)

	for i := 0; i < len(INSTALL_TPBFT); i++ {
		switch INSTALL_TPBFT[i] {
		case "CTPBFT":
			ctpbft := consensus.NewCtpbft()
			ctpbft.Install(consensusPromoter)
			ctpbft.Start()
		case "FLTPBFT":
			fltpbft := consensus.NewFLtpbft()
			fltpbft.Install(consensusPromoter)
			fltpbft.Start()
		default:
			continue
		}
	}

	if dp.selfNode.Role == dpnet.Leader {
		knownBooters := dp.netManager.BackBooters()
		if len(knownBooters) == 0 {
			loglogrus.Log.Warnf("NewDper failed: Current Node could not find any booter!\n")
			return fmt.Errorf("no booter is known")
		}
		servicePrivoderNodeID := knownBooters[0]
		orderServiceAgent := blockOrdering.NewBlockOrderAgent(servicePrivoderNodeID, dp.netManager.Syn)
		orderServiceAgent.Install(consensusPromoter)
		err = orderServiceAgent.Start()
		if err != nil {
			return err
		}
	}

	consensusPromoter.Start()

	dp.validateManager = validateManager
	dp.consensusPromoter = consensusPromoter

	return nil
}

func (dp *Dper) Stop() {
	dp.consensusPromoter.Stop()
	dp.blockDB.Close()
	dp.storageDB.Close()
	dp.netManager.Close()

}

func (dp *Dper) PublishTransactions(txs []*eles.Transaction) {
	dp.consensusPromoter.PublishTransactions(txs)
}

func (dp *Dper) SimpleInvokeTransaction(user accounts.Account, contratAddr common.Address, functionAddr common.Address, args [][]byte) error {
	tx := eles.Transaction{
		Sender:    user.Address,
		Nonce:     0,
		Version:   dp.stateManager.CurrentVersion(),
		LifeTime:  0,
		Contract:  contratAddr,
		Function:  functionAddr,
		Args:      args,
		CheckList: make([]eles.CheckElement, 0),
	}
	tx.SetTxID()
	signature, err := dp.accountManager.SignHash(user, tx.TxID)
	if err != nil {
		return err
	}
	tx.Signature = signature
	dp.PublishTransactions([]*eles.Transaction{&tx})
	return nil
}

func (dp *Dper) SimpleInvokeTransactions(user accounts.Account, contratAddr common.Address, functionAddr common.Address, args [][]byte) error {

	txs := make([]*eles.Transaction, 0)
	for i := 0; i < BenchMarkFetch; i++ {
		tx := eles.Transaction{
			Sender:    user.Address,
			Nonce:     uint64(i),
			Version:   dp.stateManager.CurrentVersion(),
			LifeTime:  0,
			Contract:  contratAddr,
			Function:  functionAddr,
			Args:      args,
			CheckList: make([]eles.CheckElement, 0),
		}
		tx.SetTxID()
		signature, err := dp.accountManager.SignHash(user, tx.TxID)
		if err != nil {
			return err
		}
		tx.Signature = signature

		txs = append(txs, &tx)
	}

	dp.PublishTransactions(txs)
	return nil
}

func (dp *Dper) SimpleInvokeTransactionLocally(user accounts.Account, contratAddr common.Address, functionAddr common.Address, args [][]byte) [][]byte {
	tx := eles.Transaction{
		Sender:    user.Address,
		Nonce:     0,
		Version:   dp.stateManager.CurrentVersion(),
		LifeTime:  0,
		Contract:  contratAddr,
		Function:  functionAddr,
		Args:      args,
		CheckList: make([]eles.CheckElement, 0),
	}

	receipt, _ := dp.contractEngine.ExecuteTransactions([]eles.Transaction{tx})
	return receipt[0].Result
}

func (dp *Dper) SimplePublishTransaction(user accounts.Account, contratAddr common.Address, functionAddr common.Address, args [][]byte) (transactionCheck.CheckResult, error) {
	tx := eles.Transaction{
		Sender:    user.Address,
		Nonce:     0,
		Version:   dp.stateManager.CurrentVersion(),
		LifeTime:  0,
		Contract:  contratAddr,
		Function:  functionAddr,
		Args:      args,
		CheckList: make([]eles.CheckElement, 0),
	}
	tx.SetTxID()
	signature, err := dp.accountManager.SignHash(user, tx.TxID)
	if err != nil {
		return transactionCheck.CheckResult{}, err
	}
	tx.Signature = signature

	dp.PublishTransactions([]*eles.Transaction{&tx})

	//1.直接将产生的TxID输入到TxIDChan中,由SimpleCheckTransaction打包成交易上链验证事件
	dp.txCheckChan <- tx.TxID
	//2.等待一段时间,如果收到事件验证结果则打印,若超时则显示超时信息
	checkTicker := time.NewTicker(1 * time.Second)
	finishTicker := time.NewTimer(10 * time.Second)
	for {
		select {
		case <-finishTicker.C:
			return transactionCheck.CheckResult{}, errors.New("time out")
		case <-checkTicker.C:
			cresult := dp.txCheckPool.RetrievalResult(tx.TxID)
			if cresult != nil {
				return *cresult, nil
			}
		}
	}

}

func (dp *Dper) SimplePublishTransactions(user accounts.Account, contratAddr common.Address, functionAddr common.Address, args [][]byte) ([]*transactionCheck.CheckResult, error) {

	txs := make([]*eles.Transaction, 0)
	for i := 0; i < BenchMarkFetch; i++ {
		tx := eles.Transaction{
			Sender:    user.Address,
			Nonce:     uint64(i),
			Version:   dp.stateManager.CurrentVersion(),
			LifeTime:  0,
			Contract:  contratAddr,
			Function:  functionAddr,
			Args:      args,
			CheckList: make([]eles.CheckElement, 0),
		}
		tx.SetTxID()
		signature, err := dp.accountManager.SignHash(user, tx.TxID)
		if err != nil {
			return []*transactionCheck.CheckResult{}, err
		}
		tx.Signature = signature

		txs = append(txs, &tx)
	}
	dp.PublishTransactions(txs)

	//1.直接将产生的TxID输入到TxIDChan中,由SimpleCheckTransaction打包成交易上链验证事件
	for _, tx := range txs {
		dp.txCheckChan <- tx.TxID
	}

	//2.等待一段时间,如果收到事件验证结果则打印,若超时则显示超时信息
	checkTicker := time.NewTicker(1 * time.Second)
	finishTicker := time.NewTimer(6 * time.Second)
	txReceiptSet := make([]*transactionCheck.CheckResult, 0, len(txs))
	for {
		select {
		case <-finishTicker.C:
			if len(txReceiptSet) == 0 {
				return []*transactionCheck.CheckResult{}, errors.New("time out")
			} else {
				return txReceiptSet, nil
			}
		case <-checkTicker.C:
			for _, tx := range txs {
				cresult := dp.txCheckPool.RetrievalResult(tx.TxID)
				txReceiptSet = append(txReceiptSet, cresult)
			}
		}
	}

}

// 启动交易检测(检测交易是否上链成功--由协程捕获用户输入的交易ID)
func (dp *Dper) SimpleCheckTransaction(TxIDChan chan common.Hash, finish chan bool) error {

	var ErrorChan chan error = make(chan error, 1)

	go func() {
		for {
			err := <-ErrorChan //等待错误消息
			fmt.Println(err)
		}
	}()

	transactionCheck.TxCheckRun(dp.txCheckPool, ErrorChan, TxIDChan, finish) //启动交易查询功能
	return nil
}

func (dp *Dper) StartCheckTransaction() {
	go dp.SimpleCheckTransaction(dp.txCheckChan, dp.txCheckClose)
}

func (dp *Dper) BackAccountManager() *accounts.Manager {
	return dp.accountManager
}

func (dp *Dper) CloseCheckTransaction() {
	dp.txCheckClose <- true
}

func (dp *Dper) BackViewNetInfo() string {
	return dp.netManager.BackViewNetInfo()
}

func (dp *Dper) BackViewNet() DPNetwork {
	viewNet := dp.netManager.BackViewNet()

	dnw := DPNetwork{
		SubnetCount: len(viewNet.SubNets),
		LeaderCount: len(viewNet.Leaders),
		BooterCount: len(viewNet.Booters),
		NetGroups:   make([]NetGroup, 0),
	}

	for netID, subnet := range viewNet.SubNets {
		ng := NetGroup{
			NetID:       netID,
			MemberCount: len(subnet.Nodes),
		}
		dnw.NetGroups = append(dnw.NetGroups, ng)
	}
	return dnw
}

func dataArchiveSetup(das *DataAchive) bool {
	Cfg, err := ini.Load("./settings/extended.ini")
	if err != nil {
		loglogrus.Log.Errorf("Fail to parse './settings/http.ini': %v\n", err)
		return false
	} else {
		err = Cfg.Section("DataArchive").MapTo(das)
		if err != nil {
			loglogrus.Log.Errorf("Cfg.MapTo ServerSetting err: %v\n", err)
			return false
		}
		das.ArchiveScanInterval = das.ArchiveScanInterval * time.Second // 特殊项再赋值
		return true
	}
}

// 区块归档服务
func (dp *Dper) ArchiveHistoryBlock(blockDBPath string, oldBlockChain *eles.BlockChain, CommitWriteSet func(writeSet []eles.WriteEle) error) {

	DataAchiveSetting := &DataAchive{}
	if flag := dataArchiveSetup(DataAchiveSetting); !flag {
		return
	}
	archiveMode = DataAchiveSetting.ArchiveMode
	if !DataAchiveSetting.ArchiveMode {
		return
	}
	archiveTicker := time.NewTicker(DataAchiveSetting.ArchiveScanInterval)

	// var reloadBlockCount uint64 = 50  // 为了保证切换数据库之后,交易过滤器的正常使用,必须加载50个最新区块

	for {
		select {
		case <-archiveTicker.C:
			if dp.stateManager.BackLocalBlockNum() >= uint64(DataAchiveSetting.ArchiveThreshold) {

				dp.stateManager.ResetLocalBlockNum()

				pointHash := dp.stateManager.CurrentVersion()        // 记录归档点区块Hash
				pointHeight := dp.stateManager.GetBlockChainHeight() // 记录归档点区块高度

				//pointBlock := dp.stateManager.GetBlockFromHash(pointHash)
				// archiveBlocks := dp.stateManager.GetBlocksFromHash(pointBlock.RawBlock.PrevBlock, reloadBlockCount) // 将归档点之前的全部区块进行归档

				cs := new(eles.ChainState) // 区块链状态量
				cs.CurrentHash = pointHash
				cs.GenesisHash = pointHash
				cs.Height = pointHeight
				cs.ArchiveHeight = pointHeight - 1
				cs.TxSum = dp.stateManager.BackTxSum()

				// 必须禁止提交任何区块到数据库
				dp.stateManager.BackBlockStateMux().Lock()

				newDBPath := blockDBPath + fmt.Sprintf("%d", cs.Height)
				newBlockDB, err := database.NewLDBDatabase(newDBPath)
				if err != nil {
					loglogrus.Log.Errorf("NewDper failed: Can't Load Persistent data from Specified path:%s , err:%v\n", newDBPath, err)
					panic("NewDper failed: Can't Load Persistent data from Specified path")
				}

				// 将区块链状态量写入新的存储数据库
				key := []byte("chainState")
				value := eles.ChainStateSerialize(cs)
				newBlockDB.Put(key, value)

				// TODO: 交易过滤器和检查池或许可以使用旧的
				// var transactionFilter *eles.TransactionFilter
				// if REPEAT_CHECK_ENABLE {
				// 	transactionFilter = eles.NewTransactionFilter()
				// }
				// TxCheckPool := transactionCheck.NewTaskPool() //创建一个交易检查事件池
				// newBlockchain := eles.InitBlockChain(newBlockDB, TxCheckPool, transactionFilter)
				// newBlockchain.InstallExecuteBlockFunc(CommitWriteSet)

				dp.stateManager.BackBlockStateMux().Unlock()
				oldBlockChain.Database.Close()
				// 更新BlockChain
				oldBlockChain.Database = newBlockDB

				ConfigBlockDBPath(newDBPath)
			}
		}
	}
}

func ConfigBlockDBPath(dbPath string) {
	result := make(map[string]interface{})
	dstPath := "." + string(os.PathSeparator) + "settings" + string(os.PathSeparator)
	jsonFile := dstPath + "dperConfig.json"
	if err := ReadJson(jsonFile, result); err != nil { // 读取json文件配置
		loglogrus.Log.Errorf("Read Dper Config file is failed,err:%v", err)
		panic(fmt.Sprintf("Read Dper Config file is failed,err:%v", err))
	}

	result["BlockDBPath"] = dbPath
	WriteJson(jsonFile, result) //重写json配置文件

}

func ReadJson(jsonFile string, result map[string]interface{}) error {
	byteValue, err := os.ReadFile(jsonFile) //读取json文件
	if err != nil {
		return err
	}
	err = json.Unmarshal(byteValue, &result) //解析json k-v对
	if err != nil {
		return err
	}
	fmt.Printf("NewAddressMode:%v\n", result["NewAddressMode"])
	fmt.Printf("BooterKeyAddress:%v\n", result["BooterKeyAddress"])
	fmt.Printf("DperKeyAddress:%v\n", result["DperKeyAddress"])
	fmt.Printf("ListenAddress:%v\n", result["ListenAddress"])
	return nil
}

func WriteJson(jsonFile string, result interface{}) {
	temp, _ := json.MarshalIndent(result, "", "")
	if err := os.WriteFile(jsonFile, temp, 0644); err != nil {
		panic(fmt.Sprintf("Reset Dper Config file is failed,err:%v", err))
	}
}
