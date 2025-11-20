package obs

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
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
	latitude := imageInfo["GPSLatitude"]   // 31/1, 162680820/10000000, 0/1
	longitude := imageInfo["GPSLongitude"] // 120/1, 429103939/10000000, 0/1
	latRef := imageInfo["GPSLatitudeRef"]  // N
	lonRef := imageInfo["GPSLongitudeRef"] // E

	// Make sure all value exist in map
	if latitude == "" || longitude == "" || latRef == "" || lonRef == "" {
		return nil
	}

	lat := parseRawGPS(latitude, latRef)
	lon := parseRawGPS(longitude, lonRef)
	if !math.IsNaN(lat) && !math.IsNaN(lon) {
		lat, lng := mediameta.NormalizeGPS(lat, lon)
		return []driver.MediaMeta{{
			Key:   mediameta.GpsLat,
			Value: fmt.Sprintf("%f", lat),
		}, {
			Key:   mediameta.GpsLng,
			Value: fmt.Sprintf("%f", lng),
		}}
	}

	return nil
}

func parseRawGPS(gpsStr string, ref string) float64 {
	elem := strings.Split(gpsStr, ", ")
	if len(elem) < 1 {
		return 0
	}

	var (
		deg     float64
		minutes float64
		seconds float64
	)

	deg = getGpsElemValue(elem[0])
	if len(elem) >= 2 {
		minutes = getGpsElemValue(elem[1])
	}
	if len(elem) >= 3 {
		seconds = getGpsElemValue(elem[2])
	}

	decimal := deg + minutes/60.0 + seconds/3600.0

	if ref == "S" || ref == "W" {
		return -decimal
	}

	return decimal
}

func getGpsElemValue(elm string) float64 {
	elements := strings.Split(elm, "/")
	if len(elements) != 2 {
		return 0
	}

	numerator, err := strconv.ParseFloat(elements[0], 64)
	if err != nil {
		return 0
	}

	denominator, err := strconv.ParseFloat(elements[1], 64)
	if err != nil || denominator == 0 {
		return 0
	}

	return numerator / denominator
}
