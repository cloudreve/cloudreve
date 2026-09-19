package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRandStringRunes(t *testing.T) {
	asserts := assert.New(t)

	// 0 长度字符
	randStr := RandStringRunes(0)
	asserts.Len(randStr, 0)

	// 16 长度字符
	randStr = RandStringRunes(16)
	asserts.Len(randStr, 16)

	// 32 长度字符
	randStr = RandStringRunes(32)
	asserts.Len(randStr, 32)

	//相同长度字符
	sameLenStr1 := RandStringRunes(32)
	sameLenStr2 := RandStringRunes(32)
	asserts.NotEqual(sameLenStr1, sameLenStr2)
}

func TestContainsUint(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(ContainsUint([]uint{0, 2, 3, 65, 4}, 65))
	asserts.True(ContainsUint([]uint{65}, 65))
	asserts.False(ContainsUint([]uint{65}, 6))
}

func TestContainsString(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(ContainsString([]string{"", "1"}, ""))
	asserts.True(ContainsString([]string{"", "1"}, "1"))
	asserts.False(ContainsString([]string{"", "1"}, " "))
}

func TestReplace(t *testing.T) {
	asserts := assert.New(t)

	asserts.Equal("origin", Replace(map[string]string{
		"123": "321",
	}, "origin"))

	asserts.Equal("321origin321", Replace(map[string]string{
		"123": "321",
	}, "123origin123"))
	asserts.Equal("321new321", Replace(map[string]string{
		"123":    "321",
		"origin": "new",
	}, "123origin123"))
}

func TestSliceDifference(t *testing.T) {
	asserts := assert.New(t)

	{
		s1 := []string{"1", "2", "3", "4"}
		s2 := []string{"2", "4"}
		asserts.Equal([]string{"1", "3"}, SliceDifference(s1, s2))
	}

	{
		s2 := []string{"1", "2", "3", "4"}
		s1 := []string{"2", "4"}
		asserts.Equal([]string{}, SliceDifference(s1, s2))
	}

	{
		s1 := []string{"1", "2", "3", "4"}
		s2 := []string{"1", "2", "3", "4"}
		asserts.Equal([]string{}, SliceDifference(s1, s2))
	}

	{
		s1 := []string{"1", "2", "3", "4"}
		s2 := []string{}
		asserts.Equal([]string{"1", "2", "3", "4"}, SliceDifference(s1, s2))
	}
}
