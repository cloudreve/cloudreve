package mediameta

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

var (
	ffprobeExts = []string{
		"mp3", "m4a", "ogg", "flac", "3g2", "3gp", "asf", "asx", "avi", "divx", "flv", "m2ts", "m2v", "m4v", "mkv", "mov", "mp4",
		"mpeg", "mpg", "mts", "mxf", "ogv", "rm", "swf", "webm", "wmv",
	}
)

type (
	FFProbeMeta struct {
		Format   *Format   `json:"format"`
		Streams  []Stream  `json:"streams"`
		Chapters []Chapter `json:"chapters"`
	}

	Stream struct {
		Index         int    `json:"index"`
		CodecName     string `json:"codec_name"`
		CodecLongName string `json:"codec_long_name"`
		CodecType     string `json:"codec_type"`
		Width         int    `json:"width"`
		Height        int    `json:"height"`
		Duration      string `json:"duration"`
		Bitrate       string `json:"bit_rate"`
	}
	Chapter struct {
		Id        int               `json:"id"`
		StartTime string            `json:"start_time"`
		EndTime   string            `json:"end_time"`
		Tags      map[string]string `json:"tags"`
	}
	Format struct {
		FormatName     string            `json:"format_name"`
		FormatLongName string            `json:"format_long_name"`
		Duration       string            `json:"duration"`
		Bitrate        string            `json:"bit_rate"`
		Tags           map[string]string `json:"tags"`
	}
)

const (
	UrlExpire                = time.Duration(60) * time.Hour
	StreamMediaFormat        = "format"
	StreamMediaFormatLong    = "formatLong"
	StreamMediaDuration      = "duration"
	StreamMediaBitrate       = "bitrate"
	StreamMediaStreamPrefix  = "stream_"
	StreamMediaChapterPrefix = "chapter_"
	StreamMediaCodec         = "codec"
	StreamMediaCodecLongName = "codec_long_name"
	StreamMediaWidth         = "width"
	StreamMediaHeight        = "height"
	StreamMediaStartTime     = "start_time"
	StreamMediaEndTime       = "end_time"
	StreamMediaChapterName   = "name"
	StreamMetaTitle          = "title"
	StreamMetaDescription    = "description"
)

func newFFProbeExtractor(settings setting.Provider, l logging.Logger) *ffprobeExtractor {
	return &ffprobeExtractor{
		l:        l,
		settings: settings,
	}
}

type ffprobeExtractor struct {
	settings setting.Provider
	l        logging.Logger
}

func (f *ffprobeExtractor) Exts() []string {
	return ffprobeExts
}

func (f *ffprobeExtractor) Extract(ctx context.Context, ext string, source entitysource.EntitySource, opts ...optionFunc) ([]driver.MediaMeta, error) {
	option := &option{}
	for _, opt := range opts {
		opt.apply(option)
	}

	localLimit, remoteLimit := f.settings.MediaMetaFFProbeSizeLimit(ctx)
	if err := checkFileSize(localLimit, remoteLimit, source); err != nil {
		return nil, err
	}

	var input string
	if source.IsLocal() && !source.Entity().Encrypted() {
		input = source.LocalPath(ctx)
	} else {
		expire := time.Now().Add(UrlExpire)
		srcUrl, err := source.Url(driver.WithForcePublicEndpoint(ctx, false), entitysource.WithNoInternalProxy(), entitysource.WithExpire(&expire))
		if err != nil {
			return nil, fmt.Errorf("failed to get entity url: %w", err)
		}
		input = srcUrl.Url
	}

	cmd := exec.CommandContext(ctx,
		f.settings.MediaMetaFFProbePath(ctx),
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-show_chapters",
		input,
	)

	res, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to invoke ffprobe: %w", err)
	}

	f.l.Debug("ffprobe output: %s", res)
	var meta FFProbeMeta
	if err := json.Unmarshal(res, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	return ProbeMetaTransform(&meta), nil
}

func ProbeMetaTransform(meta *FFProbeMeta) []driver.MediaMeta {
	if meta.Format == nil {
		return nil
	}

	res := []driver.MediaMeta{}
	if meta.Format.FormatName != "" {
		res = append(res, driver.MediaMeta{
			Key:   StreamMediaFormat,
			Value: meta.Format.FormatName,
		})
	}
	if meta.Format.FormatLongName != "" {
		res = append(res, driver.MediaMeta{
			Key:   StreamMediaFormatLong,
			Value: meta.Format.FormatLongName,
		})
	}
	if meta.Format.Duration != "" {
		res = append(res, driver.MediaMeta{
			Key:   StreamMediaDuration,
			Value: meta.Format.Duration,
		})
	}
	if meta.Format.Bitrate != "" {
		res = append(res, driver.MediaMeta{
			Key:   StreamMediaBitrate,
			Value: meta.Format.Bitrate,
		})
	}

	for _, stream := range meta.Streams {
		keyPrefix := fmt.Sprintf("%s%d_%s_", StreamMediaStreamPrefix, stream.Index, stream.CodecType)
		if stream.CodecName != "" {
			res = append(res, driver.MediaMeta{
				Key:   keyPrefix + StreamMediaCodec,
				Value: stream.CodecName,
			})
		}
		if stream.CodecLongName != "" {
			res = append(res, driver.MediaMeta{
				Key:   keyPrefix + StreamMediaCodecLongName,
				Value: stream.CodecLongName,
			})
		}
		if stream.Width > 0 {
			res = append(res, driver.MediaMeta{
				Key:   keyPrefix + StreamMediaWidth,
				Value: strconv.Itoa(stream.Width),
			})
		}
		if stream.Height > 0 {
			res = append(res, driver.MediaMeta{
				Key:   keyPrefix + StreamMediaHeight,
				Value: strconv.Itoa(stream.Height),
			})
		}
		if stream.Duration != "" {
			res = append(res, driver.MediaMeta{
				Key:   keyPrefix + StreamMediaDuration,
				Value: stream.Duration,
			})
		}
		if stream.Bitrate != "" {
			res = append(res, driver.MediaMeta{
				Key:   keyPrefix + StreamMediaBitrate,
				Value: stream.Bitrate,
			})
		}
	}

	for _, chapter := range meta.Chapters {
		keyPrefix := fmt.Sprintf("%s%d_", StreamMediaChapterPrefix, chapter.Id)
		if chapter.StartTime != "" {
			res = append(res, driver.MediaMeta{
				Key:   keyPrefix + StreamMediaStartTime,
				Value: chapter.StartTime,
			})
		}
		if chapter.EndTime != "" {
			res = append(res, driver.MediaMeta{
				Key:   keyPrefix + StreamMediaEndTime,
				Value: chapter.EndTime,
			})
		}
		if title, ok := chapter.Tags["title"]; ok {
			res = append(res, driver.MediaMeta{
				Key:   keyPrefix + StreamMediaChapterName,
				Value: title,
			})
		}
	}

	if title, ok := meta.Format.Tags["title"]; ok {
		res = append(res, driver.MediaMeta{
			Key:   StreamMetaTitle,
			Value: title,
		})
	}

	if description, ok := meta.Format.Tags["description"]; ok {
		res = append(res, driver.MediaMeta{
			Key:   StreamMetaDescription,
			Value: description[0:min(len(description), 255)],
		})
	}

	for i := 0; i < len(res); i++ {
		res[i].Type = driver.MetaTypeStreamMedia
	}

	return res
}
