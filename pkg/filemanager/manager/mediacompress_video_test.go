package manager

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNormalizeVideoContainer covers APP-103: "keep" preserves the source
// extension for recognised video containers, an explicit container overrides it,
// and non-video / unknown inputs yield "" (→ skipped).
func TestNormalizeVideoContainer(t *testing.T) {
	cases := []struct {
		container string
		sourceExt string
		want      string
	}{
		{"keep", "mp4", "mp4"},
		{"keep", "MOV", "mov"},  // case-insensitive
		{"", "mkv", "mkv"},      // empty behaves like keep
		{"keep", "txt", ""},     // non-video source → skip
		{"keep", "", ""},        // no extension → skip
		{"mp4", "avi", "mp4"},   // explicit container overrides source
		{"webm", "mp4", "webm"}, // explicit container overrides source
		{"exe", "mp4", ""},      // unknown explicit container → skip
	}
	for _, c := range cases {
		assert.Equal(t, c.want, normalizeVideoContainer(c.container, c.sourceExt),
			"normalizeVideoContainer(%q, %q)", c.container, c.sourceExt)
	}
}

// TestVideoScaleFilter covers APP-103: a valid WxH yields a scale filter that caps
// without upscaling (min(...)) and forces even dimensions; empty/invalid disables
// scaling (empty string → no -vf).
func TestVideoScaleFilter(t *testing.T) {
	assert.Equal(t, "", videoScaleFilter(""), "empty spec disables scaling")
	assert.Equal(t, "", videoScaleFilter("1080"), "missing dimension is invalid")
	assert.Equal(t, "", videoScaleFilter("axb"), "non-numeric is invalid")
	assert.Equal(t, "", videoScaleFilter("0x0"), "non-positive is invalid")

	f := videoScaleFilter("1920x1080")
	assert.Contains(t, f, "min(1920,iw)")
	assert.Contains(t, f, "min(1080,ih)")
	assert.Contains(t, f, "force_original_aspect_ratio=decrease")
	assert.Contains(t, f, "force_divisible_by=2")
	assert.True(t, strings.HasPrefix(f, "scale="))
}
