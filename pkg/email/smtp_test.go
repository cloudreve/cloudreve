package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
	mail "github.com/wneessen/go-mail"
)

func TestSMTPAuthType(t *testing.T) {
	assert.Equal(t, mail.SMTPAuthAutoDiscover, SMTPAuthType(""))
	assert.Equal(t, mail.SMTPAuthAutoDiscover, SMTPAuthType("autodiscover"))
	assert.Equal(t, mail.SMTPAuthAutoDiscover, SMTPAuthType("bogus"))

	assert.Equal(t, mail.SMTPAuthPlain, SMTPAuthType("plain"))
	assert.Equal(t, mail.SMTPAuthPlainNoEnc, SMTPAuthType("plain-noenc"))
	assert.Equal(t, mail.SMTPAuthLogin, SMTPAuthType("login"))
	assert.Equal(t, mail.SMTPAuthLoginNoEnc, SMTPAuthType("login-noenc"))
	assert.Equal(t, mail.SMTPAuthCramMD5, SMTPAuthType("cram-md5"))
	assert.Equal(t, mail.SMTPAuthSCRAMSHA1, SMTPAuthType("scram-sha-1"))
	assert.Equal(t, mail.SMTPAuthSCRAMSHA256, SMTPAuthType("scram-sha-256"))
	assert.Equal(t, mail.SMTPAuthXOAUTH2, SMTPAuthType("xoauth2"))
	assert.Equal(t, mail.SMTPAuthNoAuth, SMTPAuthType("noauth"))

	// Values are normalized before matching.
	assert.Equal(t, mail.SMTPAuthPlainNoEnc, SMTPAuthType(" Plain-NoEnc "))
}
