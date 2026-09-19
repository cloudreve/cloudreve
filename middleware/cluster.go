package middleware

import (
	"sync"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/routers/controllers"
	"github.com/gin-gonic/gin"
)

type SlaveNodeSettingGetter interface {
	// GetNodeSetting returns the node settings and its hash
	GetNodeSetting() (*types.NodeSetting, string)
}

var downloaderPool = sync.Map{}

// PrepareSlaveDownloader creates or resume a downloader based on input node settings
func PrepareSlaveDownloader(dep dependency.Dep, ctxKey interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeSettings, hash := controllers.ParametersFromContext[SlaveNodeSettingGetter](c, ctxKey).GetNodeSetting()

		// try to get downloader from pool
		if d, ok := downloaderPool.Load(hash); ok {
			c.Set(downloader.DownloaderCtxKey, d)
			c.Next()
			return
		}

		// create a new downloader
		d, err := cluster.NewDownloader(c, dep.RequestClient(request.WithContext(c), request.WithLogger(dep.Logger())), dep.SettingProvider(), nodeSettings)
		if err != nil {
			c.JSON(200, serializer.ParamErr(c, "Failed to create downloader", err))
			c.Abort()
			return
		}

		// save downloader to pool
		downloaderPool.Store(hash, d)
		c.Set(downloader.DownloaderCtxKey, d)
		c.Next()
	}
}

func SlaveRPCSignRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeId := cluster.NodeIdFromContext(c)
		if nodeId == 0 {
			c.JSON(200, serializer.ParamErr(c, "Unknown node ID", nil))
			c.Abort()
			return
		}

		np, err := dependency.FromContext(c).NodePool(c)
		if err != nil {
			c.JSON(200, serializer.NewError(serializer.CodeInternalSetting, "Failed to get node pool", err))
			c.Abort()
			return
		}

		slaveNode, err := np.Get(c, types.NodeCapabilityNone, nodeId, nil)
		if slaveNode == nil || slaveNode.IsMaster() {
			c.JSON(200, serializer.ParamErr(c, "Unknown node ID", err))
			c.Abort()
			return
		}

		SignRequired(slaveNode.AuthInstance())(c)
	}
}
