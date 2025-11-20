package oss

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/mediameta"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/samber/lo"
)

const (
	imageInfoProcess = "image/info"
	videoInfoProcess = "video/info"
	audioInfoProcess = "audio/info"
	mediaInfoTTL     = time.Duration(10) * time.Minute
)

var (
	supportedImageExt = []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff", "heic", "heif"}
	supportedAudioExt = []string{"mp3", "wav", "flac", "aac", "m4a", "ogg", "wma", "ape", "alac", "amr", "opus"}
	supportedVideoExt = []string{"mp4", "mkv", "avi", "mov", "flv", "wmv", "rmvb", "webm", "3gp", "mpg", "mpeg", "m4v", "ts", "m3u8", "vob", "f4v", "rm", "asf", "divx", "ogv", "dat", "mts", "m2ts", "swf", "avi", "3g2", "m2v", "m4p", "m4b", "m4r", "m4v", "m4a"}
)

type (
	ImageProp struct {
		Value string `json:"value"`
	}
	ImageInfo map[string]ImageProp

	Error struct {
		XMLName      xml.Name `xml:"Error"`
		Text         string   `xml:",chardata"`
		Code         string   `xml:"Code"`
		Message      string   `xml:"Message"`
		RequestId    string   `xml:"RequestId"`
		HostId       string   `xml:"HostId"`
		EC           string   `xml:"EC"`
		RecommendDoc string   `xml:"RecommendDoc"`
	}

	StreamMediaInfo struct {
		RequestID      string        `json:"RequestId"`
		Language       string        `json:"Language"`
		Title          string        `json:"Title"`
		VideoStreams   []VideoStream `json:"VideoStreams"`
		AudioStreams   []AudioStream `json:"AudioStreams"`
		Subtitles      []Subtitle    `json:"Subtitles"`
		StreamCount    int64         `json:"StreamCount"`
		ProgramCount   int64         `json:"ProgramCount"`
		FormatName     string        `json:"FormatName"`
		FormatLongName string        `json:"FormatLongName"`
		Size           int64         `json:"Size"`
		StartTime      float64       `json:"StartTime"`
		Bitrate        int64         `json:"Bitrate"`
		Artist         string        `json:"Artist"`
		AlbumArtist    string        `json:"AlbumArtist"`
		Composer       string        `json:"Composer"`
		Performer      string        `json:"Performer"`
		Album          string        `json:"Album"`
		Duration       float64       `json:"Duration"`
		ProduceTime    string        `json:"ProduceTime"`
		LatLong        string        `json:"LatLong"`
		VideoWidth     int64         `json:"VideoWidth"`
		VideoHeight    int64         `json:"VideoHeight"`
		Addresses      []Address     `json:"Addresses"`
	}

	Address struct {
		Language    string `json:"Language"`
		AddressLine string `json:"AddressLine"`
		Country     string `json:"Country"`
		Province    string `json:"Province"`
		City        string `json:"City"`
		District    string `json:"District"`
		Township    string `json:"Township"`
	}

	AudioStream struct {
		Index          int     `json:"Index"`
		Language       string  `json:"Language"`
		CodecName      string  `json:"CodecName"`
		CodecLongName  string  `json:"CodecLongName"`
		CodecTimeBase  string  `json:"CodecTimeBase"`
		CodecTagString string  `json:"CodecTagString"`
		CodecTag       string  `json:"CodecTag"`
		TimeBase       string  `json:"TimeBase"`
		StartTime      float64 `json:"StartTime"`
		Duration       float64 `json:"Duration"`
		Bitrate        int64   `json:"Bitrate"`
		FrameCount     int64   `json:"FrameCount"`
		Lyric          string  `json:"Lyric"`
		SampleFormat   string  `json:"SampleFormat"`
		SampleRate     int64   `json:"SampleRate"`
		Channels       int64   `json:"Channels"`
		ChannelLayout  string  `json:"ChannelLayout"`
	}

	Subtitle struct {
		Index          int64   `json:"Index"`
		Language       string  `json:"Language"`
		CodecName      string  `json:"CodecName"`
		CodecLongName  string  `json:"CodecLongName"`
		CodecTagString string  `json:"CodecTagString"`
		CodecTag       string  `json:"CodecTag"`
		StartTime      float64 `json:"StartTime"`
		Duration       float64 `json:"Duration"`
		Bitrate        int64   `json:"Bitrate"`
		Content        string  `json:"Content"`
		Width          int64   `json:"Width"`
		Height         int64   `json:"Height"`
	}

	VideoStream struct {
		Index              int     `json:"Index"`
		Language           string  `json:"Language"`
		CodecName          string  `json:"CodecName"`
		CodecLongName      string  `json:"CodecLongName"`
		Profile            string  `json:"Profile"`
		CodecTimeBase      string  `json:"CodecTimeBase"`
		CodecTagString     string  `json:"CodecTagString"`
		CodecTag           string  `json:"CodecTag"`
		Width              int     `json:"Width"`
		Height             int     `json:"Height"`
		HasBFrames         int     `json:"HasBFrames"`
		SampleAspectRatio  string  `json:"SampleAspectRatio"`
		DisplayAspectRatio string  `json:"DisplayAspectRatio"`
		PixelFormat        string  `json:"PixelFormat"`
		Level              int     `json:"Level"`
		FrameRate          string  `json:"FrameRate"`
		AverageFrameRate   string  `json:"AverageFrameRate"`
		TimeBase           string  `json:"TimeBase"`
		StartTime          float64 `json:"StartTime"`
		Duration           float64 `json:"Duration"`
		Bitrate            int64   `json:"Bitrate"`
		FrameCount         int64   `json:"FrameCount"`
		Rotate             string  `json:"Rotate"`
		BitDepth           int     `json:"BitDepth"`
		ColorSpace         string  `json:"ColorSpace"`
		ColorRange         string  `json:"ColorRange"`
		ColorTransfer      string  `json:"ColorTransfer"`
		ColorPrimaries     string  `json:"ColorPrimaries"`
	}
)

func (handler *Driver) extractIMMMeta(ctx context.Context, path, category string) ([]driver.MediaMeta, error) {
	resp, err := handler.extractMediaInfo(ctx, path, category, true)
	if err != nil {
		return nil, err
	}

	var info StreamMediaInfo
	if err := json.Unmarshal([]byte(resp), &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal media info: %w", err)
	}

	streams := lo.Map(info.VideoStreams, func(stream VideoStream, index int) mediameta.Stream {
		bitrate := ""
		if stream.Bitrate != 0 {
			bitrate = strconv.FormatInt(stream.Bitrate, 10)
		}
		return mediameta.Stream{
			Index:         stream.Index,
			CodecName:     stream.CodecName,
			CodecLongName: stream.CodecLongName,
			CodecType:     "video",
			Width:         stream.Width,
			Height:        stream.Height,
			Duration:      strconv.FormatFloat(stream.Duration, 'f', -1, 64),
			Bitrate:       bitrate,
		}
	})
	streams = append(streams, lo.Map(info.AudioStreams, func(stream AudioStream, index int) mediameta.Stream {
		bitrate := ""
		if stream.Bitrate != 0 {
			bitrate = strconv.FormatInt(stream.Bitrate, 10)
		}
		return mediameta.Stream{
			Index:         stream.Index,
			CodecName:     stream.CodecName,
			CodecLongName: stream.CodecLongName,
			CodecType:     "audio",
			Duration:      strconv.FormatFloat(stream.Duration, 'f', -1, 64),
			Bitrate:       bitrate,
		}
	})...)

	metas := make([]driver.MediaMeta, 0)
	metas = append(metas, mediameta.ProbeMetaTransform(&mediameta.FFProbeMeta{
		Format: &mediameta.Format{
			FormatName:     info.FormatName,
			FormatLongName: info.FormatLongName,
			Duration:       strconv.FormatFloat(info.Duration, 'f', -1, 64),
			Bitrate:        strconv.FormatInt(info.Bitrate, 10),
		},
		Streams: streams,
	})...)

	if info.Artist != "" {
		metas = append(metas, driver.MediaMeta{
			Key:   mediameta.MusicArtist,
			Value: info.Artist,
			Type:  driver.MediaTypeMusic,
		})
	}

	if info.AlbumArtist != "" {
		metas = append(metas, driver.MediaMeta{
			Key:   mediameta.MusicAlbumArtists,
			Value: info.AlbumArtist,
			Type:  driver.MediaTypeMusic,
		})
	}

	if info.Composer != "" {
		metas = append(metas, driver.MediaMeta{
			Key:   mediameta.MusicComposer,
			Value: info.Composer,
			Type:  driver.MediaTypeMusic,
		})
	}

	if info.Album != "" {
		metas = append(metas, driver.MediaMeta{
			Key:   mediameta.MusicAlbum,
			Value: info.Album,
			Type:  driver.MediaTypeMusic,
		})
	}

	return metas, nil
}

func (handler *Driver) extractImageMeta(ctx context.Context, path string) ([]driver.MediaMeta, error) {
	resp, err := handler.extractMediaInfo(ctx, path, imageInfoProcess, false)
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

// extractMediaInfo Sends API calls to OSS IMM service to extract media info.
func (handler *Driver) extractMediaInfo(ctx context.Context, path string, category string, forceSign bool) (string, error) {
	mediaInfoExpire := time.Now().Add(mediaInfoTTL)
	thumbURL, err := handler.signSourceURL(
		ctx,
		path,
		&mediaInfoExpire,
		&oss.GetObjectRequest{
			Process: oss.Ptr(category),
		},
		forceSign,
	)
	if err != nil {
		return "", fmt.Errorf("failed to sign media info url: %w", err)
	}

	resp, err := handler.httpClient.
		Request(http.MethodGet, thumbURL, nil, request.WithContext(ctx)).
		CheckHTTPResponse(http.StatusOK).
		GetResponseIgnoreErr()
	if err != nil {
		return "", handleOssError(resp, err)
	}

	return resp, nil
}

func parseGpsInfo(imageInfo ImageInfo) []driver.MediaMeta {
	latitude := imageInfo["GPSLatitude"]   // 31deg 16.26808'
	longitude := imageInfo["GPSLongitude"] // 120deg 42.91039'
	latRef := imageInfo["GPSLatitudeRef"]  // North
	lonRef := imageInfo["GPSLongitudeRef"] // East

	// Make sure all value exist in map
	if latitude.Value == "" || longitude.Value == "" || latRef.Value == "" || lonRef.Value == "" {
		return nil
	}

	lat := parseRawGPS(latitude.Value, latRef.Value)
	lon := parseRawGPS(longitude.Value, lonRef.Value)
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
	elem := strings.Split(gpsStr, " ")
	if len(elem) < 1 {
		return 0
	}

	var (
		deg     float64
		minutes float64
		seconds float64
	)

	deg, _ = strconv.ParseFloat(strings.TrimSuffix(elem[0], "deg"), 64)
	if len(elem) >= 2 {
		minutes, _ = strconv.ParseFloat(strings.TrimSuffix(elem[1], "'"), 64)
	}
	if len(elem) >= 3 {
		seconds, _ = strconv.ParseFloat(strings.TrimSuffix(elem[2], "\""), 64)
	}

	decimal := deg + minutes/60.0 + seconds/3600.0

	if ref == "South" || ref == "West" {
		return -decimal
	}

	return decimal
}

func handleOssError(resp string, originErr error) error {
	if resp == "" {
		return originErr
	}

	var err Error
	if err := xml.Unmarshal([]byte(resp), &err); err != nil {
		return fmt.Errorf("failed to unmarshal oss error: %w", err)
	}

	return fmt.Errorf("oss error: %s", err.Message)
}
