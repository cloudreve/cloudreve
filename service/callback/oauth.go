package callback

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

// OauthService OAuth 存储策略授权回调服务
type OauthService struct {
	Code     string `form:"code"`
	Error    string `form:"error"`
	ErrorMsg string `form:"error_description"`
	Scope    string `form:"scope"`
}

//
//// GDriveAuth Google Drive 更新认证信息
//func (service *OauthService) GDriveAuth(c *gin.Context) serializer.Response {
//	if service.Error != "" {
//	}
//
//	// validate required scope
//	if missing, found := lo.Find[string](googledrive.RequiredScope, func(item string) bool {
//		return !strings.Contains(service.Scope, item)
//	}); found {
//	}
//
//	policyID, ok := util.GetSession(c, "googledrive_oauth_policy").(uint)
//	if !ok {
//	}
//
//	util.DeleteSession(c, "googledrive_oauth_policy")
//
//	policy, err := model.GetPolicyByID(policyID)
//	if err != nil {
//	}
//
//	client, err := googledrive.NewClient(&policy)
//	if err != nil {
//	}
//
//	credential, err := client.ObtainToken(c, service.Code, "")
//	if err != nil {
//	}
//
//	// 更新存储策略的 RefreshToken
//	client.Policy.AccessKey = credential.RefreshToken
//	if err := client.Policy.SaveAndClearCache(); err != nil {
//	}
//
//	cache.Deletes([]string{client.Policy.AccessKey}, googledrive.TokenCachePrefix)
//	return serializer.Response{}
//}

// OdAuth OneDrive 更新认证信息
func (service *OauthService) OdAuth(c *gin.Context) serializer.Response {
	//if service.Error != "" {
	//}
	//
	//policyID, ok := util.GetSession(c, "onedrive_oauth_policy").(uint)
	//if !ok {
	//}
	//
	//util.DeleteSession(c, "onedrive_oauth_policy")
	//
	//policy, err := model.GetPolicyByID(policyID)
	//if err != nil {
	//}
	//
	//client, err := onedrive.NewClient(&policy)
	//if err != nil {
	//}
	//
	//credential, err := client.ObtainToken(c, onedrive.WithCode(service.Code))
	//if err != nil {
	//}
	//
	//// 更新存储策略的 RefreshToken
	//client.Policy.AccessKey = credential.RefreshToken
	//if err := client.Policy.SaveAndClearCache(); err != nil {
	//}
	//
	//cache.Deletes([]string{client.Policy.AccessKey}, "onedrive_")
	//if client.Policy.OptionsSerialized.OdDriver != "" && strings.Contains(client.Policy.OptionsSerialized.OdDriver, "http") {
	//	if err := querySharePointSiteID(c, client.Policy); err != nil {
	//	}
	//}

	return serializer.Response{}
}

//func querySharePointSiteID(ctx context.Context, policy *model.Policy) error {
//client, err := onedrive.NewClient(policy)
//if err != nil {
//	return err
//}
//
//id, err := client.GetSiteIDByURL(ctx, client.Policy.OptionsSerialized.OdDriver)
//if err != nil {
//	return err
//}
//
//client.Policy.OptionsSerialized.OdDriver = fmt.Sprintf("sites/%s/drive", id)
//if err := client.Policy.SaveAndClearCache(); err != nil {
//	return err
//}

//return nil
//}
