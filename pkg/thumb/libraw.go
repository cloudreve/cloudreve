package thumb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gofrs/uuid"
)

func NewLibRawGenerator(l logging.Logger, settings setting.Provider) *LibRawGenerator {
	return &LibRawGenerator{l: l, settings: settings}
}

type LibRawGenerator struct {
	l        logging.Logger
	settings setting.Provider
}

func (l *LibRawGenerator) Generate(ctx context.Context, es entitysource.EntitySource, ext string, previous *Result) (*Result, error) {
	if !util.IsInExtensionListExt(l.settings.LibRawThumbExts(ctx), ext) {
		return nil, fmt.Errorf("unsupported video format: %w", ErrPassThrough)
	}

	if es.Entity().Size() > l.settings.LibRawThumbMaxSize(ctx) {
		return nil, fmt.Errorf("file is too big: %w", ErrPassThrough)
	}

	// If download/copy files to temp folder
	tempFolder := filepath.Join(
		util.DataPath(l.settings.TempPath(ctx)),
		"thumb",
		fmt.Sprintf("libraw_%s", uuid.Must(uuid.NewV4()).String()),
	)
	tempInputFileName := fmt.Sprintf("libraw_%s.%s", uuid.Must(uuid.NewV4()).String(), ext)
	tempPath := filepath.Join(tempFolder, tempInputFileName)
	tempInputFile, err := util.CreatNestedFile(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	defer os.Remove(tempPath)
	defer tempInputFile.Close()

	if _, err = io.Copy(tempInputFile, es); err != nil {
		return &Result{Path: tempPath}, fmt.Errorf("failed to write input file: %w", err)
	}

	tempInputFile.Close()

	cmd := exec.CommandContext(ctx,
		l.settings.LibRawThumbPath(ctx), "-e", tempPath)

	// Redirect IO
	var dcrawErr bytes.Buffer
	cmd.Stderr = &dcrawErr

	if err := cmd.Run(); err != nil {
		l.l.Warning("Failed to invoke dcraw: %s", dcrawErr.String())
		return &Result{Path: tempPath}, fmt.Errorf("failed to invoke dcraw: %w, raw output: %s", err, dcrawErr.String())
	}

	return &Result{
		Path: filepath.Join(
			tempFolder,
			tempInputFileName+".thumb.jpg",
		),
		Continue: true,
		Cleanup:  []func(){func() { _ = os.RemoveAll(tempFolder) }},
	}, nil
}

func (l *LibRawGenerator) Priority() int {
	return 50
}

func (l *LibRawGenerator) Enabled(ctx context.Context) bool {
	return l.settings.LibRawThumbGeneratorEnabled(ctx)
}

func rotateImg(filePath string, orientation int) error {
	resultImg, err := os.OpenFile(filePath, os.O_RDWR, 0777)
	if err != nil {
		return err
	}
	defer func() { _ = resultImg.Close() }()

	imgFlag := make([]byte, 3)
	if _, err = io.ReadFull(resultImg, imgFlag); err != nil {
		return err
	}
	if _, err = resultImg.Seek(0, 0); err != nil {
		return err
	}

	var img image.Image
	if bytes.Equal(imgFlag, []byte{0xFF, 0xD8, 0xFF}) {
		img, err = jpeg.Decode(resultImg)
	} else {
		img, err = png.Decode(resultImg)
	}
	if err != nil {
		return err
	}

	switch orientation {
	case 8:
		img = rotate90(img)
	case 3:
		img = rotate90(rotate90(img))
	case 6:
		img = rotate90(rotate90(rotate90(img)))
	case 2:
		img = mirrorImg(img)
	case 7:
		img = rotate90(mirrorImg(img))
	case 4:
		img = rotate90(rotate90(mirrorImg(img)))
	case 5:
		img = rotate90(rotate90(rotate90(mirrorImg(img))))
	}

	if err = resultImg.Truncate(0); err != nil {
		return err
	}
	if _, err = resultImg.Seek(0, 0); err != nil {
		return err
	}

	if bytes.Equal(imgFlag, []byte{0xFF, 0xD8, 0xFF}) {
		return jpeg.Encode(resultImg, img, nil)
	}
	return png.Encode(resultImg, img)
}

func getJpegOrientation(fileName string) (int, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	header := make([]byte, 6)
	defer func() { header = nil }()
	if _, err = io.ReadFull(f, header); err != nil {
		return 0, err
	}

	// jpeg format header
	if !bytes.Equal(header[:3], []byte{0xFF, 0xD8, 0xFF}) {
		return 0, errors.New("not a jpeg")
	}

	// not a APP1 marker
	if header[3] != 0xE1 {
		return 1, nil
	}

	// exif data total length
	totalLen := int(header[4])<<8 + int(header[5]) - 2
	buf := make([]byte, totalLen)
	defer func() { buf = nil }()
	if _, err = io.ReadFull(f, buf); err != nil {
		return 0, err
	}

	// remove Exif identifier code
	buf = buf[6:]

	// byte order
	parse16, parse32, err := initParseMethod(buf[:2])
	if err != nil {
		return 0, err
	}

	// version
	_ = buf[2:4]

	// first IFD offset
	offset := parse32(buf[4:8])

	// first DE offset
	offset += 2
	buf = buf[offset:]

	const (
		orientationTag = 0x112
		deEntryLength  = 12
	)
	for len(buf) > deEntryLength {
		tag := parse16(buf[:2])
		if tag == orientationTag {
			return int(parse32(buf[8:12])), nil
		}
		buf = buf[deEntryLength:]
	}

	return 0, errors.New("orientation not found")
}

func initParseMethod(buf []byte) (func([]byte) int16, func([]byte) int32, error) {
	if bytes.Equal(buf, []byte{0x49, 0x49}) {
		return littleEndian16, littleEndian32, nil
	}
	if bytes.Equal(buf, []byte{0x4D, 0x4D}) {
		return bigEndian16, bigEndian32, nil
	}
	return nil, nil, errors.New("invalid byte order")
}

func littleEndian16(buf []byte) int16 {
	return int16(buf[0]) | int16(buf[1])<<8
}

func bigEndian16(buf []byte) int16 {
	return int16(buf[1]) | int16(buf[0])<<8
}

func littleEndian32(buf []byte) int32 {
	return int32(buf[0]) | int32(buf[1])<<8 | int32(buf[2])<<16 | int32(buf[3])<<24
}

func bigEndian32(buf []byte) int32 {
	return int32(buf[3]) | int32(buf[2])<<8 | int32(buf[1])<<16 | int32(buf[0])<<24
}

func rotate90(img image.Image) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	newImg := image.NewRGBA(image.Rect(0, 0, height, width))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			newImg.Set(y, width-x-1, img.At(x, y))
		}
	}
	return newImg
}

func mirrorImg(img image.Image) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	newImg := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			newImg.Set(width-x-1, y, img.At(x, y))
		}
	}
	return newImg
}
