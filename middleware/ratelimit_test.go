package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/gin-gonic/gin"
)

var testEngine = func() *gin.Engine {
	e := gin.New()
	e.ContextWithFallback = true
	return e
}()

func newRateLimitContext(t *testing.T, dep dependency.Dep, remoteAddr string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c := gin.CreateTestContextOnly(w, testEngine)
	req := httptest.NewRequest(http.MethodPost, "/session/token", nil)
	req.RemoteAddr = remoteAddr + ":12345"
	c.Request = req.WithContext(context.WithValue(req.Context(), dependency.DepCtx{}, dep))
	return c, w
}

func TestRateLimitByIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dep := dependency.NewDependency(dependency.WithKV(cache.NewMemoStore("", nil)))

	handler := RateLimitByIP("login", 3, time.Minute)

	for i := 0; i < 3; i++ {
		c, _ := newRateLimitContext(t, dep, "192.0.2.1")
		handler(c)
		if c.IsAborted() {
			t.Fatalf("request %d unexpectedly rejected", i+1)
		}
	}

	c, w := newRateLimitContext(t, dep, "192.0.2.1")
	handler(c)
	if !c.IsAborted() {
		t.Fatal("request over limit was not rejected")
	}
	if w.Header().Get("Retry-After") != "60" {
		t.Fatalf("missing/incorrect Retry-After: %q", w.Header().Get("Retry-After"))
	}

	// A different IP has its own bucket.
	c, _ = newRateLimitContext(t, dep, "192.0.2.2")
	handler(c)
	if c.IsAborted() {
		t.Fatal("unrelated IP shared the bucket")
	}

	// A different bucket name on the same IP has its own counter.
	handler2 := RateLimitByIP("register", 1, time.Minute)
	c, _ = newRateLimitContext(t, dep, "192.0.2.1")
	handler2(c)
	if c.IsAborted() {
		t.Fatal("unrelated bucket shared the counter")
	}
}

func TestRateLimitExpiry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dep := dependency.NewDependency(dependency.WithKV(cache.NewMemoStore("", nil)))

	handler := RateLimitByIP("login", 1, time.Second)

	c, _ := newRateLimitContext(t, dep, "192.0.2.1")
	handler(c)
	if c.IsAborted() {
		t.Fatal("first request rejected")
	}

	c, _ = newRateLimitContext(t, dep, "192.0.2.1")
	handler(c)
	if !c.IsAborted() {
		t.Fatal("second request not rejected")
	}

	// Window expiry frees the bucket. MemoStore TTL is Unix-second granular
	// (item valid while Expires >= now), so a 1s window can live ~2s.
	time.Sleep(2100 * time.Millisecond)

	c, _ = newRateLimitContext(t, dep, "192.0.2.1")
	handler(c)
	if c.IsAborted() {
		t.Fatal("request after window expiry still rejected")
	}
}

func TestRateLimitCountOverflow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	kv := cache.NewMemoStore("", nil)
	dep := dependency.NewDependency(dependency.WithKV(kv))

	// A non-int value in the bucket is treated as a fresh window.
	if err := kv.Set(rateLimitPrefix+"login:192.0.2.1", "garbage", 60); err != nil {
		t.Fatal(err)
	}

	handler := RateLimitByIP("login", 1, time.Minute)
	c, _ := newRateLimitContext(t, dep, "192.0.2.1")
	handler(c)
	if c.IsAborted() {
		t.Fatal("corrupt bucket state rejected request")
	}

	raw, ok := kv.Get(rateLimitPrefix + "login:192.0.2.1")
	if !ok {
		t.Fatal("bucket not persisted")
	}
	if _, isInt := raw.(int); !isInt {
		t.Fatalf("bucket value type drifted: %T", raw)
	}
}
