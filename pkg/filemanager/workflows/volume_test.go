package workflows

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVolumeInfo(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		order int64
		ok    bool
	}{
		{"movie.part1.rar", "rar:movie", 1, true},
		{"movie.part02.rar", "rar:movie", 2, true},
		{"MOVIE.PART10.RAR", "rar:movie", 10, true},
		{"movie.rar", "rarold:movie", 0, true},
		{"movie.r00", "rarold:movie", 1, true},
		{"movie.r15", "rarold:movie", 16, true},
		{"movie.zip", "", 0, false},
		{"movie.z01", "", 0, false},
		{"movie.001", "", 0, false},
		{"movie.7z", "", 0, false},
		{"movie.tar.gz", "", 0, false},
		{"readme.txt", "", 0, false},
	}
	for _, c := range cases {
		key, order, ok := volumeInfo(c.name)
		require.Equal(t, c.ok, ok, c.name)
		if c.ok {
			require.Equal(t, c.key, key, c.name)
			require.Equal(t, c.order, order, c.name)
		}
	}
}

func TestFirstVolumeName(t *testing.T) {
	require.Equal(t, "movie.part01.rar", firstVolumeName("movie.part07.rar"))
	require.Equal(t, "movie.part1.rar", firstVolumeName("movie.part3.rar"))
	require.Equal(t, "movie.rar", firstVolumeName("movie.r02"))
	require.Equal(t, "", firstVolumeName("movie.part1.rar"))
	require.Equal(t, "", firstVolumeName("movie.rar"))
	require.Equal(t, "", firstVolumeName("movie.zip"))
	require.Equal(t, "", firstVolumeName("movie.001"))
	require.Equal(t, "", firstVolumeName("plain.txt"))
}

func TestFirstVolumeURI(t *testing.T) {
	base := "cloudreve://my/archives"
	src, ok := FirstVolumeURI([]string{
		base + "/movie.part2.rar",
		base + "/movie.part1.rar",
		base + "/movie.part3.rar",
	})
	require.True(t, ok)
	require.Equal(t, base+"/movie.part1.rar", src)

	// Old-style set: .rar is first.
	src, ok = FirstVolumeURI([]string{
		base + "/movie.r00",
		base + "/movie.rar",
	})
	require.True(t, ok)
	require.Equal(t, base+"/movie.rar", src)

	// Single source never matches.
	_, ok = FirstVolumeURI([]string{base + "/movie.part1.rar"})
	require.False(t, ok)

	// Mixed sets do not match.
	_, ok = FirstVolumeURI([]string{base + "/a.part1.rar", base + "/b.part2.rar"})
	require.False(t, ok)

	// Different folders do not match.
	_, ok = FirstVolumeURI([]string{base + "/movie.part1.rar", "cloudreve://my/other/movie.part2.rar"})
	require.False(t, ok)

	// Non-volume names do not match.
	_, ok = FirstVolumeURI([]string{base + "/a.txt", base + "/b.txt"})
	require.False(t, ok)
}
