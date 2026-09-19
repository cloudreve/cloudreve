package upyun

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/mediameta"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/samber/lo"
	"net/http"
	"strings"
	"time"
)

var (
	mediaInfoTTL = time.Duration(10) * time.Minute
)

type (
	ImageInfo struct {
		Exif map[string]string `json:"EXIF"`
	}
)

func (handler *Driver) extractImageMeta(ctx context.Context, path string) ([]driver.MediaMeta, error) {
	resp, err := handler.extractMediaInfo(ctx, path, "!/meta")
	if err != nil {
		return nil, err
	}

	var imageInfo ImageInfo
	if err := json.Unmarshal([]byte(resp), &imageInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal image info: %w", err)
	}

	metas := make([]driver.MediaMeta, 0, len(imageInfo.Exif))
	exifMap := lo.MapEntries(imageInfo.Exif, func(key string, value string) (string, string) {
		switch key {
		case "0xA434":
			key = "LensModel"
		}
		return key, value
	})
	metas = append(metas, mediameta.ExtractExifMap(exifMap, time.Time{})...)
	metas = append(metas, parseGpsInfo(imageInfo.Exif)...)

	for i := 0; i < len(metas); i++ {
		metas[i].Type = driver.MetaTypeExif
	}

	return metas, nil
}

func (handler *Driver) extractMediaInfo(ctx context.Context, path string, param string) (string, error) {
	mediaInfoExpire := time.Now().Add(mediaInfoTTL)
	mediaInfoUrl, err := handler.signURL(ctx, path+param, nil, &mediaInfoExpire)
	if err != nil {
		return "", err
	}

	resp, err := handler.httpClient.
		Request(http.MethodGet, mediaInfoUrl, nil, request.WithContext(ctx)).
		CheckHTTPResponse(http.StatusOK).
		GetResponseIgnoreErr()
	if err != nil {
		return "", unmarshalError(resp, err)
	}

	return resp, nil
}

func unmarshalError(resp string, err error) error {
	return fmt.Errorf("upyun error: %s", err)
}

func parseGpsInfo(imageInfo map[string]string) []driver.MediaMeta {
	return mediameta.GpsMeta(imageInfo["GPSLatitude"], imageInfo["GPSLongitude"],
		imageInfo["GPSLatitudeRef"], imageInfo["GPSLongitudeRef"], parseRawGPS)
}

func parseRawGPS(gpsStr string, ref string) float64 {
	elem := strings.Split(gpsStr, ",")

	var deg, minutes, seconds float64
	if len(elem) >= 1 {
		deg = mediameta.GpsRationalElem(elem[0])
	}
	if len(elem) >= 2 {
		minutes = mediameta.GpsRationalElem(elem[1])
	}
	if len(elem) >= 3 {
		seconds = mediameta.GpsRationalElem(elem[2])
	}

	return mediameta.DMSDecimal(deg, minutes, seconds, ref == "S" || ref == "W")
}
