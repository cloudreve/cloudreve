package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/admin"
	"github.com/gin-gonic/gin"
)

// AdminSummary 获取管理站点概况
func AdminSummary(c *gin.Context) {
	service := ParametersFromContext[*admin.SummaryService](c, admin.SummaryParamCtx{})
	res, err := service.Summary(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// AdminGetSettings 获取站点设定项
func AdminGetSettings(c *gin.Context) {
	service := ParametersFromContext[*admin.GetSettingService](c, admin.GetSettingParamCtx{})
	res, err := service.GetSetting(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

func AdminSetSettings(c *gin.Context) {
	service := ParametersFromContext[*admin.SetSettingService](c, admin.SetSettingParamCtx{})
	res, err := service.SetSetting(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// AdminListGroups 获取用户组列表
func AdminListGroups(c *gin.Context) {
	service := ParametersFromContext[*admin.AdminListService](c, admin.AdminListServiceParamsCtx{})
	res, err := service.List(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

func AdminFetchWopi(c *gin.Context) {
	service := ParametersFromContext[*admin.FetchWOPIDiscoveryService](c, admin.FetchWOPIDiscoveryParamCtx{})
	res, err := service.Fetch(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// AdminTestThumbGenerator Tests thumb generator
func AdminTestThumbGenerator(c *gin.Context) {
	service := ParametersFromContext[*admin.ThumbGeneratorTestService](c, admin.ThumbGeneratorTestParamCtx{})
	res, err := service.Test(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

func AdminGetQueueMetrics(c *gin.Context) {
	res, err := admin.GetQueueMetrics(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminListPolicies(c *gin.Context) {
	service := ParametersFromContext[*admin.AdminListService](c, admin.AdminListServiceParamsCtx{})
	res, err := service.Policies(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminGetPolicy(c *gin.Context) {
	service := ParametersFromContext[*admin.SingleStoragePolicyService](c, admin.GetStoragePolicyParamCtx{})
	res, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// AdminSendTestMail 发送测试邮件
func AdminSendTestMail(c *gin.Context) {
	service := ParametersFromContext[*admin.TestSMTPService](c, admin.TestSMTPParamCtx{})
	err := service.Test(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{})
}

func AdminCreatePolicy(c *gin.Context) {
	service := ParametersFromContext[*admin.CreateStoragePolicyService](c, admin.CreateStoragePolicyParamCtx{})
	res, err := service.Create(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminUpdatePolicy(c *gin.Context) {
	service := ParametersFromContext[*admin.UpdateStoragePolicyService](c, admin.UpdateStoragePolicyParamCtx{})
	res, err := service.Update(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminListNodes(c *gin.Context) {
	service := ParametersFromContext[*admin.AdminListService](c, admin.AdminListServiceParamsCtx{})
	res, err := service.Nodes(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminGetNode(c *gin.Context) {
	service := ParametersFromContext[*admin.SingleNodeService](c, admin.SingleNodeParamCtx{})
	res, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminClearEntityUrlCache(c *gin.Context) {
	admin.ClearEntityUrlCache(c)
	c.JSON(200, serializer.Response{})
}

func AdminCreateStoragePolicyCors(c *gin.Context) {
	service := ParametersFromContext[*admin.CreateStoragePolicyCorsService](c, admin.CreateStoragePolicyCorsParamCtx{})
	err := service.Create(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{})
}

func AdminOdOAuthURL(c *gin.Context) {
	service := ParametersFromContext[*admin.GetOauthRedirectService](c, admin.GetOauthRedirectParamCtx{})
	res, err := service.GetOAuth(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminGetPolicyOAuthCallbackURL(c *gin.Context) {
	res := admin.GetPolicyOAuthURL(c)
	c.JSON(200, serializer.Response{Data: res})
}

func AdminGetPolicyOAuthStatus(c *gin.Context) {
	service := ParametersFromContext[*admin.SingleStoragePolicyService](c, admin.GetStoragePolicyParamCtx{})
	res, err := service.GetOauthCredentialStatus(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminFinishOauthCallback(c *gin.Context) {
	service := ParametersFromContext[*admin.FinishOauthCallbackService](c, admin.FinishOauthCallbackParamCtx{})
	err := service.Finish(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{})
}

func AdminGetSharePointDriverRoot(c *gin.Context) {
	service := ParametersFromContext[*admin.SingleStoragePolicyService](c, admin.GetStoragePolicyParamCtx{})
	res, err := service.GetSharePointDriverRoot(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminDeletePolicy(c *gin.Context) {
	service := ParametersFromContext[*admin.SingleStoragePolicyService](c, admin.GetStoragePolicyParamCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{})
}

func AdminGetGroup(c *gin.Context) {
	service := ParametersFromContext[*admin.SingleGroupService](c, admin.SingleGroupParamCtx{})
	res, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminCreateGroup(c *gin.Context) {
	service := ParametersFromContext[*admin.UpsertGroupService](c, admin.UpsertGroupParamCtx{})
	res, err := service.Create(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminUpdateGroup(c *gin.Context) {
	service := ParametersFromContext[*admin.UpsertGroupService](c, admin.UpsertGroupParamCtx{})
	res, err := service.Update(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminListUsers(c *gin.Context) {
	service := ParametersFromContext[*admin.AdminListService](c, admin.AdminListServiceParamsCtx{})
	res, err := service.Users(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminGetUser(c *gin.Context) {
	service := ParametersFromContext[*admin.SingleUserService](c, admin.SingleUserParamCtx{})
	res, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminUpdateUser(c *gin.Context) {
	service := ParametersFromContext[*admin.UpsertUserService](c, admin.UpsertUserParamCtx{})
	res, err := service.Update(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminCreateUser(c *gin.Context) {
	service := ParametersFromContext[*admin.UpsertUserService](c, admin.UpsertUserParamCtx{})
	res, err := service.Create(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

//	func AdminHashIDEncode(c *gin.Context) {
//		service := ParametersFromContext[*admin.HashIDService](c, admin.HashIDParamCtx{})
//		resp, err := service.Encode(c)
//		if err != nil {
//			c.JSON(200, serializer.Err(c, err))
//			c.Abort()
//			return
//		}
//
//		c.JSON(200, serializer.Response{
//			Data: resp,
//		})
//	}
//
//	func AdminHashIDDecode(c *gin.Context) {
//		service := ParametersFromContext[*admin.HashIDService](c, admin.HashIDParamCtx{})
//		resp, err := service.Decode(c)
//		if err != nil {
//			c.JSON(200, serializer.Err(c, err))
//			c.Abort()
//			return
//		}
//
//		c.JSON(200, serializer.Response{
//			Data: resp,
//		})
//	}
//
//	func AdminBsEncode(c *gin.Context) {
//		service := ParametersFromContext[*admin.BsEncodeService](c, admin.BsEncodeParamCtx{})
//		resp, err := service.Encode(c)
//		if err != nil {
//			c.JSON(200, serializer.Err(c, err))
//			c.Abort()
//			return
//		}
//
//		c.JSON(200, serializer.Response{
//			Data: resp,
//		})
//	}
//
//	func AdminBsDecode(c *gin.Context) {
//		service := ParametersFromContext[*admin.BsDecodeService](c, admin.BsDecodeParamCtx{})
//		resp, err := service.Decode(c)
//		if err != nil {
//			c.JSON(200, serializer.Err(c, err))
//			c.Abort()
//			return
//		}
//
//		c.JSON(200, serializer.Response{
//			Data: resp,
//		})
//	}
//

func AdminDeleteGroup(c *gin.Context) {
	service := ParametersFromContext[*admin.SingleGroupService](c, admin.SingleGroupParamCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{})
}

// AdminTestSlave 测试从机可用性
func AdminTestSlave(c *gin.Context) {
	service := ParametersFromContext[*admin.TestNodeService](c, admin.TestNodeParamCtx{})
	err := service.Test(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{})
}

// AdminTestDownloader 测试下载器连接
func AdminTestDownloader(c *gin.Context) {
	service := ParametersFromContext[*admin.TestNodeDownloaderService](c, admin.TestNodeDownloaderParamCtx{})
	res, err := service.Test(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminCreateNode(c *gin.Context) {
	service := ParametersFromContext[*admin.UpsertNodeService](c, admin.UpsertNodeParamCtx{})
	res, err := service.Create(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminUpdateNode(c *gin.Context) {
	service := ParametersFromContext[*admin.UpsertNodeService](c, admin.UpsertNodeParamCtx{})
	res, err := service.Update(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminDeleteNode(c *gin.Context) {
	service := ParametersFromContext[*admin.SingleNodeService](c, admin.SingleNodeParamCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{})
}

// AdminDeleteUser 批量删除用户
func AdminDeleteUser(c *gin.Context) {
	service := ParametersFromContext[*admin.BatchUserService](c, admin.BatchUserParamCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{})
}

func AdminListFiles(c *gin.Context) {
	service := ParametersFromContext[*admin.AdminListService](c, admin.AdminListServiceParamsCtx{})
	res, err := service.Files(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminGetFile(c *gin.Context) {
	service := ParametersFromContext[*admin.SingleFileService](c, admin.SingleFileParamCtx{})
	res, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminUpdateFile(c *gin.Context) {
	service := ParametersFromContext[*admin.UpsertFileService](c, admin.UpsertFileParamCtx{})
	res, err := service.Update(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminGetFileUrl(c *gin.Context) {
	service := ParametersFromContext[*admin.SingleFileService](c, admin.SingleFileParamCtx{})
	res, err := service.Url(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminBatchDeleteFile(c *gin.Context) {
	service := ParametersFromContext[*admin.BatchFileService](c, admin.BatchFileParamCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{})
}

func AdminListEntities(c *gin.Context) {
	service := ParametersFromContext[*admin.AdminListService](c, admin.AdminListServiceParamsCtx{})
	res, err := service.Entities(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

func AdminGetEntity(c *gin.Context) {
	service := ParametersFromContext[*admin.SingleEntityService](c, admin.SingleEntityParamCtx{})
	res, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminGetEntityUrl(c *gin.Context) {
	service := ParametersFromContext[*admin.SingleEntityService](c, admin.SingleEntityParamCtx{})
	res, err := service.Url(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminBatchDeleteEntity(c *gin.Context) {
	service := ParametersFromContext[*admin.BatchEntityService](c, admin.BatchEntityParamCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
}

func AdminCleanupTask(c *gin.Context) {
	service := ParametersFromContext[*admin.CleanupTaskService](c, admin.CleanupTaskParameterCtx{})
	err := service.CleanupTask(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{})
}

func AdminListTasks(c *gin.Context) {
	service := ParametersFromContext[*admin.AdminListService](c, admin.AdminListServiceParamsCtx{})
	res, err := service.Tasks(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminGetTask(c *gin.Context) {
	service := ParametersFromContext[*admin.SingleTaskService](c, admin.SingleTaskParamCtx{})
	res, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminBatchDeleteTask(c *gin.Context) {
	service := ParametersFromContext[*admin.BatchTaskService](c, admin.BatchTaskParamCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{})
}

func AdminListShares(c *gin.Context) {
	service := ParametersFromContext[*admin.AdminListService](c, admin.AdminListServiceParamsCtx{})
	res, err := service.Shares(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminGetShare(c *gin.Context) {
	service := ParametersFromContext[*admin.SingleShareService](c, admin.SingleShareParamCtx{})
	res, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

func AdminBatchDeleteShare(c *gin.Context) {
	service := ParametersFromContext[*admin.BatchShareService](c, admin.BatchShareParamCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
}

func AdminCalibrateStorage(c *gin.Context) {
	service := ParametersFromContext[*admin.SingleUserService](c, admin.SingleUserParamCtx{})
	res, err := service.CalibrateStorage(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}
