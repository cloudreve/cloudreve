package upyun

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/mediameta"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/samber/lo"
	"math"
	"net/http"
	"strconv"
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
	elem := strings.Split(gpsStr, ",")
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
