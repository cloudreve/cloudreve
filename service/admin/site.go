package admin

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/thumb"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

func init() {
	gob.Register(map[string]interface{}{})
	gob.Register(map[string]string{})
}

// NoParamService 无需参数的服务
type NoParamService struct {
}

// BatchSettingChangeService 设定批量更改服务
type BatchSettingChangeService struct {
	Options []SettingChangeService `json:"options"`
}

// SettingChangeService  设定更改服务
type SettingChangeService struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value"`
}

// Change 批量更改站点设定
func (service *BatchSettingChangeService) Change() serializer.Response {
	//cacheClean := make([]string, 0, len(service.Options))
	//tx := model.DB.Begin()
	//
	//for _, setting := range service.Options {
	//
	//	if err := tx.Model(&model.Setting{}).Where("name = ?", setting.Key).Update("value", setting.Value).Error; err != nil {
	//		cache.Deletes(cacheClean, "setting_")
	//		tx.Rollback()
	//		return serializer.ErrDeprecated(serializer.CodeUpdateSetting, "Setting "+setting.Key+" failed to update", err)
	//	}
	//
	//	cacheClean = append(cacheClean, setting.Key)
	//}
	//
	//if err := tx.Commit().Error; err != nil {
	//	return serializer.DBErrDeprecated("Failed to update setting", err)
	//}
	//
	//cache.Deletes(cacheClean, "setting_")

	return serializer.Response{}
}

const (
	SummaryRangeDays = 12
	MetricCacheKey   = "admin_summary"
	metricErrMsg     = "Failed to generate metrics summary"
)

type (
	SummaryService struct {
		Generate bool `form:"generate"`
	}
	SummaryParamCtx struct{}
)

// Summary 获取站点统计概况
func (s *SummaryService) Summary(c *gin.Context) (*HomepageSummary, error) {
	dep := dependency.FromContext(c)
	kv := dep.KV()
	res := &HomepageSummary{
		Version: &Version{
			Version: constants.BackendVersion,
			Pro:     constants.IsProBool,
			Commit:  constants.LastCommit,
		},
		SiteURls: lo.Map(dep.SettingProvider().AllSiteURLs(c), func(item *url.URL, index int) string {
			return item.String()
		}),
	}

	if summary, ok := kv.Get(MetricCacheKey); ok {
		summaryCasted := summary.(MetricsSummary)
		res.MetricsSummary = &summaryCasted
		return res, nil
	}

	if !s.Generate {
		return res, nil
	}

	summary := &MetricsSummary{
		Files:       make([]int, SummaryRangeDays),
		Users:       make([]int, SummaryRangeDays),
		Shares:      make([]int, SummaryRangeDays),
		Dates:       make([]time.Time, SummaryRangeDays),
		GeneratedAt: time.Now(),
	}

	fileClient := dep.FileClient()
	userClient := dep.UserClient()
	shareClient := dep.ShareClient()

	toRound := time.Now()
	timeBase := time.Date(toRound.Year(), toRound.Month(), toRound.Day()+1, 0, 0, 0, 0, toRound.Location())
	for day := range summary.Files {
		start := timeBase.Add(-time.Duration(SummaryRangeDays-day) * time.Hour * 24)
		end := timeBase.Add(-time.Duration(SummaryRangeDays-day-1) * time.Hour * 24)
		summary.Dates[day] = start
		fileTotal, err := fileClient.CountByTimeRange(c, &start, &end)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, metricErrMsg, nil)
		}
		userTotal, err := userClient.CountByTimeRange(c, &start, &end)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, metricErrMsg, nil)
		}
		shareTotal, err := shareClient.CountByTimeRange(c, &start, &end)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, metricErrMsg, nil)
		}
		summary.Files[day] = fileTotal
		summary.Users[day] = userTotal
		summary.Shares[day] = shareTotal
	}

	var err error
	summary.FileTotal, err = fileClient.CountByTimeRange(c, nil, nil)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, metricErrMsg, nil)
	}
	summary.UserTotal, err = userClient.CountByTimeRange(c, nil, nil)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, metricErrMsg, nil)
	}
	summary.ShareTotal, err = shareClient.CountByTimeRange(c, nil, nil)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, metricErrMsg, nil)
	}
	summary.EntitiesTotal, err = fileClient.CountEntityByTimeRange(c, nil, nil)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, metricErrMsg, nil)
	}

	_ = kv.Set(MetricCacheKey, *summary, 86400)
	res.MetricsSummary = summary

	return res, nil
}

// ThumbGeneratorTestService 缩略图生成测试服务
type (
	ThumbGeneratorTestService struct {
		Name       string `json:"name" binding:"required"`
		Executable string `json:"executable" binding:"required"`
	}
	ThumbGeneratorTestParamCtx struct{}
)

// Test 通过获取生成器版本来测试
func (s *ThumbGeneratorTestService) Test(c *gin.Context) (string, error) {
	version, err := thumb.TestGenerator(c, s.Name, s.Executable)
	if err != nil {
		return "", serializer.NewError(serializer.CodeParamErr, "Failed to invoke generator: "+err.Error(), err)
	}

	return version, nil
}

type (
	GetSettingService struct {
		Keys []string `json:"keys" binding:"required"`
	}
	GetSettingParamCtx struct{}
)

func (s *GetSettingService) GetSetting(c *gin.Context) (map[string]string, error) {
	dep := dependency.FromContext(c)
	res, err := dep.SettingClient().Gets(c, lo.Filter(s.Keys, func(item string, index int) bool {
		_, ok := inventory.RedactedSettings[strings.ToLower(item)]
		return !ok
	}))
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get settings", err)
	}

	return res, nil
}

type (
	SetSettingService struct {
		Settings map[string]string `json:"settings" binding:"required"`
	}
	SetSettingParamCtx   struct{}
	SettingPreProcessor  func(ctx context.Context, settings map[string]string) error
	SettingPostProcessor func(ctx context.Context, settings map[string]string) error
)

var (
	preprocessors = map[string]SettingPreProcessor{
		"siteURL":      siteUrlPreProcessor,
		"mime_mapping": mimeMappingPreProcessor,
		"secret_key":   secretKeyPreProcessor,
	}
	postprocessors = map[string]SettingPostProcessor{
		"mime_mapping":                               mimeMappingPostProcessor,
		"media_meta_exif":                            mediaMetaPostProcessor,
		"media_meta_music":                           mediaMetaPostProcessor,
		"media_meta_ffprobe":                         mediaMetaPostProcessor,
		"smtpUser":                                   emailPostProcessor,
		"smtpPass":                                   emailPostProcessor,
		"smtpHost":                                   emailPostProcessor,
		"smtpPort":                                   emailPostProcessor,
		"smtpEncryption":                             emailPostProcessor,
		"smtpFrom":                                   emailPostProcessor,
		"replyTo":                                    emailPostProcessor,
		"fromName":                                   emailPostProcessor,
		"fromAdress":                                 emailPostProcessor,
		"queue_media_meta_worker_num":                mediaMetaQueuePostProcessor,
		"queue_media_meta_max_execution":             mediaMetaQueuePostProcessor,
		"queue_media_meta_backoff_factor":            mediaMetaQueuePostProcessor,
		"queue_media_meta_backoff_max_duration":      mediaMetaQueuePostProcessor,
		"queue_media_meta_max_retry":                 mediaMetaQueuePostProcessor,
		"queue_media_meta_retry_delay":               mediaMetaQueuePostProcessor,
		"queue_thumb_worker_num":                     thumbQueuePostProcessor,
		"queue_thumb_max_execution":                  thumbQueuePostProcessor,
		"queue_thumb_backoff_factor":                 thumbQueuePostProcessor,
		"queue_thumb_backoff_max_duration":           thumbQueuePostProcessor,
		"queue_thumb_max_retry":                      thumbQueuePostProcessor,
		"queue_thumb_retry_delay":                    thumbQueuePostProcessor,
		"queue_recycle_worker_num":                   entityRecycleQueuePostProcessor,
		"queue_recycle_max_execution":                entityRecycleQueuePostProcessor,
		"queue_recycle_backoff_factor":               entityRecycleQueuePostProcessor,
		"queue_recycle_backoff_max_duration":         entityRecycleQueuePostProcessor,
		"queue_recycle_max_retry":                    entityRecycleQueuePostProcessor,
		"queue_recycle_retry_delay":                  entityRecycleQueuePostProcessor,
		"queue_io_intense_worker_num":                ioIntenseQueuePostProcessor,
		"queue_io_intense_max_execution":             ioIntenseQueuePostProcessor,
		"queue_io_intense_backoff_factor":            ioIntenseQueuePostProcessor,
		"queue_io_intense_backoff_max_duration":      ioIntenseQueuePostProcessor,
		"queue_io_intense_max_retry":                 ioIntenseQueuePostProcessor,
		"queue_io_intense_retry_delay":               ioIntenseQueuePostProcessor,
		"queue_remote_download_worker_num":           remoteDownloadQueuePostProcessor,
		"queue_remote_download_max_execution":        remoteDownloadQueuePostProcessor,
		"queue_remote_download_backoff_factor":       remoteDownloadQueuePostProcessor,
		"queue_remote_download_backoff_max_duration": remoteDownloadQueuePostProcessor,
		"queue_remote_download_max_retry":            remoteDownloadQueuePostProcessor,
		"queue_remote_download_retry_delay":          remoteDownloadQueuePostProcessor,
		"secret_key":                                 secretKeyPostProcessor,
	}
)

func (s *SetSettingService) SetSetting(c *gin.Context) (map[string]string, error) {
	dep := dependency.FromContext(c)
	kv := dep.KV()
	settingClient := dep.SettingClient()

	// Preprocess settings
	allPreprocessors := make(map[string]SettingPreProcessor)
	allPostprocessors := make(map[string]SettingPostProcessor)
	for k, _ := range s.Settings {
		if preprocessor, ok := preprocessors[k]; ok {
			fnName := reflect.TypeOf(preprocessor).Name()
			if _, ok := allPreprocessors[fnName]; !ok {
				allPreprocessors[fnName] = preprocessor
			}
		}

		if postprocessor, ok := postprocessors[k]; ok {
			fnName := reflect.TypeOf(postprocessor).Name()
			if _, ok := allPostprocessors[fnName]; !ok {
				allPostprocessors[fnName] = postprocessor
			}
		}
	}

	// Execute all preprocessors
	for _, preprocessor := range allPreprocessors {
		if err := preprocessor(c, s.Settings); err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Failed to validate settings", err)
		}
	}

	// Save to db
	sc, tx, ctx, err := inventory.WithTx(c, settingClient)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create transaction", err)
	}

	if err := sc.Set(ctx, s.Settings); err != nil {
		_ = inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to save settings", err)
	}

	if err := inventory.Commit(tx); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to commit transaction", err)
	}

	// Clean cache
	if err := kv.Delete(setting.KvSettingPrefix, lo.Keys(s.Settings)...); err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to clear cache", err)
	}

	// Execute post preprocessors
	for _, postprocessor := range allPostprocessors {
		if err := postprocessor(ctx, s.Settings); err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Failed to post process settings", err)
		}
	}

	return s.Settings, nil
}

func siteUrlPreProcessor(ctx context.Context, settings map[string]string) error {
	siteURL := settings["siteURL"]
	urls := strings.Split(siteURL, ",")
	for index, u := range urls {
		urlParsed, err := url.Parse(u)
		if err != nil {
			return fmt.Errorf("Failed to parse siteURL %q: %w", u, err)
		}

		urls[index] = urlParsed.String()
	}
	settings["siteURL"] = strings.Join(urls, ",")
	return nil
}

func secretKeyPreProcessor(ctx context.Context, settings map[string]string) error {
	settings["secret_key"] = util.RandStringRunes(256)
	return nil
}

func mimeMappingPreProcessor(ctx context.Context, settings map[string]string) error {
	var mapping map[string]string
	if err := json.Unmarshal([]byte(settings["mime_mapping"]), &mapping); err != nil {
		return serializer.NewError(serializer.CodeParamErr, "Invalid mime mapping", err)
	}

	return nil
}

func mimeMappingPostProcessor(ctx context.Context, settings map[string]string) error {
	dep := dependency.FromContext(ctx)
	dep.MimeDetector(context.WithValue(ctx, dependency.ReloadCtx{}, true))

	return nil
}

func mediaMetaPostProcessor(ctx context.Context, settings map[string]string) error {
	dep := dependency.FromContext(ctx)
	dep.MediaMetaExtractor(context.WithValue(ctx, dependency.ReloadCtx{}, true))
	return nil
}

func emailPostProcessor(ctx context.Context, settings map[string]string) error {
	dep := dependency.FromContext(ctx)
	dep.EmailClient(context.WithValue(ctx, dependency.ReloadCtx{}, true))
	return nil
}

func mediaMetaQueuePostProcessor(ctx context.Context, settings map[string]string) error {
	dep := dependency.FromContext(ctx)
	dep.MediaMetaQueue(context.WithValue(ctx, dependency.ReloadCtx{}, true)).Start()
	return nil
}

func ioIntenseQueuePostProcessor(ctx context.Context, settings map[string]string) error {
	dep := dependency.FromContext(ctx)
	dep.IoIntenseQueue(context.WithValue(ctx, dependency.ReloadCtx{}, true)).Start()
	return nil
}

func remoteDownloadQueuePostProcessor(ctx context.Context, settings map[string]string) error {
	dep := dependency.FromContext(ctx)
	dep.RemoteDownloadQueue(context.WithValue(ctx, dependency.ReloadCtx{}, true)).Start()
	return nil
}

func entityRecycleQueuePostProcessor(ctx context.Context, settings map[string]string) error {
	dep := dependency.FromContext(ctx)
	dep.EntityRecycleQueue(context.WithValue(ctx, dependency.ReloadCtx{}, true)).Start()
	return nil
}

func thumbQueuePostProcessor(ctx context.Context, settings map[string]string) error {
	dep := dependency.FromContext(ctx)
	dep.ThumbQueue(context.WithValue(ctx, dependency.ReloadCtx{}, true)).Start()
	return nil
}

func secretKeyPostProcessor(ctx context.Context, settings map[string]string) error {
	dep := dependency.FromContext(ctx)
	dep.KV().Delete(manager.EntityUrlCacheKeyPrefix)
	settings["secret_key"] = ""
	return nil
}
