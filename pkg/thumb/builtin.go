package thumb

import (
	"context"
	"fmt"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gofrs/uuid"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	//"github.com/nfnt/resize"
	"golang.org/x/image/draw"
)

const thumbTempFolder = "thumb"

// BuiltinSupportedExts lists file extensions supported by the built-in
// thumbnail generator. Extensions are lowercased and do not include the dot.
var BuiltinSupportedExts = []string{"jpg", "jpeg", "png", "gif"}

// Thumb 缩略图
type Thumb struct {
	src image.Image
	ext string
}

// NewThumbFromFile 从文件数据获取新的Thumb对象，
// 尝试通过文件名name解码图像
func NewThumbFromFile(file io.Reader, ext string) (*Thumb, error) {
	// 无扩展名时
	if ext == "" {
		return nil, fmt.Errorf("unknown image format: %w", ErrPassThrough)
	}

	var err error
	var img image.Image
	switch ext {
	case "jpg", "jpeg":
		img, err = jpeg.Decode(file)
	case "gif":
		img, err = gif.Decode(file)
	case "png":
		img, err = png.Decode(file)
	default:
		return nil, fmt.Errorf("unknown image format %q: %w", ext, ErrPassThrough)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse image: %w (%w)", err, ErrPassThrough)
	}

	return &Thumb{
		src: img,
		ext: ext[1:],
	}, nil
}

// GetThumb 生成给定最大尺寸的缩略图
func (image *Thumb) GetThumb(width, height uint) {
	//image.src = resize.Thumbnail(width, height, image.src, resize.Lanczos3)
	image.src = Thumbnail(width, height, image.src)
}

// GetSize 获取图像尺寸
func (image *Thumb) GetSize() (int, int) {
	b := image.src.Bounds()
	return b.Max.X, b.Max.Y
}

// Save 保存图像到给定路径
func (image *Thumb) Save(w io.Writer, encodeSetting *setting.ThumbEncode) (err error) {
	switch encodeSetting.Format {
	case "png":
		err = png.Encode(w, image.src)
	default:
		err = jpeg.Encode(w, image.src, &jpeg.Options{Quality: encodeSetting.Quality})
	}

	return err

}

// Thumbnail will downscale provided image to max width and height preserving
// original aspect ratio and using the interpolation function interp.
// It will return original image, without processing it, if original sizes
// are already smaller than provided constraints.
func Thumbnail(maxWidth, maxHeight uint, img image.Image) image.Image {
	origBounds := img.Bounds()
	origWidth := uint(origBounds.Dx())
	origHeight := uint(origBounds.Dy())
	newWidth, newHeight := origWidth, origHeight

	// Return original image if it have same or smaller size as constraints
	if maxWidth >= origWidth && maxHeight >= origHeight {
		return img
	}

	// Preserve aspect ratio
	if origWidth > maxWidth {
		newHeight = uint(origHeight * maxWidth / origWidth)
		if newHeight < 1 {
			newHeight = 1
		}
		newWidth = maxWidth
	}

	if newHeight > maxHeight {
		newWidth = uint(newWidth * maxHeight / newHeight)
		if newWidth < 1 {
			newWidth = 1
		}
		newHeight = maxHeight
	}
	return Resize(newWidth, newHeight, img)
}

func Resize(newWidth, newHeight uint, img image.Image) image.Image {
	// Set the expected size that you want:
	dst := image.NewRGBA(image.Rect(0, 0, int(newWidth), int(newHeight)))
	// Resize:
	draw.BiLinear.Scale(dst, dst.Rect, img, img.Bounds(), draw.Src, nil)
	return dst
}

// CreateAvatar 创建头像
func (image *Thumb) CreateAvatar(width int) {
	image.src = Resize(uint(width), uint(width), image.src)
}

type Builtin struct {
	settings setting.Provider
}

func NewBuiltinGenerator(settings setting.Provider) *Builtin {
	return &Builtin{
		settings: settings,
	}
}

func (b Builtin) Generate(ctx context.Context, es entitysource.EntitySource, ext string, previous *Result) (*Result, error) {
	if es.Entity().Size() > b.settings.BuiltinThumbMaxSize(ctx) {
		return nil, fmt.Errorf("file is too big: %w", ErrPassThrough)
	}

	img, err := NewThumbFromFile(es, ext)
	if err != nil {
		return nil, err
	}

	w, h := b.settings.ThumbSize(ctx)
	img.GetThumb(uint(w), uint(h))
	tempPath := filepath.Join(
		util.DataPath(b.settings.TempPath(ctx)),
		thumbTempFolder,
		fmt.Sprintf("thumb_%s", uuid.Must(uuid.NewV4()).String()),
	)

	thumbFile, err := util.CreatNestedFile(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	defer thumbFile.Close()
	if err := img.Save(thumbFile, b.settings.ThumbEncode(ctx)); err != nil {
		return &Result{Path: tempPath}, err
	}

	return &Result{Path: tempPath}, nil
}

func (b Builtin) Priority() int {
	return 300
}

func (b Builtin) Enabled(ctx context.Context) bool {
	return b.settings.BuiltinThumbGeneratorEnabled(ctx)
}
