package separatortest

import (
	"crypto/ecdsa"
	"dpchain/accounts"
	"dpchain/core/dpnet"
	"dpchain/core/netconfig"
	"dpchain/core/separator"
	"dpchain/crypto"
	"dpchain/crypto/keys"
	"dpchain/database"
	"dpchain/p2p/discover"
	"dpchain/p2p/nat"
	"dpchain/p2p/server"
	"dpchain/p2p/whisper"
	"errors"
	"fmt"
	"testing"
	"time"

	loglogrus "dpchain/log_logrus"
)

// 模拟用本地监听地址
var listenAddr = []string{
	"127.0.0.1:20130",
	"127.0.0.1:20131",
	"127.0.0.1:20132",
	"127.0.0.1:20133",
}

var testNodes []mockNode //存储所有测试用模拟节点

type mockNode struct {
	prvKey       *ecdsa.PrivateKey
	listenAddr   string
	discoverNode *discover.Node
}

type Dper struct {
	selfKey        keys.Key
	blockDB        database.Database //本地维护一个区块数据库
	storageDB      database.Database //本地维护一个存储数据库
	accountManager *accounts.Manager

	netManager *netconfig.NetManager //本地维护一个dp-chain网络
	p2pServer  *server.Server
}

type DpPer struct {
	peer     Dper
	peerchan chan bool
}

var DpPers = [4]DpPer{
	{
		peerchan: make(chan bool),
	},
	{
		peerchan: make(chan bool),
	},
	{
		peerchan: make(chan bool),
	},
	{
		peerchan: make(chan bool),
	},
}

// 创建一组测试用网络节点(discover.Node)
func initTest() error {
	for i := 0; i < len(listenAddr); i++ {
		prv, err := crypto.GenerateKey()
		if err != nil {
			return err
		}
		discoverNode, err := discover.ParseUDP(prv, listenAddr[i]) //根据给定的私钥和监听地址返回一个node
		if err != nil {
			return err
		}
		mn := mockNode{
			prvKey:       prv,           //节点的私钥
			listenAddr:   listenAddr[i], //节点的监听地址
			discoverNode: discoverNode,  //本地节点
		}
		testNodes = append(testNodes, mn)
	}
	return nil
}

type mockConfig struct {
	bootstrapNodes []*discover.Node
	listenAddr     string
	nat            nat.Interface
	isBooter       bool
	netID          string
}

// 在本地创建一个dp-chain网络节点(开放whisper客户端、separator客户端、服务器三大模块)
// 然后为此节点在本地维护一个dp-chain网络
func mockDper(mc *mockConfig) (*Dper, error) {
	prvKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	selfKey := keys.NewKeyFromECDSA(prvKey)
	tempName := "dpchain:test"
	shh := whisper.New()          //创建本节点的whisper客户端
	spr := separator.New()        //创建本节点的separator客户端
	cen := netconfig.CentralNew() //创建本节点的central客户端
	srv := &server.Server{        //创建本节点的服务器
		PrivateKey:     prvKey,
		Discovery:      true,
		BootstrapNodes: mc.bootstrapNodes,
		MaxPeers:       10,
		Name:           tempName,
		Protocols:      []server.Protocol{shh.Protocol(), spr.Protocol()},
		ListenAddr:     mc.listenAddr,
		NAT:            mc.nat,
	}

	//启动服务器模块
	srv.Start()

	var role int
	if mc.isBooter {
		role = int(dpnet.Booter)
	} else {
		role = int(dpnet.UnKnown)
	}

	var netID string
	if mc.netID == "" {
		netID = dpnet.InitialNetID
	} else {
		netID = mc.netID
	}

	selfNodeID := crypto.KeytoNodeID(&prvKey.PublicKey)
	selfNode := dpnet.NewNode(selfNodeID, role, netID)                                  //根据NodeID/netID/role 创建一个dp-chain 本地网络节点
	netManager := netconfig.NewNetManager(prvKey, srv, shh, spr, cen, nil, 0, selfNode) //为本地节点维护一个dp-chain网络

	dper := &Dper{
		selfKey:    *selfKey,
		netManager: netManager, //本地dp-chain网络的控制对象(或者说工具)
		p2pServer:  srv,        //本地p2p网络的控制对象
	}
	return dper, nil
}

// 根据配置信息，网络ID，节点身份 让本地节点加入到dp-net中
func dpnetNode(index int, tempMockCfg *mockConfig, netID string, role dpnet.RoleType) (*Dper, error) {
	testDper, err := mockDper(tempMockCfg) //按照配置信息，在本地创建一个dp-chain网络节点，并维护一个dp-chain网络
	if err != nil {
		return nil, errors.New("fail in mockDper: " + fmt.Sprint(err))
	}
	testDper.netManager.SetSelfNodeNetID(netID) //设置本地节点的网络名(归属于哪一个网络)
	testDper.netManager.SetSelfNodeRole(role)   //将本地节点设置为本地dp-chain网络的领导者

	if err := testDper.netManager.ViewNetAddSelf(); err != nil { //将本地节点按照上述两步配置加入到本地dp-chain网络
		return nil, errors.New("fail in init view net: " + fmt.Sprint(err))
	}
	//打印netManager字段的selfNode字段中保存的dp-net网络中本地节点的NodeID 和 netID
	loglogrus.Log.Infof("dper netID: %s,Role: %d\n", testDper.netManager.GetSelfNodeNetID(), testDper.netManager.GetSelfNodeRole())

	if err := testDper.netManager.StartConfigChannel(); err != nil { //为whisper客户端配置一个对接收消息的处理方法
		return nil, errors.New("fail in start config channel: " + fmt.Sprint(err))
	}
	time.Sleep(3 * time.Second)
	if err := testDper.netManager.SendInitSelfNodeState(); err != nil { //将自身的NodeID/NetID/Role等信息whisper广播发送给目标节点(用于构建separator网络)
		return nil, errors.New("fail in send self node state: " + fmt.Sprint(err))
	}
	time.Sleep(2 * time.Second)

	if err := testDper.netManager.ConfigLowerChannel(); err != nil { //将netManager字段中获取网络信息同步到spr字段中(只把当前自己所在的子网的部署情况进行同步)，最后赋给netManager的lowerChannel字段
		return nil, errors.New("fail in config lower channel: " + fmt.Sprint(err))
	}
	time.Sleep(2 * time.Second)
	if err := testDper.netManager.StartSeparateChannels(); err != nil { //让本地节点运行separator协议
		return nil, errors.New("fail in start separate channels: " + fmt.Sprint(err))
	}
	time.Sleep(5 * time.Second)

	//打印netManager字段的viewNet字段中保存的dp-net网络中其他网络节点的NodeID 和 netID
	/*viewNet := make(map[string]*dpnet.SubNet)
	viewNet = testDper.netManager.GetDPNetInfo()
	for netID, _ := range viewNet {
		fmt.Println("当前网络ID: ", viewNet[netID].NetID)
		fmt.Println("当前网络的领导节点哈希：", viewNet[netID].Leaders)
		fmt.Println("当前子网下网络节点个数:", len(viewNet[netID].Nodes))
		fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~子网分割线")
	}
	fmt.Println("-------------------------------------------------------------节点分割线")
	*/
	//打印netManager字段的spr字段中
	/*sprGroup := make([]string, 0)
	sprGroup = testDper.netManager.Spr.GetPeerGroup()

	for i, v := range sprGroup {
		fmt.Printf("第%d个子网,netID = %s\n", i, v)
	}
	fmt.Println("-------------------------------------------------------------节点分割线")
	*/
	DpPers[index].peer = *testDper
	DpPers[index].peerchan <- true
	return testDper, nil
}

func TestMultiNodeWorkFlow(t *testing.T) {
	if err := initTest(); err != nil { //创造一组测试用Node节点
		loglogrus.Log.Errorf("fail in initTest: %v", err)
	}
	tempMockCfg := [4]mockConfig{
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[0].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
			netID:          "testnet1",
		},
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[1].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
			netID:          "testnet1",
		},
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[2].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
			netID:          "testnet2",
		},
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[3].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
			netID:          "testnet2",
		},
	}
	tempMockCfg[1].bootstrapNodes = append(tempMockCfg[1].bootstrapNodes, testNodes[0].discoverNode)
	tempMockCfg[2].bootstrapNodes = append(tempMockCfg[2].bootstrapNodes, testNodes[0].discoverNode)
	tempMockCfg[3].bootstrapNodes = append(tempMockCfg[3].bootstrapNodes, testNodes[0].discoverNode)

	go dpnetNode(0, &tempMockCfg[0], "testnet1", dpnet.Leader)
	go dpnetNode(1, &tempMockCfg[1], "testnet1", dpnet.Follower)
	go dpnetNode(2, &tempMockCfg[2], "testnet2", dpnet.Leader)
	go dpnetNode(3, &tempMockCfg[3], "testnet2", dpnet.Follower)

	for i := 0; i < 4; i++ {
		<-DpPers[i].peerchan
	}

	msgcode := separator.MessageCode
	var payload []byte
	tempMsg := [4]separator.Message{
		{ //设定配置信息
			MsgCode: uint64(msgcode),
			NetID:   "testnet1",
			From:    DpPers[0].peer.netManager.SelfNode.NodeID,
			PayLoad: payload,
		},
		{ //设定配置信息
			MsgCode: uint64(msgcode),
			NetID:   "testnet1",
			From:    DpPers[1].peer.netManager.SelfNode.NodeID,
			PayLoad: payload,
		},
		{ //设定配置信息
			MsgCode: uint64(msgcode),
			NetID:   "testnet2",
			From:    DpPers[2].peer.netManager.SelfNode.NodeID,
			PayLoad: payload,
		},
		{ //设定配置信息
			MsgCode: uint64(msgcode),
			NetID:   "testnet2",
			From:    DpPers[3].peer.netManager.SelfNode.NodeID,
			PayLoad: payload,
		},
	}

	for i := 0; i < 4; i++ { //计算Msg哈希值
		tempMsg[i].Hash = tempMsg[i].CalculateHash()
		//fmt.Printf("tempMsg[%d].Hash: %v\n", i, tempMsg[i].Hash)
	}

	//1. 0号节点(1号网络的leader)广播给节点组内所有成员节点
	if testPg, err := DpPers[0].peer.netManager.Spr.BackPeerGroup(tempMsg[0].NetID); err != nil { //返回子网ID对应的 peergroup
		loglogrus.Log.Errorf("fail in Back PeerGroup: %s", err)
	} else {
		// tempMsg[0].Send(testPg, tempMsg[1].From) //1号网络的leader向1号网络的follower发送消息tempMsg[0]
		tempMsg[0].Broadcast(testPg)
		time.Sleep(2 * time.Second)
	}
	//2. 1号节点(1号网络的follower)广播给节点组内所有成员节点
	if testPg, err := DpPers[1].peer.netManager.Spr.BackPeerGroup(tempMsg[1].NetID); err != nil { //返回子网ID对应的 peergroup
		loglogrus.Log.Errorf("fail in Back PeerGroup: %s", err)
	} else {
		tempMsg[1].Broadcast(testPg)
		time.Sleep(2 * time.Second)
	}
	//3. 2号节点(1号网络的follower)广播给节点组内所有成员节点
	if testPg, err := DpPers[2].peer.netManager.Spr.BackPeerGroup(tempMsg[2].NetID); err != nil { //返回子网ID对应的 peergroup
		loglogrus.Log.Errorf("fail in Back PeerGroup: %s", err)
	} else {
		tempMsg[2].Broadcast(testPg)
		time.Sleep(2 * time.Second)
	}
	//3. 3号节点(1号网络的follower)广播给节点组内所有成员节点
	if testPg, err := DpPers[3].peer.netManager.Spr.BackPeerGroup(tempMsg[3].NetID); err != nil { //返回子网ID对应的 peergroup
		loglogrus.Log.Errorf("fail in Back PeerGroup: %s", err)
	} else {
		tempMsg[3].Broadcast(testPg)
		time.Sleep(2 * time.Second)
	}

	node_0_group, _ := DpPers[0].peer.netManager.Spr.BackPeerGroup(tempMsg[0].NetID)
	node_1_group, _ := DpPers[1].peer.netManager.Spr.BackPeerGroup(tempMsg[1].NetID)
	node_2_group, _ := DpPers[2].peer.netManager.Spr.BackPeerGroup(tempMsg[2].NetID)
	node_3_group, _ := DpPers[3].peer.netManager.Spr.BackPeerGroup(tempMsg[3].NetID)

	var msgIndex [4]int

	msgIndex[0] = separator.Recieve(node_0_group)
	loglogrus.Log.Debug("接收者 : 1号网络Leader")
	loglogrus.Log.Debug("收到的message消息数:", msgIndex[0])
	time.Sleep(3 * time.Second)
	loglogrus.Log.Debug("--------------------------------------------------------------------------")

	msgIndex[1] = separator.Recieve(node_1_group)
	loglogrus.Log.Debug("接收者 : 1号网络Follwer")
	loglogrus.Log.Debug("收到的message消息数:", msgIndex[1])
	time.Sleep(3 * time.Second)
	loglogrus.Log.Debug("--------------------------------------------------------------------------")

	msgIndex[2] = separator.Recieve(node_2_group)
	loglogrus.Log.Debug("接收者 : 2号网络Leader")
	loglogrus.Log.Debug("收到的message消息数:", msgIndex[2])
	time.Sleep(3 * time.Second)
	loglogrus.Log.Debug("--------------------------------------------------------------------------")

	msgIndex[3] = separator.Recieve(node_3_group)
	loglogrus.Log.Debug("接收者 : 2号网络Follwer")
	loglogrus.Log.Debug("收到的message消息数:", msgIndex[3])
	loglogrus.Log.Debug("--------------------------------------------------------------------------")

}
