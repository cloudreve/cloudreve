package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/node"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader/slave"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

const (
	nodeStatusCondition = "node_status"
)

func (service *AdminListService) Nodes(c *gin.Context) (*ListNodeResponse, error) {
	dep := dependency.FromContext(c)
	nodeClient := dep.NodeClient()

	ctx := context.WithValue(c, inventory.LoadNodeStoragePolicy{}, true)
	res, err := nodeClient.ListNodes(ctx, &inventory.ListNodeParameters{
		PaginationArgs: &inventory.PaginationArgs{
			Page:     service.Page - 1,
			PageSize: service.PageSize,
			OrderBy:  service.OrderBy,
			Order:    inventory.OrderDirection(service.OrderDirection),
		},
		Status: node.Status(service.Conditions[nodeStatusCondition]),
	})

	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list nodes", err)
	}

	return &ListNodeResponse{Nodes: res.Nodes, Pagination: res.PaginationResults}, nil
}

type (
	SingleNodeService struct {
		ID int `uri:"id" json:"id" binding:"required"`
	}
	SingleNodeParamCtx struct{}
)

func (service *SingleNodeService) Get(c *gin.Context) (*GetNodeResponse, error) {
	dep := dependency.FromContext(c)
	nodeClient := dep.NodeClient()

	ctx := context.WithValue(c, inventory.LoadNodeStoragePolicy{}, true)
	node, err := nodeClient.GetNodeById(ctx, service.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get node", err)
	}

	return &GetNodeResponse{Node: node}, nil
}

type (
	TestNodeService struct {
		Node *ent.Node `json:"node" binding:"required"`
	}
	TestNodeParamCtx struct{}
)

func (service *TestNodeService) Test(c *gin.Context) error {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()

	slave, err := url.Parse(service.Node.Server)
	if err != nil {
		return serializer.NewError(serializer.CodeParamErr, "Failed to parse node URL", err)
	}

	primaryURL := settings.SiteURL(setting.UseFirstSiteUrl(c)).String()
	body := map[string]string{
		"callback": primaryURL,
	}
	bodyByte, _ := json.Marshal(body)

	r := dep.RequestClient()
	res, err := r.Request(
		"POST",
		routes.SlavePingRoute(slave),
		bytes.NewReader(bodyByte),
		request.WithTimeout(time.Duration(10)*time.Second),
		request.WithCredential(
			auth.HMACAuth{SecretKey: []byte(service.Node.SlaveKey)},
			int64(settings.SlaveRequestSignTTL(c)),
		),
		request.WithSlaveMeta(int(service.Node.ID)),
		request.WithMasterMeta(settings.SiteBasic(c).ID, primaryURL),
		request.WithCorrelationID(),
	).CheckHTTPResponse(http.StatusOK).DecodeResponse()

	if err != nil {
		return serializer.NewError(serializer.CodeParamErr, "Failed to connect to node: "+err.Error(), nil)
	}

	if res.Code != 0 {
		return serializer.NewError(serializer.CodeParamErr, "Successfully connected to slave node, but slave returns: "+res.Msg, nil)
	}

	return nil
}

type (
	TestNodeDownloaderService struct {
		Node *ent.Node `json:"node" binding:"required"`
	}
	TestNodeDownloaderParamCtx struct{}
)

func (service *TestNodeDownloaderService) Test(c *gin.Context) (string, error) {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()
	var (
		dl  downloader.Downloader
		err error
	)
	if service.Node.Type == node.TypeMaster {
		dl, err = cluster.NewDownloader(c, dep.RequestClient(request.WithContext(c)), dep.SettingProvider(), service.Node.Settings)
	} else {
		dl = slave.NewSlaveDownloader(dep.RequestClient(
			request.WithContext(c),
			request.WithCorrelationID(),
			request.WithSlaveMeta(service.Node.ID),
			request.WithMasterMeta(settings.SiteBasic(c).ID, settings.SiteURL(setting.UseFirstSiteUrl(c)).String()),
			request.WithCredential(auth.HMACAuth{SecretKey: []byte(service.Node.SlaveKey)}, int64(settings.SlaveRequestSignTTL(c))),
			request.WithEndpoint(service.Node.Server),
		), service.Node.Settings)
	}

	if err != nil {
		return "", serializer.NewError(serializer.CodeParamErr, "Failed to create downloader", err)
	}

	version, err := dl.Test(c)
	if err != nil {
		return "", serializer.NewError(serializer.CodeParamErr, "Failed to test downloader: "+err.Error(), nil)
	}

	return version, nil
}

type (
	UpsertNodeService struct {
		Node *ent.Node `json:"node" binding:"required"`
	}
	UpsertNodeParamCtx struct{}
)

func (s *UpsertNodeService) Update(c *gin.Context) (*GetNodeResponse, error) {
	dep := dependency.FromContext(c)
	nodeClient := dep.NodeClient()

	if s.Node.ID == 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "ID is required", nil)
	}

	node, err := nodeClient.Upsert(c, s.Node)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to update node", err)
	}

	// reload node pool
	np, err := dep.NodePool(c)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to get node pool", err)
	}
	np.Upsert(c, node)

	// Clear policy cache since some this node maybe cached by some storage policy
	kv := dep.KV()
	kv.Delete(inventory.StoragePolicyCacheKey)

	service := &SingleNodeService{ID: node.ID}
	return service.Get(c)
}

func (s *UpsertNodeService) Create(c *gin.Context) (*GetNodeResponse, error) {
	dep := dependency.FromContext(c)
	nodeClient := dep.NodeClient()

	if s.Node.ID > 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "ID must be 0", nil)
	}

	node, err := nodeClient.Upsert(c, s.Node)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create node", err)
	}

	// reload node pool
	np, err := dep.NodePool(c)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to get node pool", err)
	}
	np.Upsert(c, node)

	service := &SingleNodeService{ID: node.ID}
	return service.Get(c)
}

func (s *SingleNodeService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	nodeClient := dep.NodeClient()

	ctx := context.WithValue(c, inventory.LoadNodeStoragePolicy{}, true)
	existing, err := nodeClient.GetNodeById(ctx, s.ID)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to get node", err)
	}

	if existing.Type == node.TypeMaster {
		return serializer.NewError(serializer.CodeInvalidActionOnSystemNode, "", nil)
	}

	if len(existing.Edges.StoragePolicy) > 0 {
		return serializer.NewError(
			serializer.CodeNodeUsedByStoragePolicy,
			strings.Join(lo.Map(existing.Edges.StoragePolicy, func(i *ent.StoragePolicy, _ int) string {
				return i.Name
			}), ", "),
			nil,
		)
	}

	// insert dummpy disabled node in nodepool to evict it
	disabledNode := &ent.Node{
		ID:     s.ID,
		Type:   node.TypeSlave,
		Status: node.StatusSuspended,
	}
	np, err := dep.NodePool(c)
	if err != nil {
		return serializer.NewError(serializer.CodeInternalSetting, "Failed to get node pool", err)
	}
	np.Upsert(c, disabledNode)
	return nodeClient.Delete(c, s.ID)
}
