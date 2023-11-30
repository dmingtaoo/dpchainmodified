package dper

import (
	"dpchain/common"
	"dpchain/core/blockOrdering"
	"dpchain/core/dpnet"
	"dpchain/core/netconfig"
	"dpchain/core/separator"
	"dpchain/crypto"
	"dpchain/crypto/keys"
	"dpchain/crypto/randentropy"
	"dpchain/logger"
	"dpchain/logger/glog"
	"dpchain/p2p/discover"
	"dpchain/p2p/nat"
	"dpchain/p2p/server"
	"dpchain/p2p/whisper"
	"fmt"
	"os"
)

type OrderServiceServer struct {
	selfKey              keys.Key
	selfNode             *dpnet.Node
	netManager           *netconfig.NetManager
	p2pServer            *server.Server
	orderServiceProvider *blockOrdering.BlockOrderServer
}

type OrderServiceServerConfig struct {
	// basic information
	NewAddressMode  bool //whether to create a new account when start
	SelfNodeAddress common.Address
	KeyStoreDir     string

	// connect configuration
	ServerName     string
	ListenAddr     string
	NAT            nat.Interface
	BootstrapNodes []*discover.Node //整个dp-chain中已知的引导节点
	MaxPeers       int

	// about dpnet configuration
	CentralConfigMode bool // whether to use central protocol to config dpnet
}

func NewOrderServiceServer(cfg *OrderServiceServerConfig) (*OrderServiceServer, error) {
	glog.V(logger.Info).Infof("Initialize the dper...")
	glog.V(logger.Info).Infof("Loading key store...")
	err := os.Mkdir(cfg.KeyStoreDir, 0750)
	if err != nil {
		if os.IsExist(err) {
			glog.V(logger.Info).Infof("Key store path: %s, is existed", cfg.KeyStoreDir)
		} else {
			return nil, fmt.Errorf("Fail in loading key store, path: %s", cfg.KeyStoreDir)
		}
	}
	// keyStore := keys.NewKeyStorePlain(cfg.KeyStoreDir)
	keyStore := keys.NewKeyStorePassphrase(cfg.KeyStoreDir)
	var selfKey *keys.Key
	if cfg.NewAddressMode {
		var err error
		selfKey, err = keyStore.GenerateNewKey(randentropy.Reader, "")
		if err != nil {
			return nil, fmt.Errorf("fail in create new dper: %v", err)
		}
	} else {
		var err error
		selfKey, err = keyStore.GetKey(cfg.SelfNodeAddress, "")
		if err != nil {
			return nil, fmt.Errorf("fail in create new dper: %v", err)
		}
	}
	publicKey := &selfKey.PrivateKey.PublicKey
	glog.V(logger.Info).Infof("Complete loading key store")

	glog.V(logger.Info).Infof("Create self node...")
	selfNode := dpnet.NewNode(crypto.KeytoNodeID(publicKey), int(dpnet.Booter), dpnet.UpperNetID)
	glog.V(logger.Info).Infof("Complete creating self node")

	glog.V(logger.Info).Infof("Now start net manager...")
	protocols := make([]server.Protocol, 0)
	shh := whisper.New() //本节点的whisper客户端对象
	protocols = append(protocols, shh.Protocol())
	spr := separator.New() //本节点的separator客户端对象
	protocols = append(protocols, spr.Protocol())
	var cen *netconfig.Centre
	if cfg.CentralConfigMode {
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
	netManager := netconfig.NewNetManager(selfKey.PrivateKey, srv, shh, spr, cen, nil, 0, selfNode) //让本地节点维护一个本地的dp-chain网络

	err = netManager.ViewNetAddSelf()
	if err != nil {
		return nil, err
	}
	err = netManager.StartConfigChannel()
	if err != nil {
		return nil, err
	}
	glog.V(logger.Info).Infof("Net manager hsa started")

	glog.V(logger.Info).Infof("Now construct order service server...")
	oss := &OrderServiceServer{
		selfKey:    *selfKey,
		selfNode:   selfNode,
		netManager: netManager,
		p2pServer:  srv,
	}
	glog.V(logger.Info).Infof("Complete constructing order service server")
	return oss, nil
}

func (oss *OrderServiceServer) StateSelf() error {
	err := oss.netManager.SendInitSelfNodeState()
	if err != nil {
		return err
	}
	return nil
}

func (oss *OrderServiceServer) ConstructDpnet() error {
	err := oss.netManager.ConfigUpperChannel()
	if err != nil {
		return err
	}
	return nil
}

func (oss *OrderServiceServer) Start() error {
	err := oss.netManager.StartSeparateChannels()
	if err != nil {
		return err
	}
	upperChannel, err := oss.netManager.BackUpperChannel()
	if err != nil {
		return err
	}
	//orderServiceProvider := consensus.NewOrderServiceProvider(oss.selfNode, common.Hash{}, upperChannel)
	orderServiceProvider := blockOrdering.NewBlockOrderServer(oss.selfNode, common.Hash{}, 0, upperChannel)
	err = orderServiceProvider.Start()
	if err != nil {
		return err
	}
	oss.orderServiceProvider = orderServiceProvider
	return nil
}

func (oss *OrderServiceServer) Stop() {
	oss.netManager.Close()
	oss.p2pServer.Stop()
	oss.orderServiceProvider.Stop()
}

func (oss *OrderServiceServer) BackSelfUrl() (string, error) {
	return oss.netManager.BackSelfUrl()
}

func (oss *OrderServiceServer) BackViewNetInfo() string {
	return oss.netManager.BackViewNetInfo()
}

func (oss *OrderServiceServer) BackNetManager() *netconfig.NetManager {
	return oss.netManager
}
