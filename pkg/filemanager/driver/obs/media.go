package obs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/mediameta"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/samber/lo"
)

func (d *Driver) MediaMeta(ctx context.Context, path, ext, language string) ([]driver.MediaMeta, error) {
	thumbURL, err := d.signSourceURL(&obs.CreateSignedUrlInput{
		Method:  obs.HttpMethodGet,
		Bucket:  d.policy.BucketName,
		Key:     path,
		Expires: int(mediaInfoTTL.Seconds()),
		QueryParams: map[string]string{
			imageProcessHeader: imageInfoProcessor,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to sign media info url: %w", err)
	}

	resp, err := d.httpClient.
		Request(http.MethodGet, thumbURL, nil, request.WithContext(ctx)).
		CheckHTTPResponse(http.StatusOK).
		GetResponseIgnoreErr()
	if err != nil {
		return nil, handleJsonError(resp, err)
	}

	var imageInfo map[string]any
	if err := json.Unmarshal([]byte(resp), &imageInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal media info: %w", err)
	}

	imageInfoMap := lo.MapEntries(imageInfo, func(k string, v any) (string, string) {
		if vStr, ok := v.(string); ok {
			return strings.TrimPrefix(k, "exif:"), vStr
		}

		return k, fmt.Sprintf("%v", v)
	})
	metas := make([]driver.MediaMeta, 0)
	metas = append(metas, mediameta.ExtractExifMap(imageInfoMap, time.Time{})...)
	metas = append(metas, parseGpsInfo(imageInfoMap)...)
	for i := 0; i < len(metas); i++ {
		metas[i].Type = driver.MetaTypeExif
	}
	return metas, nil
}

func parseGpsInfo(imageInfo map[string]string) []driver.MediaMeta {
	return mediameta.GpsMeta(imageInfo["GPSLatitude"], imageInfo["GPSLongitude"],
		imageInfo["GPSLatitudeRef"], imageInfo["GPSLongitudeRef"], parseRawGPS)
}

func parseRawGPS(gpsStr string, ref string) float64 {
	elem := strings.Split(gpsStr, ", ")

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
