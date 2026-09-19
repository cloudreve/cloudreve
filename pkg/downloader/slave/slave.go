package slave

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"crypto/sha1"
	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
)

type slaveDownloader struct {
	client          request.Client
	nodeSetting     *types.NodeSetting
	nodeSettingHash string
}

// NewSlaveDownloader creates a new slave downloader
func NewSlaveDownloader(client request.Client, nodeSetting *types.NodeSetting) downloader.Downloader {
	nodeSettingJson, err := json.Marshal(nodeSetting)
	if err != nil {
		nodeSettingJson = []byte{}
	}

	return &slaveDownloader{
		client:          client,
		nodeSetting:     nodeSetting,
		nodeSettingHash: fmt.Sprintf("%x", sha1.Sum(nodeSettingJson)),
	}
}

func (s *slaveDownloader) CreateTask(ctx context.Context, url string, options map[string]interface{}) (*downloader.TaskHandle, error) {
	reqBody, err := json.Marshal(&CreateSlaveDownload{
		NodeSetting:     s.nodeSetting,
		Url:             url,
		Options:         options,
		NodeSettingHash: s.nodeSettingHash,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := s.client.Request(
		"POST",
		constants.APIPrefixSlave+"/download/task",
		bytes.NewReader(reqBody),
		request.WithContext(ctx),
		request.WithLogger(logging.FromContext(ctx)),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return nil, err
	}

	// 处理列取结果
	if resp.Code != 0 {
		return nil, serializer.NewErrorFromResponse(resp)
	}

	var taskHandle *downloader.TaskHandle
	if resp.DecodeGob(&taskHandle); taskHandle != nil {
		return taskHandle, nil
	}

	return nil, fmt.Errorf("unexpected response data: %v", resp.Data)
}

func (s *slaveDownloader) Info(ctx context.Context, handle *downloader.TaskHandle) (*downloader.TaskStatus, error) {
	reqBody, err := json.Marshal(&GetSlaveDownload{
		NodeSetting:     s.nodeSetting,
		Handle:          handle,
		NodeSettingHash: s.nodeSettingHash,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := s.client.Request(
		"POST",
		constants.APIPrefixSlave+"/download/status",
		bytes.NewReader(reqBody),
		request.WithContext(ctx),
		request.WithLogger(logging.FromContext(ctx)),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return nil, err
	}

	// 处理列取结果
	if resp.Code != 0 {
		err = serializer.NewErrorFromResponse(resp)
		if resp.Code == serializer.CodeNotFound {
			return nil, fmt.Errorf("%s (%w)", err.Error(), downloader.ErrTaskNotFount)
		}
		return nil, err
	}

	var taskStatus *downloader.TaskStatus
	if resp.DecodeGob(&taskStatus); taskStatus != nil {
		return taskStatus, nil
	}

	return nil, fmt.Errorf("unexpected response data: %v", resp.Data)
}

func (s *slaveDownloader) Cancel(ctx context.Context, handle *downloader.TaskHandle) error {
	reqBody, err := json.Marshal(&CancelSlaveDownload{
		NodeSetting:     s.nodeSetting,
		Handle:          handle,
		NodeSettingHash: s.nodeSettingHash,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := s.client.Request(
		"POST",
		constants.APIPrefixSlave+"/download/cancel",
		bytes.NewReader(reqBody),
		request.WithContext(ctx),
		request.WithLogger(logging.FromContext(ctx)),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return err
	}

	// 处理列取结果
	if resp.Code != 0 {
		return serializer.NewErrorFromResponse(resp)
	}

	return nil
}

func (s *slaveDownloader) SetFilesToDownload(ctx context.Context, handle *downloader.TaskHandle, args ...*downloader.SetFileToDownloadArgs) error {
	reqBody, err := json.Marshal(&SetSlaveFilesToDownload{
		NodeSetting:     s.nodeSetting,
		Handle:          handle,
		NodeSettingHash: s.nodeSettingHash,
		Args:            args,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := s.client.Request(
		"POST",
		constants.APIPrefixSlave+"/download/select",
		bytes.NewReader(reqBody),
		request.WithContext(ctx),
		request.WithLogger(logging.FromContext(ctx)),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return err
	}

	// 处理列取结果
	if resp.Code != 0 {
		return serializer.NewErrorFromResponse(resp)
	}

	return nil
}

func (s *slaveDownloader) Test(ctx context.Context) (string, error) {
	reqBody, err := json.Marshal(&TestSlaveDownload{
		NodeSetting:     s.nodeSetting,
		NodeSettingHash: s.nodeSettingHash,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := s.client.Request(
		"POST",
		constants.APIPrefixSlave+"/download/test",
		bytes.NewReader(reqBody),
		request.WithContext(ctx),
		request.WithLogger(logging.FromContext(ctx)),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return "", err
	}

	if resp.Code != 0 {
		return "", serializer.NewErrorFromResponse(resp)
	}

	return resp.Data.(string), nil
}

// Slave remote download related
type (
	// Request body for creating tasks on slave node
	CreateSlaveDownload struct {
		NodeSetting     *types.NodeSetting     `json:"node_setting"  binding:"required"`
		NodeSettingHash string                 `json:"node_setting_hash"  binding:"required"`
		Url             string                 `json:"url"  binding:"required"`
		Options         map[string]interface{} `json:"options"`
	}
	// Request body for get download task info from slave node
	GetSlaveDownload struct {
		Handle          *downloader.TaskHandle `json:"handle"  binding:"required"`
		NodeSetting     *types.NodeSetting     `json:"node_setting"  binding:"required"`
		NodeSettingHash string                 `json:"node_setting_hash"  binding:"required"`
	}

	// Request body for cancel download task on slave node
	CancelSlaveDownload struct {
		Handle          *downloader.TaskHandle `json:"handle"  binding:"required"`
		NodeSetting     *types.NodeSetting     `json:"node_setting"  binding:"required"`
		NodeSettingHash string                 `json:"node_setting_hash"  binding:"required"`
	}

	// Request body for selecting files to download on slave node
	SetSlaveFilesToDownload struct {
		Handle          *downloader.TaskHandle              `json:"handle"  binding:"required"`
		Args            []*downloader.SetFileToDownloadArgs `json:"args"  binding:"required"`
		NodeSettingHash string                              `json:"node_setting_hash"  binding:"required"`
		NodeSetting     *types.NodeSetting                  `json:"node_setting"  binding:"required"`
	}

	TestSlaveDownload struct {
		NodeSetting     *types.NodeSetting `json:"node_setting"  binding:"required"`
		NodeSettingHash string             `json:"node_setting_hash"  binding:"required"`
	}
)

// GetNodeSetting implements SlaveNodeSettingGetter interface
func (d *CreateSlaveDownload) GetNodeSetting() (*types.NodeSetting, string) {
	return d.NodeSetting, d.NodeSettingHash
}

// GetNodeSetting implements SlaveNodeSettingGetter interface
func (d *GetSlaveDownload) GetNodeSetting() (*types.NodeSetting, string) {
	return d.NodeSetting, d.NodeSettingHash
}

// GetNodeSetting implements SlaveNodeSettingGetter interface
func (d *CancelSlaveDownload) GetNodeSetting() (*types.NodeSetting, string) {
	return d.NodeSetting, d.NodeSettingHash
}

// GetNodeSetting implements SlaveNodeSettingGetter interface
func (d *SetSlaveFilesToDownload) GetNodeSetting() (*types.NodeSetting, string) {
	return d.NodeSetting, d.NodeSettingHash
}

// GetNodeSetting implements SlaveNodeSettingGetter interface
func (d *TestSlaveDownload) GetNodeSetting() (*types.NodeSetting, string) {
	return d.NodeSetting, d.NodeSettingHash
}
