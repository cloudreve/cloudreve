package workflows

import (
	"errors"
	"io"
	iofs "io/fs"
	"strings"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/stretchr/testify/require"
)

func testProgress(count, size int64) queue.Progresses {
	return queue.Progresses{
		ProgressTypeExtractCount: &queue.Progress{Current: count},
		ProgressTypeExtractSize:  &queue.Progress{Current: size},
	}
}

func TestCheckExtractGuards(t *testing.T) {
	// Under all limits.
	require.NoError(t, checkExtractGuards(testProgress(10, 100), 1000))

	// At the cumulative size limit — aborts as non-retryable.
	err := checkExtractGuards(testProgress(10, 1000), 1000)
	require.Error(t, err)
	require.True(t, errors.Is(err, queue.CriticalErr))

	// At the entry cap — aborts as non-retryable.
	err = checkExtractGuards(testProgress(maxExtractEntries, 10), 1000)
	require.Error(t, err)
	require.True(t, errors.Is(err, queue.CriticalErr))

	// Zero size limit disables the size bound; entry cap still applies.
	require.NoError(t, checkExtractGuards(testProgress(10, 1<<62), 0))
	require.Error(t, checkExtractGuards(testProgress(maxExtractEntries, 0), 0))
}

type stubFile struct {
	io.Reader
}

func (stubFile) Stat() (iofs.FileInfo, error) { return nil, nil }
func (stubFile) Close() error                 { return nil }

func TestCappedFileExactBoundary(t *testing.T) {
	// Entry ends exactly at the budget — stream must terminate with EOF.
	capped := &cappedFile{File: stubFile{strings.NewReader("12345")}, remaining: 5}
	buf := make([]byte, 8)

	n, err := capped.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 5, n)
	require.Equal(t, "12345", string(buf[:n]))

	_, err = capped.Read(buf)
	require.Equal(t, io.EOF, err)
}

func TestCappedFileBomb(t *testing.T) {
	// Entry data beyond the budget — read fails with the critical limit error.
	capped := &cappedFile{File: stubFile{strings.NewReader("123456")}, remaining: 5}
	buf := make([]byte, 8)

	n, err := capped.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 5, n)

	_, err = capped.Read(buf)
	require.Error(t, err)
	require.True(t, errors.Is(err, queue.CriticalErr))
}

func TestCappedFileShortReads(t *testing.T) {
	// Budget caps each read; consecutive reads drain only the remaining bytes.
	capped := &cappedFile{File: stubFile{strings.NewReader("abcdef")}, remaining: 3}
	buf := make([]byte, 2)

	var got []byte
	var err error
	for {
		var n int
		n, err = capped.Read(buf)
		got = append(got, buf[:n]...)
		if err != nil {
			break
		}
	}
	require.Equal(t, "abc", string(got))
	require.True(t, errors.Is(err, queue.CriticalErr))
}
