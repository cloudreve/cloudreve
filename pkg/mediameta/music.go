package mediameta

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/dhowden/tag"
)

var (
	audioExts = []string{
		"mp3", "m4a", "ogg", "flac",
	}
)

const (
	MusicFormat       = "format"
	MusicFileType     = "file_type"
	MusicTitle        = "title"
	MusicAlbum        = "album"
	MusicArtist       = "artist"
	MusicAlbumArtists = "album_artists"
	MusicComposer     = "composer"
	MusicGenre        = "genre"
	MusicYear         = "year"
	MusicTrack        = "track"
	MusicDisc         = "disc"
)

func newMusicExtractor(settings setting.Provider, l logging.Logger) *musicExtractor {
	return &musicExtractor{
		l:        l,
		settings: settings,
	}
}

type musicExtractor struct {
	l        logging.Logger
	settings setting.Provider
}

func (a *musicExtractor) Exts() []string {
	return audioExts
}

func (a *musicExtractor) Extract(ctx context.Context, ext string, source entitysource.EntitySource, opts ...optionFunc) ([]driver.MediaMeta, error) {
	option := &option{}
	for _, opt := range opts {
		opt.apply(option)
	}

	localLimit, remoteLimit := a.settings.MediaMetaMusicSizeLimit(ctx)
	if err := checkFileSize(localLimit, remoteLimit, source); err != nil {
		return nil, err
	}

	m, err := tag.ReadFrom(source)
	if err != nil {
		if errors.Is(err, tag.ErrNoTagsFound) {
			a.l.Debug("No tags found in file.")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read tags from file: %w", err)
	}

	metas := []driver.MediaMeta{
		{
			Key:   MusicFormat,
			Value: string(m.Format()),
		},
		{
			Key:   MusicFileType,
			Value: string(m.FileType()),
		},
	}

	if title := m.Title(); title != "" {
		metas = append(metas, driver.MediaMeta{
			Key:   MusicTitle,
			Value: title,
		})
	}

	if album := m.Album(); album != "" {
		metas = append(metas, driver.MediaMeta{
			Key:   MusicAlbum,
			Value: album,
		})
	}

	if artist := m.Artist(); artist != "" {
		metas = append(metas, driver.MediaMeta{
			Key:   MusicArtist,
			Value: artist,
		})
	}

	if albumArtists := m.AlbumArtist(); albumArtists != "" {
		metas = append(metas, driver.MediaMeta{
			Key:   MusicAlbumArtists,
			Value: albumArtists,
		})
	}

	if composer := m.Composer(); composer != "" {
		metas = append(metas, driver.MediaMeta{
			Key:   MusicComposer,
			Value: composer,
		})
	}

	if genre := m.Genre(); genre != "" {
		metas = append(metas, driver.MediaMeta{
			Key:   MusicGenre,
			Value: genre,
		})
	}

	if year := m.Year(); year != 0 {
		metas = append(metas, driver.MediaMeta{
			Key:   MusicYear,
			Value: fmt.Sprintf("%d", year),
		})
	}

	if track, total := m.Track(); track != 0 {
		metas = append(metas, driver.MediaMeta{
			Key:   MusicTrack,
			Value: fmt.Sprintf("%d/%d", track, total),
		})
	}

	if disc, total := m.Disc(); disc != 0 {
		metas = append(metas, driver.MediaMeta{
			Key:   MusicDisc,
			Value: fmt.Sprintf("%d/%d", disc, total),
		})
	}

	for i := 0; i < len(metas); i++ {
		metas[i].Type = driver.MediaTypeMusic
	}

	return metas, nil
}
