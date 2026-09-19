package qiniu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/mediameta"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/samber/lo"
)

const (
	exifParam    = "exif"
	avInfoParam  = "avinfo"
	mediaInfoTTL = time.Duration(10) * time.Minute
)

var (
	supportedImageExt = []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff"}
)

type (
	ImageProp struct {
		Value string `json:"val"`
	}
	ImageInfo       map[string]ImageProp
	QiniuMediaError struct {
		Error string `json:"error"`
		Code  int    `json:"code"`
	}
)

func (handler *Driver) extractAvMeta(ctx context.Context, path string) ([]driver.MediaMeta, error) {
	resp, err := handler.extractMediaInfo(ctx, path, avInfoParam)
	if err != nil {
		return nil, err
	}

	var avInfo *mediameta.FFProbeMeta
	if err := json.Unmarshal([]byte(resp), &avInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal media info: %w", err)
	}

	metas := mediameta.ProbeMetaTransform(avInfo)
	if artist, ok := avInfo.Format.Tags["artist"]; ok {
		metas = append(metas, driver.MediaMeta{
			Key:   mediameta.Artist,
			Value: artist,
			Type:  driver.MediaTypeMusic,
		})
	}

	if album, ok := avInfo.Format.Tags["album"]; ok {
		metas = append(metas, driver.MediaMeta{
			Key:   mediameta.MusicAlbum,
			Value: album,
			Type:  driver.MediaTypeMusic,
		})
	}

	if title, ok := avInfo.Format.Tags["title"]; ok {
		metas = append(metas, driver.MediaMeta{
			Key:   mediameta.MusicTitle,
			Value: title,
			Type:  driver.MediaTypeMusic,
		})
	}

	return metas, nil
}

func (handler *Driver) extractImageMeta(ctx context.Context, path string) ([]driver.MediaMeta, error) {
	resp, err := handler.extractMediaInfo(ctx, path, exifParam)
	if err != nil {
		return nil, err
	}

	var imageInfo ImageInfo
	if err := json.Unmarshal([]byte(resp), &imageInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal media info: %w", err)
	}

	metas := make([]driver.MediaMeta, 0)
	exifMap := lo.MapEntries(imageInfo, func(key string, value ImageProp) (string, string) {
		return key, value.Value
	})
	metas = append(metas, mediameta.ExtractExifMap(exifMap, time.Time{})...)
	metas = append(metas, parseGpsInfo(imageInfo)...)
	for i := 0; i < len(metas); i++ {
		metas[i].Type = driver.MetaTypeExif
	}

	return metas, nil
}

func (handler *Driver) extractMediaInfo(ctx context.Context, path string, param string) (string, error) {
	mediaInfoExpire := time.Now().Add(mediaInfoTTL)
	ediaInfoUrl := handler.signSourceURL(path, url.Values{
		param: []string{},
	}, &mediaInfoExpire)
	resp, err := handler.httpClient.
		Request(http.MethodGet, ediaInfoUrl, nil, request.WithContext(ctx)).
		CheckHTTPResponse(http.StatusOK).
		GetResponseIgnoreErr()
	if err != nil {
		return "", unmarshalError(resp, err)
	}

	return resp, nil
}

func unmarshalError(resp string, originErr error) error {
	if resp == "" {
		return originErr
	}

	var err QiniuMediaError
	if err := json.Unmarshal([]byte(resp), &err); err != nil {
		return fmt.Errorf("failed to unmarshal qiniu error: %w", err)
	}

	return fmt.Errorf("qiniu error: %s", err.Error)
}

func parseGpsInfo(imageInfo ImageInfo) []driver.MediaMeta {
	return mediameta.GpsMeta(imageInfo["GPSLatitude"].Value, imageInfo["GPSLongitude"].Value,
		imageInfo["GPSLatitudeRef"].Value, imageInfo["GPSLongitudeRef"].Value, parseRawGPS)
}

func parseRawGPS(gpsStr string, ref string) float64 {
	elem := strings.Split(gpsStr, ", ")

	var deg, minutes, seconds float64
	if len(elem) >= 1 {
		deg, _ = strconv.ParseFloat(elem[0], 64)
	}
	if len(elem) >= 2 {
		minutes, _ = strconv.ParseFloat(elem[1], 64)
	}
	if len(elem) >= 3 {
		seconds, _ = strconv.ParseFloat(elem[2], 64)
	}

	return mediameta.DMSDecimal(deg, minutes, seconds, ref == "S" || ref == "W")
}
