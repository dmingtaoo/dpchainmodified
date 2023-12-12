package router

import (
	"dpchain/api"
	"dpchain/ginHttp/pkg/setting"

	"github.com/gin-gonic/gin"
)
func CORSMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept")
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(200)
            return
        }
        c.Next()
    }
}

func InitRouter(bs *api.BlockService, ds *api.DperService, ns *api.NetWorkService, ct *api.ContractService) *gin.Engine {
	r := gin.New()
    r.Use(gin.Logger())
    r.Use(gin.Recovery())
    r.Use(CORSMiddleware()) // 应用 CORS 中间件

    gin.SetMode(setting.ServerSetting.RunMode)
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	gin.SetMode(setting.ServerSetting.RunMode)

	block := r.Group("block")
	network := r.Group("network")
	dper := r.Group("dper")

	contract := r.Group("contract")

	if bs != nil {
		block.GET("/blockNumber", GetCurrentBlockNumber(bs))
		block.GET("/blockHash", GetCurrentBlockHash(bs))
		block.GET("/getBlockByHash/:hash", GetBlockByHash(bs))
		block.GET("/getBlockByNumber/:number", GetBlockByNumber(bs))
		block.GET("/getBlockHashByNumber/:number", GetBlockHashByNumber(bs))
		block.GET("/getRecentBlocks/:count", GetRecentBlocks(bs))
		block.GET("/getAllBlocks", GetAllBlocks(bs))

		block.GET("/getTransactionNumber", GetTransactionNumber(bs))
		block.GET("/getTransactionByHash/:hash", GetTransactionByHash(bs))
		block.GET("/getTransactionByBlockHashAndIndex", GetTransactionByBlockHashAndIndex(bs))
		block.GET("/getTransactionsByBlockHash/:blockHash", GetTransactionsByBlockHash(bs))
		block.GET("/getTransactionsByBlockNumber/:blockNumber", GetTransactionsByBlockNumber(bs))
		block.GET("/getRecentTransactions/:count", GetRecentTransactions(bs))
		block.GET("/getAllTransactions", GetAllTransactions(bs))
	}

	if ds != nil {
		dper.GET("/accountsList", BackAccountList(ds))
		dper.POST("/newAccount", CreateNewAccount(ds))
		dper.POST("/useAccount", UseAccount(ds))
		dper.GET("/currentAccount", CurrentAccount(ds))
		dper.POST("/txCheck", OpenTxCheckMode(ds))
		dper.POST("/solidInvoke", SolidInvoke(ds))
		dper.POST("/solidCall", SolidCall(ds))
		dper.POST("/softInvoke", SoftInvoke(ds))

		dper.POST("/sendvcrequest/:destinationPort", SendVCrequest(ds)) //just for scene 3, when sp request ue's signature
		dper.POST("/sendvc/:destinationPort", SendVC(ds))               //just for scene 3, when sp request ue's signature
		dper.POST("/sendrandom/:destinationPort", SendRandom(ds))
		dper.POST("/signaturereturn", SignatureReturn(ds))
		dper.POST("/signaturereturn2", SignatureReturn(ds))
		dper.POST("/signvalid", SignValid(ds))
		dper.POST("/vcreceive", VCReceive(ds))
		dper.POST("/vcreceive2", VCReceive2(ds))
		dper.POST("/vcvalid", VCValid(ds))
		dper.POST("/datarequest/:destinationPort", DataRequest(ds))
		dper.POST("/datasend", DataSend(ds))
		dper.POST("/datarequest2/:destinationPort", DataRequest(ds))
		dper.POST("/datasend2", DataSend(ds))
		dper.POST("/getaddress", GetAddress(ds))
		dper.POST("/transvcrequest/:destinationPort", TransVCrequest(ds))
		dper.POST("/transvc", TransVC(ds))

		dper.POST("/softInvokeQuery", SoftInvokeQuery(ds))
		dper.POST("/publishTx", PublishTx(ds))
		dper.POST("/softCall", SoftCall(ds))
		dper.PUT("/beBooter", BecomeBooter(ds))
		dper.GET("/viewNet", BackViewNet(ds))
		dper.GET("/helpMenu", HelpMenu(ds))
		dper.GET("/exit", Exit(ds))

	}

	if ns != nil {
		network.GET("/networkInfo", BackDPNetWork(ns))
		network.GET("/allConsensusNode", BackAllConsensusNode(ns))
		network.GET("/selfNodeInfo", BackNodeInfoSelf(ns))
		network.POST("/nodeInfoByNodeID", BackNodeInfoByNodeID(ns))

		network.GET("/groupCount", BackGroupCount(ns))
		network.GET("/allGroupName", BackAllGroupName(ns))
		network.GET("/upperNet", BackUpperNetNodeList(ns))
		network.GET("/allBooters", BackAllBooters(ns))
		network.GET("/allLeaders", BackAllLeaders(ns))
		network.POST("/subNetNodeID", BackNodeListByGroupName(ns))
		network.POST("/subNetLeaderID", BackLeaderNodeIDByGroupName(ns))
		network.POST("/subNetInfo", BackSubNetByGroupName(ns))

	}

	if ct != nil {
		contract.GET("/credit", BackCredit(ct))
		contract.GET("/stampList", BackStampList(ct))
		contract.POST("/mintNewStamp", MintNewStamp(ct))
		contract.POST("/transStamp", TransStamp(ct))
	}

	return r

}

/////what the fuck this is just a test for github commit
