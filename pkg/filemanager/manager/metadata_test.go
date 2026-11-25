package manager

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/stretchr/testify/require"
)

type emojiMetadataTestSettings struct {
	setting.Provider
	presets string
}

func (s emojiMetadataTestSettings) EmojiPresets(context.Context) string {
	return s.presets
}

func resetEmojiMetadataCache(t *testing.T) {
	t.Helper()
	emojiPresetsMu.Lock()
	oldHash, oldPresets := lastEmojiHash, emojiPresets
	lastEmojiHash, emojiPresets = "", nil
	emojiPresetsMu.Unlock()
	t.Cleanup(func() {
		emojiPresetsMu.Lock()
		lastEmojiHash, emojiPresets = oldHash, oldPresets
		emojiPresetsMu.Unlock()
	})
}

func TestEmojiMetadataValidation(t *testing.T) {
	resetEmojiMetadataCache(t)
	const first = `{"faces":["\ud83d\ude00"],"travel":["\ud83d\ude80"]}`
	const second = `{"animals":["\ud83d\udc31"]}`
	tests := []struct {
		name    string
		presets string
		value   string
		wantErr string
	}{
		{name: "initial preset", presets: first, value: "\U0001f600"},
		{name: "cached other category", presets: first, value: "\U0001f680"},
		{name: "unsupported emoji", presets: first, value: "\U0001f431", wantErr: "unsupported emoji"},
		{name: "new settings", presets: second, value: "\U0001f431"},
		{name: "removed preset", presets: second, value: "\U0001f600", wantErr: "unsupported emoji"},
		{name: "invalid JSON", presets: "{", value: "\U0001f431", wantErr: "failed to read emoji setting"},
		{name: "invalid JSON not cached", presets: "{", value: "\U0001f431", wantErr: "failed to read emoji setting"},
		{name: "last valid settings", presets: second, value: "\U0001f431"},
		{name: "empty presets", presets: "{}", value: "\U0001f431", wantErr: "unsupported emoji"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := &manager{settings: emojiMetadataTestSettings{presets: test.presets}}
			patches, err := m.validateMetadata(context.Background(), fs.MetadataPatch{
				Key: emojiIconMetadataKey, Value: test.value,
			})
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				require.Empty(t, patches)
				return
			}
			require.NoError(t, err)
			require.Len(t, patches, 1)
			require.True(t, patches[0].UpdateModifiedAt)
		})
	}
}

func TestEmojiMetadataRemoveDoesNotLoadPresets(t *testing.T) {
	m := &manager{}
	patches, err := m.validateMetadata(context.Background(), fs.MetadataPatch{
		Key: emojiIconMetadataKey, Remove: true,
	})
	require.NoError(t, err)
	require.Len(t, patches, 1)
	require.True(t, patches[0].Remove)
	require.True(t, patches[0].UpdateModifiedAt)
}

func TestEmojiMetadataConcurrentValidation(t *testing.T) {
	type presetCase struct {
		presets string
		valid   string
		invalid string
	}
	defaultPresets := presetCase{
		presets: inventory.DefaultSettings["emojis"],
		valid:   "\U0001f600",
		invalid: "__not_a_configured_emoji__",
	}
	differentPresets := []presetCase{
		{presets: `{"faces":["\ud83d\ude00"]}`, valid: "\U0001f600", invalid: "\U0001f680"},
		{presets: `{"travel":["\ud83d\ude80"]}`, valid: "\U0001f680", invalid: "\U0001f600"},
	}
	for _, test := range []struct {
		name  string
		warm  bool
		mixed bool
	}{
		{name: "cold cache"},
		{name: "warm cache", warm: true},
		{name: "different settings", mixed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			resetEmojiMetadataCache(t)
			if test.warm {
				m := &manager{settings: emojiMetadataTestSettings{presets: defaultPresets.presets}}
				_, err := m.validateMetadata(context.Background(), fs.MetadataPatch{
					Key: emojiIconMetadataKey, Value: defaultPresets.valid,
				})
				require.NoError(t, err)
			}

			const workers = 16
			start := make(chan struct{})
			errs := make(chan error, workers)
			var ready, done sync.WaitGroup
			ready.Add(workers)
			done.Add(workers)
			for i := range workers {
				presets := defaultPresets
				if test.mixed {
					presets = differentPresets[i%len(differentPresets)]
				}
				go func() {
					defer done.Done()
					m := &manager{settings: emojiMetadataTestSettings{presets: presets.presets}}
					ready.Done()
					<-start
					for range 32 {
						if _, err := m.validateMetadata(context.Background(), fs.MetadataPatch{
							Key: emojiIconMetadataKey, Value: presets.valid,
						}); err != nil {
							errs <- fmt.Errorf("valid emoji rejected: %w", err)
							return
						}
						if _, err := m.validateMetadata(context.Background(), fs.MetadataPatch{
							Key: emojiIconMetadataKey, Value: presets.invalid,
						}); err == nil {
							errs <- fmt.Errorf("emoji from another preset configuration accepted")
							return
						}
					}
				}()
			}
			ready.Wait()
			close(start)
			done.Wait()
			close(errs)
			for err := range errs {
				require.NoError(t, err)
			}
		})
	}
}
