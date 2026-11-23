package jwt

import (
	"testing"
	"time"
)

func TestSignAndParse(t *testing.T) {
	m := NewManager(&JWTConfig{JWTSecret: "s3cret", JWTTTL: time.Minute})

	token, err := m.Sign(7, "ADMIN")
	if err != nil {
		t.Fatal(err)
	}

	c, err := m.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if c.UserID != 7 || c.Role != "ADMIN" {
		t.Fatalf("unexpected claims: %+v", c)
	}
}

func TestParseRejectsBadTokens(t *testing.T) {
	m := NewManager(&JWTConfig{JWTSecret: "s3cret", JWTTTL: time.Minute})
	other := NewManager(&JWTConfig{JWTSecret: "lain", JWTTTL: time.Minute})
	expired := NewManager(&JWTConfig{JWTSecret: "s3cret", JWTTTL: -time.Minute})

	wrongKey, _ := other.Sign(1, "USER")
	old, _ := expired.Sign(1, "USER")

	for name, tok := range map[string]string{
		"garbage":   "abc.def.ghi",
		"wrong key": wrongKey,
		"expired":   old,
	} {
		if _, err := m.Parse(tok); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
