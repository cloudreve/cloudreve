package cos

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/mediameta"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/samber/lo"
	"net/http"
	"strings"
	"time"
)

const (
	mediaInfoTTL = time.Duration(10) * time.Minute
	videoInfo    = "videoinfo"
)

var (
	supportedImageExt = []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff", "heic", "heif"}
)

type (
	ImageProp struct {
		Value string `json:"val"`
	}
	ImageInfo map[string]ImageProp
	Error     struct {
		XMLName   xml.Name `xml:"Error"`
		Code      string   `xml:"Code"`
		Message   string   `xml:"Message"`
		RequestId string   `xml:"RequestId"`
	}
	Video struct {
		Index          int    `xml:"Index"`
		CodecName      string `xml:"CodecName"`
		CodecLongName  string `xml:"CodecLongName"`
		CodecTimeBase  string `xml:"CodecTimeBase"`
		CodecTagString string `xml:"CodecTagString"`
		CodecTag       string `xml:"CodecTag"`
		ColorPrimaries string `xml:"ColorPrimaries"`
		ColorRange     string `xml:"ColorRange"`
		ColorTransfer  string `xml:"ColorTransfer"`
		Profile        string `xml:"Profile"`
		Width          int    `xml:"Width"`
		Height         int    `xml:"Height"`
		HasBFrame      string `xml:"HasBFrame"`
		RefFrames      string `xml:"RefFrames"`
		Sar            string `xml:"Sar"`
		Dar            string `xml:"Dar"`
		PixFormat      string `xml:"PixFormat"`
		FieldOrder     string `xml:"FieldOrder"`
		Level          string `xml:"Level"`
		Fps            string `xml:"Fps"`
		AvgFps         string `xml:"AvgFps"`
		Timebase       string `xml:"Timebase"`
		StartTime      string `xml:"StartTime"`
		Duration       string `xml:"Duration"`
		Bitrate        string `xml:"Bitrate"`
		NumFrames      string `xml:"NumFrames"`
		Language       string `xml:"Language"`
	}
	Audio struct {
		Index          int    `xml:"Index"`
		CodecName      string `xml:"CodecName"`
		CodecLongName  string `xml:"CodecLongName"`
		CodecTimeBase  string `xml:"CodecTimeBase"`
		CodecTagString string `xml:"CodecTagString"`
		CodecTag       string `xml:"CodecTag"`
		SampleFmt      string `xml:"SampleFmt"`
		SampleRate     string `xml:"SampleRate"`
		Channel        string `xml:"Channel"`
		ChannelLayout  string `xml:"ChannelLayout"`
		Timebase       string `xml:"Timebase"`
		StartTime      string `xml:"StartTime"`
		Duration       string `xml:"Duration"`
		Bitrate        string `xml:"Bitrate"`
		Language       string `xml:"Language"`
	}
	Subtitle struct {
		Index    string `xml:"Index"`
		Language string `xml:"Language"`
	}
	Response struct {
		XMLName   xml.Name `xml:"Response"`
		MediaInfo struct {
			Stream struct {
				Video    []Video    `xml:"Video"`
				Audio    []Audio    `xml:"Audio"`
				Subtitle []Subtitle `xml:"Subtitle"`
			} `xml:"Stream"`
			Format struct {
				NumStream      string `xml:"NumStream"`
				NumProgram     string `xml:"NumProgram"`
				FormatName     string `xml:"FormatName"`
				FormatLongName string `xml:"FormatLongName"`
				StartTime      string `xml:"StartTime"`
				Duration       string `xml:"Duration"`
				Bitrate        string `xml:"Bitrate"`
				Size           string `xml:"Size"`
			} `xml:"Format"`
		} `xml:"MediaInfo"`
	}
)

func (handler *Driver) extractStreamMeta(ctx context.Context, path string) ([]driver.MediaMeta, error) {
	resp, err := handler.extractMediaInfo(ctx, path, &urlOption{CiProcess: videoInfo})
	if err != nil {
		return nil, err
	}

	var info Response
	if err := xml.Unmarshal([]byte(resp), &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal media info: %w", err)
	}

	streams := lo.Map(info.MediaInfo.Stream.Video, func(stream Video, index int) mediameta.Stream {
		return mediameta.Stream{
			Index:         stream.Index,
			CodecName:     stream.CodecName,
			CodecLongName: stream.CodecLongName,
			CodecType:     "video",
			Width:         stream.Width,
			Height:        stream.Height,
			Bitrate:       stream.Bitrate,
		}
	})
	streams = append(streams, lo.Map(info.MediaInfo.Stream.Audio, func(stream Audio, index int) mediameta.Stream {
		return mediameta.Stream{
			Index:         stream.Index,
			CodecName:     stream.CodecName,
			CodecLongName: stream.CodecLongName,
			CodecType:     "audio",
			Bitrate:       stream.Bitrate,
		}
	})...)

	metas := make([]driver.MediaMeta, 0)
	metas = append(metas, mediameta.ProbeMetaTransform(&mediameta.FFProbeMeta{
		Format: &mediameta.Format{
			FormatName:     info.MediaInfo.Format.FormatName,
			FormatLongName: info.MediaInfo.Format.FormatLongName,
			Duration:       info.MediaInfo.Format.Duration,
			Bitrate:        info.MediaInfo.Format.Bitrate,
		},
		Streams: streams,
	})...)

	return nil, nil
}

func (handler *Driver) extractImageMeta(ctx context.Context, path string) ([]driver.MediaMeta, error) {
	exif := ""
	resp, err := handler.extractMediaInfo(ctx, path, &urlOption{
		Exif: &exif,
	})
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

// extractMediaInfo Sends API calls to COS service to extract media info.
func (handler *Driver) extractMediaInfo(ctx context.Context, path string, opt *urlOption) (string, error) {
	mediaInfoExpire := time.Now().Add(mediaInfoTTL)
	thumbURL, err := handler.signSourceURL(
		ctx,
		path,
		&mediaInfoExpire,
		opt,
	)
	if err != nil {
		return "", fmt.Errorf("failed to sign media info url: %w", err)
	}

	resp, err := handler.httpClient.
		Request(http.MethodGet, thumbURL, nil, request.WithContext(ctx)).
		CheckHTTPResponse(http.StatusOK).
		GetResponseIgnoreErr()
	if err != nil {
		return "", handleCosError(resp, err)
	}

	return resp, nil
}

func parseGpsInfo(imageInfo ImageInfo) []driver.MediaMeta {
	return mediameta.GpsMeta(imageInfo["GPSLatitude"].Value, imageInfo["GPSLongitude"].Value,
		imageInfo["GPSLatitudeRef"].Value, imageInfo["GPSLongitudeRef"].Value, parseRawGPS)
}

func parseRawGPS(gpsStr string, ref string) float64 {
	elem := strings.Split(gpsStr, " ")

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

func handleCosError(resp string, originErr error) error {
	if resp == "" {
		return originErr
	}

	var err Error
	if err := xml.Unmarshal([]byte(resp), &err); err != nil {
		return fmt.Errorf("failed to unmarshal cos error: %w", err)
	}

	return fmt.Errorf("cos error: %s", err.Message)
}
