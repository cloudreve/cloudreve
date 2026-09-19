package cache

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewMemoStore(t *testing.T) {
	asserts := assert.New(t)

	store := NewMemoStore("", logging.NewConsoleLogger(logging.LevelDebug))
	asserts.NotNil(store)
	asserts.NotNil(store.Store)
}

func TestMemoStore_Set(t *testing.T) {
	asserts := assert.New(t)

	store := NewMemoStore("", logging.NewConsoleLogger(logging.LevelDebug))
	err := store.Set("KEY", "vAL", -1)
	asserts.NoError(err)

	val, ok := store.Store.Load("KEY")
	asserts.True(ok)
	asserts.Equal("vAL", val.(itemWithTTL).Value)
}

func TestMemoStore_Get(t *testing.T) {
	asserts := assert.New(t)
	store := NewMemoStore("", logging.NewConsoleLogger(logging.LevelDebug))

	// 正常情况
	{
		_ = store.Set("string", "string_val", -1)
		val, ok := store.Get("string")
		asserts.Equal("string_val", val)
		asserts.True(ok)
	}

	// Key不存在
	{
		val, ok := store.Get("something")
		asserts.Equal(nil, val)
		asserts.False(ok)
	}

	// 存储struct
	{
		type testStruct struct {
			key int
		}
		test := testStruct{key: 233}
		_ = store.Set("struct", test, -1)
		val, ok := store.Get("struct")
		asserts.True(ok)
		res, ok := val.(testStruct)
		asserts.True(ok)
		asserts.Equal(test, res)
	}

	// 过期
	{
		_ = store.Set("string", "string_val", 1)
		store.Store.Store("string", itemWithTTL{Value: "string_val", Expires: time.Now().Unix() - 1})
		val, ok := store.Get("string")
		asserts.Nil(val)
		asserts.False(ok)
	}

}

func TestMemoStore_Gets(t *testing.T) {
	asserts := assert.New(t)
	store := NewMemoStore("", logging.NewConsoleLogger(logging.LevelDebug))

	err := store.Set("1", "1,val", -1)
	err = store.Set("2", "2,val", -1)
	err = store.Set("3", "3,val", -1)
	err = store.Set("4", "4,val", -1)
	asserts.NoError(err)

	// 全部命中
	{
		values, miss := store.Gets([]string{"1", "2", "3", "4"}, "")
		asserts.Len(values, 4)
		asserts.Len(miss, 0)
	}

	// 命中一半
	{
		values, miss := store.Gets([]string{"1", "2", "9", "10"}, "")
		asserts.Len(values, 2)
		asserts.Equal([]string{"9", "10"}, miss)
	}
}

func TestMemoStore_Sets(t *testing.T) {
	asserts := assert.New(t)
	store := NewMemoStore("", logging.NewConsoleLogger(logging.LevelDebug))

	err := store.Sets(map[string]interface{}{
		"1": "1.val",
		"2": "2.val",
		"3": "3.val",
		"4": "4.val",
	}, "test_")
	asserts.NoError(err)

	vals, miss := store.Gets([]string{"1", "2", "3", "4"}, "test_")
	asserts.Len(miss, 0)
	asserts.Equal(map[string]interface{}{
		"1": "1.val",
		"2": "2.val",
		"3": "3.val",
		"4": "4.val",
	}, vals)
}

func TestMemoStore_Delete(t *testing.T) {
	asserts := assert.New(t)
	store := NewMemoStore("", logging.NewConsoleLogger(logging.LevelDebug))

	err := store.Sets(map[string]interface{}{
		"1": "1.val",
		"2": "2.val",
		"3": "3.val",
		"4": "4.val",
	}, "test_")
	asserts.NoError(err)

	err = store.Delete("test_", "1", "2")
	asserts.NoError(err)
	values, miss := store.Gets([]string{"1", "2", "3", "4"}, "test_")
	asserts.Equal([]string{"1", "2"}, miss)
	asserts.Equal(map[string]interface{}{"3": "3.val", "4": "4.val"}, values)
}

func TestMemoStore_GarbageCollect(t *testing.T) {
	asserts := assert.New(t)
	store := NewMemoStore("", logging.NewConsoleLogger(logging.LevelDebug))
	store.Set("test", 1, 1)
	store.Store.Store("test", itemWithTTL{Value: 1, Expires: time.Now().Unix() - 1})
	store.GarbageCollect(logging.NewConsoleLogger(logging.LevelDebug))
	_, ok := store.Get("test")
	asserts.False(ok)
}
