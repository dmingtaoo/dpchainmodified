package netconfig

import (
	"context"
	"dpchain/common"
	"dpchain/core/dpnet"
	"dpchain/logger"
	"dpchain/logger/glog"
	"dpchain/p2p/discover"
	"dpchain/p2p/server"
	"sync"
)

const (
	statusCode    = 0x00 // for handshake
	ElectionCode  = 0x01 // 选举中心节点使用的广播消息
	ConfigureCode = 0x02 //中心节点的网络配置消息

	protocolVersion uint64 = 0x00 // ver 0.0
	protocolName           = "cen"
)

// Central协议在whisper协议实现节点P2P通信的基础上,在网络中选举出一个中心节点
type Centre struct {
	protocol   server.Protocol                  //继承Protocal类，需自行实现central协议功能
	addedPeers map[common.NodeID]*discover.Node // 需要维持 静态连接 的节点列表
	knownPeers map[common.NodeID]*peer          //所有存在central协议通信的节点的peer对象集合

	target int //根据计算能力选举出中心节点

	staticMu sync.Mutex // Mutex to sync the static nodes to be added
	peerMu   sync.Mutex // Mutex to sync the active known peers

	quitElection chan struct{} //关闭选举广播的协程 通道
	quitMining   chan struct{} //停止挖矿协程 通道

	DpNet        dpnet.DpNet        //中心节点需要完成对整个dpNet网络的部署,然后广播给所有节点;
	DpNetCentral dpnet.DpNetCentral //其余节点则是以此为依据,利用separator协议完成整个网络的初始化
	selfNode     *dpnet.Node        //自身的节点信息(未完成central协议之前只有NodeID是已知的)
}

// 新建一个Central对象,自主实现protocal类,协议Run函数为Central.handlePeer
func CentralNew() *Centre {
	centre := &Centre{
		addedPeers:   make(map[common.NodeID]*discover.Node),
		knownPeers:   make(map[common.NodeID]*peer),
		quitElection: make(chan struct{}),
		quitMining:   make(chan struct{}),
	}

	centre.protocol = server.Protocol{
		Name:    protocolName,
		Version: uint(protocolVersion),
		Length:  6, // 0x00 to 0x05
		Run:     centre.handlePeer,
	}
	centre.DpNet = *dpnet.NewDpNet()

	return centre
}

// 获取当前separator对象支持的协议Protocal对象
func (cen *Centre) Protocol() server.Protocol {
	return cen.protocol
}

// 必须保证中心节点与其他所有P2P网络中的节点存在静态连接
func (cen *Centre) handlePeer(peer *server.Peer, rw server.MsgReadWriter, ctx context.Context) error {
	centralPeer := newPeer(cen, peer, rw) //在p2p连接的基础上与对方peer建立centralPeer协议
	cen.peerMu.Lock()
	cen.knownPeers[centralPeer.nodeID] = centralPeer //在knownPeers记录本节点与对端节点的centralPeer连接控制器
	cen.peerMu.Unlock()

	defer func() {
		cen.peerMu.Lock()
		delete(cen.knownPeers, centralPeer.nodeID) //无法正常收到对端节点的消息时，将此对端节点从knownPeers切片中移除
		cen.peerMu.Unlock()
	}()

	//向对方节点发送central协议启动消息，并等待回复
	if err := centralPeer.handshake(); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			packet, err := rw.ReadMsg() //等待中心节点的消息
			if err != nil {
				return err
			}
			switch packet.Code {
			case ConfigureCode: //此消息中包含对整个dpnet网络的部署
				var msg Message
				if err := packet.Decode(&msg); err != nil { //对message包进行rlp解码
					glog.V(logger.Warn).Infof("%v: failed to decode msg: %v", peer, err)
					continue
				}
				//1.TODO:验证消息是否来自于中心节点(数字签名。)

				//2.根据消息的DpNet字段利用separator协议进行dpnet网络的本地初始化
				cen.DpNetCentral = msg.Payload
				cen.CentralizedNetConf()
			}
		}
	}
}

// 根据从中心节点收到的配置消息(msg.Payload)，在本地根据separator协议完成dpnet部署(利用DpNetCentral对象完成对DpNet的更新)
func (cen *Centre) CentralizedNetConf() error {
	//1.先更新Booter列表
	for _, booterID := range cen.DpNetCentral.Booters {
		cen.DpNet.Booters = append(cen.DpNet.Booters, booterID)
	}
	//2.更新SubNets列表(同时完成Leader列表的更新)
	for _, group := range cen.DpNetCentral.SubNets {
		netID := group.NetID //记录子网id
		subNet := new(dpnet.SubNet)
		subNet = &dpnet.SubNet{
			NetID:   netID,
			Leaders: make([]common.NodeID, 0),
			Nodes:   make(map[common.NodeID]*dpnet.Node),
		}
		for _, node := range group.Nodes { //遍历同一子网下的节点集合,将一个子网的组成添加到新的dpnet.SubNet对象中

			if node.Role == dpnet.Leader {
				cen.DpNet.Leaders[netID] = node.NodeID //如果是Leader节点,还需要额外加入DpNet对象的Leaders集合中
				subNet.Leaders = append(subNet.Leaders, node.NodeID)
			}
			subNet.Nodes[node.NodeID] = &node
		}
		cen.DpNet.SubNets[netID] = subNet
	}
	return nil
}

// 获取knowsPeer集合所有节点的NodeID组成的集合
func (cen *Centre) GetKnowsPeerNodeID() []common.NodeID {
	idArray := make([]common.NodeID, 0)
	for id, _ := range cen.knownPeers {
		idArray = append(idArray, id)
	}
	return idArray
}

// 获取knowsPeer集合所有节点的peer对象
func (cen *Centre) GetKnowsPeer() []*peer {
	peerArray := make([]*peer, 0)
	for _, peer := range cen.knownPeers {
		peerArray = append(peerArray, peer)
	}
	return peerArray
}
func (cen *Centre) GetDPNetInfo() map[string]*dpnet.SubNet {
	return cen.DpNet.SubNets
}
