package auth

import (
	"testing"
	"time"
)

func TestHMACSignCheckRoundTrip(t *testing.T) {
	a := HMACAuth{SecretKey: []byte("secret")}

	sign := a.Sign("body-content", 0)
	if err := a.Check("body-content", sign); err != nil {
		t.Fatalf("expected valid sign, got %v", err)
	}
}

func TestHMACCheckExpired(t *testing.T) {
	a := HMACAuth{SecretKey: []byte("secret")}

	sign := a.Sign("body", time.Now().Add(-time.Hour).Unix())
	if err := a.Check("body", sign); err != ErrExpired {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestHMACCheckMissingExpires(t *testing.T) {
	a := HMACAuth{SecretKey: []byte("secret")}

	if err := a.Check("body", "abc:"); err != ErrExpiresMissing {
		t.Fatalf("expected ErrExpiresMissing, got %v", err)
	}
}

func TestHMACCheckTampered(t *testing.T) {
	a := HMACAuth{SecretKey: []byte("secret")}

	if err := a.Check("other-body", a.Sign("body", 0)); err == nil {
		t.Fatal("expected invalid sign error for tampered body")
	}

	if err := a.Check("body", a.Sign("body", 0)+"tampered"); err == nil {
		t.Fatal("expected invalid sign error for tampered signature")
	}
}

func TestHMACZeroExpiresNeverExpires(t *testing.T) {
	a := HMACAuth{SecretKey: []byte("secret")}

	sign := a.Sign("body", 0)
	if err := a.Check("body", sign); err != nil {
		t.Fatalf("expires=0 should never expire, got %v", err)
	}
}
