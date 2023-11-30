package ginHttp

import (
	"dpchain/api"
	"dpchain/ginHttp/pkg/setting"
	router "dpchain/ginHttp/router/api"
	"fmt"
	"net/http"
)

func NewGinRouter(bs *api.BlockService, ds *api.DperService, ns *api.NetWorkService, ct *api.ContractService) {

	setting.Setup()

	router := router.InitRouter(bs, ds, ns, ct) //返回一个gin路由器

	s := &http.Server{
		Addr:           setting.ServerSetting.IP + fmt.Sprintf(":%d", setting.ServerSetting.HttpPort),
		Handler:        router,
		ReadTimeout:    setting.ServerSetting.ReadTimeout,
		WriteTimeout:   setting.ServerSetting.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	s.ListenAndServe()
}
