package admin

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/credmanager"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/cos"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/ks3"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/obs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/onedrive"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/oss"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/s3"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"

	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

// PathTestService 本地路径测试服务
type PathTestService struct {
	Path string `json:"path" binding:"required"`
}

// SlaveTestService 从机测试服务
type SlaveTestService struct {
	Secret string `json:"secret" binding:"required"`
	Server string `json:"server" binding:"required"`
}

type (
	SlavePingParameterCtx struct{}
	// SlavePingService ping slave node
	SlavePingService struct {
		Callback string `json:"callback" binding:"required"`
	}
)

// AddPolicyService 存储策略添加服务
type AddPolicyService struct {
	//Policy model.Policy `json:"policy" binding:"required"`
}

// PolicyService 存储策略ID服务
type PolicyService struct {
	ID     uint   `uri:"id" json:"id" binding:"required"`
	Region string `json:"region"`
}

// Delete 删除存储策略
func (service *SingleStoragePolicyService) Delete(c *gin.Context) error {
	// 禁止删除默认策略
	if service.ID == 1 {
		return serializer.NewError(serializer.CodeDeleteDefaultPolicy, "", nil)
	}

	dep := dependency.FromContext(c)
	storagePolicyClient := dep.StoragePolicyClient()

	ctx := context.WithValue(c, inventory.LoadStoragePolicyGroup{}, true)
	ctx = context.WithValue(ctx, inventory.SkipStoragePolicyCache{}, true)
	policy, err := storagePolicyClient.GetPolicyByID(ctx, service.ID)
	if err != nil {
		return serializer.NewError(serializer.CodePolicyNotExist, "", err)
	}

	// If policy is used by groups, return error
	if len(policy.Edges.Groups) > 0 {
		return serializer.NewError(serializer.CodePolicyUsedByGroups, strconv.Itoa(len(policy.Edges.Groups)), nil)
	}

	used, err := dep.FileClient().IsStoragePolicyUsedByEntities(ctx, service.ID)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to check if policy is used by entities", err)
	}

	if used {
		return serializer.NewError(serializer.CodePolicyUsedByFiles, "", nil)
	}

	err = storagePolicyClient.Delete(ctx, policy)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete policy", err)
	}

	return nil
}

// Test 从机响应ping
func (service *SlavePingService) Test(c *gin.Context) error {
	master, err := url.Parse(service.Callback)
	if err != nil {
		return serializer.NewError(serializer.CodeParamErr, "Failed to parse callback url", err)
	}

	dep := dependency.FromContext(c)
	r := dep.RequestClient()
	res, err := r.Request(
		"GET",
		routes.MasterPingUrl(master).String(),
		nil,
		request.WithContext(c),
		request.WithLogger(logging.FromContext(c)),
		request.WithCorrelationID(),
		request.WithTimeout(time.Duration(10)*time.Second),
	).DecodeResponse()

	if err != nil {
		return serializer.NewError(serializer.CodeSlavePingMaster, err.Error(), nil)
	}

	version := constants.BackendVersion

	if strings.TrimSuffix(res.Data.(string), "-pro") != version {
		return serializer.NewError(serializer.CodeVersionMismatch, "Master: "+res.Data.(string)+", Slave: "+version, nil)
	}

	return nil
}

// Test 测试从机通信
func (service *SlaveTestService) Test() serializer.Response {
	//slave, err := url.Parse(service.Server)
	//if err != nil {
	//	return serializer.ParamErrDeprecated("Failed to parse slave node server URL: "+err.Error(), nil)
	//}
	//
	//controller, _ := url.Parse("/api/v3/slave/ping")
	//
	//// 请求正文
	//body := map[string]string{
	//	"callback": model.GetSiteURL().String(),
	//}
	//bodyByte, _ := json.Marshal(body)
	//
	//r := request.NewClientDeprecated()
	//res, err := r.Request(
	//	"POST",
	//	slave.ResolveReference(controller).String(),
	//	bytes.NewReader(bodyByte),
	//	request.WithTimeout(time.Duration(10)*time.Second),
	//	request.WithCredential(
	//		auth.HMACAuth{SecretKey: []byte(service.Secret)},
	//		int64(model.GetIntSetting("slave_api_timeout", 60)),
	//	),
	//).DecodeResponse()
	//if err != nil {
	//	return serializer.ParamErrDeprecated("Failed to connect to slave node: "+err.Error(), nil)
	//}
	//
	//if res.Code != 0 {
	//	return serializer.ParamErrDeprecated("Successfully connected to slave node, but slave returns: "+res.Msg, nil)
	//}

	return serializer.Response{}
}

// Test 测试本地路径
func (service *PathTestService) Test() serializer.Response {
	//policy := model.Policy{DirNameRule: service.Path}
	//path := policy.GeneratePath(1, "/My File")
	//path = filepath.Join(path, "test.txt")
	//file, err := util.CreatNestedFile(util.RelativePath(path))
	//if err != nil {
	//	return serializer.ParamErrDeprecated(fmt.Sprintf("Failed to create \"%s\": %s", path, err.Error()), nil)
	//}
	//
	//file.Close()
	//os.Remove(path)

	return serializer.Response{}
}

const (
	policyTypeCondition = "policy_type"
)

// Policies 列出存储策略
func (service *AdminListService) Policies(c *gin.Context) (*ListPolicyResponse, error) {
	dep := dependency.FromContext(c)
	storagePolicyClient := dep.StoragePolicyClient()

	ctx := context.WithValue(c, inventory.LoadStoragePolicyGroup{}, true)
	res, err := storagePolicyClient.ListPolicies(ctx, &inventory.ListPolicyParameters{
		PaginationArgs: &inventory.PaginationArgs{
			Page:     service.Page - 1,
			PageSize: service.PageSize,
			OrderBy:  service.OrderBy,
			Order:    inventory.OrderDirection(service.OrderDirection),
		},
		Type: types.PolicyType(service.Conditions[policyTypeCondition]),
	})

	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list policies", err)
	}

	return &ListPolicyResponse{
		Pagination: res.PaginationResults,
		Policies:   res.Policies,
	}, nil
}

type (
	SingleStoragePolicyService struct {
		ID int `uri:"id" json:"id" binding:"required"`
	}
	GetStoragePolicyParamCtx struct{}
)

const (
	countEntityQuery = "countEntity"
)

func (service *SingleStoragePolicyService) Get(c *gin.Context) (*GetStoragePolicyResponse, error) {
	dep := dependency.FromContext(c)
	storagePolicyClient := dep.StoragePolicyClient()

	ctx := context.WithValue(c, inventory.LoadStoragePolicyGroup{}, true)
	ctx = context.WithValue(ctx, inventory.SkipStoragePolicyCache{}, true)
	policy, err := storagePolicyClient.GetPolicyByID(ctx, service.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get policy", err)
	}

	res := &GetStoragePolicyResponse{StoragePolicy: policy}
	if c.Query(countEntityQuery) != "" {
		count, size, err := dep.FileClient().CountEntityByStoragePolicyID(ctx, service.ID)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to count entities", err)
		}
		res.EntitiesCount = count
		res.EntitiesSize = size
	}

	return res, nil
}

type (
	CreateStoragePolicyService struct {
		Policy *ent.StoragePolicy `json:"policy" binding:"required"`
	}
	CreateStoragePolicyParamCtx struct{}
)

func (service *CreateStoragePolicyService) Create(c *gin.Context) (*GetStoragePolicyResponse, error) {
	dep := dependency.FromContext(c)
	storagePolicyClient := dep.StoragePolicyClient()

	if service.Policy.Type == types.PolicyTypeLocal {
		service.Policy.DirNameRule = util.DataPath("uploads/{uid}/{path}")
	}

	service.Policy.ID = 0
	policy, err := storagePolicyClient.Upsert(c, service.Policy)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create policy", err)
	}

	return &GetStoragePolicyResponse{StoragePolicy: policy}, nil
}

type (
	UpdateStoragePolicyService struct {
		Policy *ent.StoragePolicy `json:"policy" binding:"required"`
	}
	UpdateStoragePolicyParamCtx struct{}
)

func (service *UpdateStoragePolicyService) Update(c *gin.Context) (*GetStoragePolicyResponse, error) {
	dep := dependency.FromContext(c)
	storagePolicyClient := dep.StoragePolicyClient()

	id := c.Param("id")
	if id == "" {
		return nil, serializer.NewError(serializer.CodeParamErr, "ID is required", nil)
	}
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid ID", err)
	}

	service.Policy.ID = idInt

	sc, tx, ctx, err := inventory.WithTx(c, storagePolicyClient)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create transaction", err)
	}

	_, err = sc.Upsert(ctx, service.Policy)
	if err != nil {
		_ = inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to update policy", err)
	}

	if err := inventory.Commit(tx); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to commit transaction", err)
	}

	_ = dep.KV().Delete(manager.EntityUrlCacheKeyPrefix)

	s := SingleStoragePolicyService{ID: idInt}
	return s.Get(c)
}

type (
	CreateStoragePolicyCorsService struct {
		Policy *ent.StoragePolicy `json:"policy" binding:"required"`
	}
	CreateStoragePolicyCorsParamCtx struct{}
)

func (service *CreateStoragePolicyCorsService) Create(c *gin.Context) error {
	dep := dependency.FromContext(c)

	switch service.Policy.Type {
	case types.PolicyTypeOss:
		handler, err := oss.New(c, service.Policy, dep.SettingProvider(), dep.ConfigProvider(), dep.Logger(), dep.MimeDetector(c))
		if err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to create oss driver", err)
		}
		if err := handler.CORS(); err != nil {
			return serializer.NewError(serializer.CodeInternalSetting, "Failed to create cors: "+err.Error(), err)
		}

		return nil

	case types.PolicyTypeCos:
		handler, err := cos.New(c, service.Policy, dep.SettingProvider(), dep.ConfigProvider(), dep.Logger(), dep.MimeDetector(c))
		if err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to create cos driver", err)
		}

		if err := handler.CORS(); err != nil {
			return serializer.NewError(serializer.CodeInternalSetting, "Failed to create cors: "+err.Error(), err)
		}

		return nil

	case types.PolicyTypeS3:
		handler, err := s3.New(c, service.Policy, dep.SettingProvider(), dep.ConfigProvider(), dep.Logger(), dep.MimeDetector(c))
		if err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to create s3 driver", err)
		}

		if err := handler.CORS(); err != nil {
			return serializer.NewError(serializer.CodeInternalSetting, "Failed to create cors: "+err.Error(), err)
		}

		return nil

	case types.PolicyTypeKs3:
		handler, err := ks3.New(c, service.Policy, dep.SettingProvider(), dep.ConfigProvider(), dep.Logger(), dep.MimeDetector(c))
		if err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to create ks3 driver", err)
		}

		if err := handler.CORS(); err != nil {
			return serializer.NewError(serializer.CodeInternalSetting, "Failed to create cors: "+err.Error(), err)
		}

		return nil
	case types.PolicyTypeObs:
		handler, err := obs.New(c, service.Policy, dep.SettingProvider(), dep.ConfigProvider(), dep.Logger(), dep.MimeDetector(c))
		if err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to create obs driver", err)
		}

		if err := handler.CORS(); err != nil {
			return serializer.NewError(serializer.CodeInternalSetting, "Failed to create cors: "+err.Error(), err)
		}

		return nil
	default:
		return serializer.NewError(serializer.CodeParamErr, "Unsupported policy type", nil)
	}
}

type (
	GetOauthRedirectService struct {
		ID     int    `json:"id" binding:"required"`
		Secret string `json:"secret" binding:"required"`
		AppID  string `json:"app_id" binding:"required"`
	}
	GetOauthRedirectParamCtx struct{}
)

// GetOAuth 获取 OneDrive OAuth 地址
func (service *GetOauthRedirectService) GetOAuth(c *gin.Context) (string, error) {
	dep := dependency.FromContext(c)
	storagePolicyClient := dep.StoragePolicyClient()

	policy, err := storagePolicyClient.GetPolicyByID(c, service.ID)
	if err != nil || policy.Type != types.PolicyTypeOd {
		return "", serializer.NewError(serializer.CodePolicyNotExist, "", nil)
	}

	// Update to latest redirect url
	policy.Settings.OauthRedirect = routes.MasterPolicyOAuthCallback(dep.SettingProvider().SiteURL(c)).String()
	policy.SecretKey = service.Secret
	policy.BucketName = service.AppID
	policy, err = storagePolicyClient.Upsert(c, policy)
	if err != nil {
		return "", serializer.NewError(serializer.CodeDBError, "Failed to update policy", err)
	}

	client := onedrive.NewClient(policy, dep.RequestClient(), dep.CredManager(), dep.Logger(), dep.SettingProvider(), 0)
	redirect := client.OAuthURL(context.Background(), []string{
		"offline_access",
		"files.readwrite.all",
	})

	return redirect, nil
}

func GetPolicyOAuthURL(c *gin.Context) string {
	dep := dependency.FromContext(c)
	return routes.MasterPolicyOAuthCallback(dep.SettingProvider().SiteURL(c)).String()
}

// GetOauthCredentialStatus returns last refresh time of oauth credential
func (service *SingleStoragePolicyService) GetOauthCredentialStatus(c *gin.Context) (*OauthCredentialStatus, error) {
	dep := dependency.FromContext(c)
	storagePolicyClient := dep.StoragePolicyClient()

	policy, err := storagePolicyClient.GetPolicyByID(c, service.ID)
	if err != nil || policy.Type != types.PolicyTypeOd {
		return nil, serializer.NewError(serializer.CodePolicyNotExist, "", nil)
	}

	if policy.AccessKey == "" {
		return &OauthCredentialStatus{Valid: false}, nil
	}

	token, err := dep.CredManager().Obtain(c, onedrive.CredentialKey(policy.ID))
	if err != nil {
		if errors.Is(err, credmanager.ErrNotFound) {
			return &OauthCredentialStatus{Valid: false}, nil
		}

		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get credential", err)
	}

	return &OauthCredentialStatus{Valid: true, LastRefreshTime: token.RefreshedAt()}, nil
}

type (
	FinishOauthCallbackService struct {
		Code  string `json:"code" binding:"required"`
		State string `json:"state" binding:"required"`
	}
	FinishOauthCallbackParamCtx struct{}
)

func (service *FinishOauthCallbackService) Finish(c *gin.Context) error {
	dep := dependency.FromContext(c)
	storagePolicyClient := dep.StoragePolicyClient()

	policyId, err := strconv.Atoi(service.State)
	if err != nil {
		return serializer.NewError(serializer.CodeParamErr, "Invalid state", err)
	}

	policy, err := storagePolicyClient.GetPolicyByID(c, policyId)
	if err != nil {
		return serializer.NewError(serializer.CodePolicyNotExist, "", nil)
	}

	if policy.Type != types.PolicyTypeOd {
		return serializer.NewError(serializer.CodeParamErr, "Invalid policy type", nil)
	}

	client := onedrive.NewClient(policy, dep.RequestClient(), dep.CredManager(), dep.Logger(), dep.SettingProvider(), 0)
	credential, err := client.ObtainToken(c, onedrive.WithCode(service.Code))
	if err != nil {
		return serializer.NewError(serializer.CodeParamErr, "Failed to obtain token: "+err.Error(), err)
	}

	credManager := dep.CredManager()
	err = credManager.Upsert(c, credential)
	if err != nil {
		return serializer.NewError(serializer.CodeInternalSetting, "Failed to upsert credential", err)
	}

	_, err = credManager.Obtain(c, onedrive.CredentialKey(policy.ID))
	if err != nil {
		return serializer.NewError(serializer.CodeInternalSetting, "Failed to obtain credential", err)
	}

	return nil
}

func (service *SingleStoragePolicyService) GetSharePointDriverRoot(c *gin.Context) (string, error) {
	dep := dependency.FromContext(c)
	storagePolicyClient := dep.StoragePolicyClient()

	policy, err := storagePolicyClient.GetPolicyByID(c, service.ID)
	if err != nil {
		return "", serializer.NewError(serializer.CodePolicyNotExist, "", nil)
	}

	if policy.Type != types.PolicyTypeOd {
		return "", serializer.NewError(serializer.CodeParamErr, "Invalid policy type", nil)
	}

	client := onedrive.NewClient(policy, dep.RequestClient(), dep.CredManager(), dep.Logger(), dep.SettingProvider(), 0)
	root, err := client.GetSiteIDByURL(c, c.Query("url"))
	if err != nil {
		return "", serializer.NewError(serializer.CodeInternalSetting, "Failed to get site id", err)
	}

	return fmt.Sprintf("sites/%s/drive", root), nil
}
