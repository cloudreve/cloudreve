package admin

import (
	"encoding/hex"
	"net/http"
	"strconv"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	request2 "github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/wopi"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/gin-gonic/gin"
	"github.com/go-mail/mail"
)

type (
	HashIDService struct {
		ID     int    `json:"id"`
		Type   int    `json:"type"`
		HashID string `json:"hash_id"`
	}
	HashIDParamCtx struct{}
)

func (service *HashIDService) Encode(c *gin.Context) (string, error) {
	dep := dependency.FromContext(c)
	res, err := dep.HashIDEncoder().Encode([]int{service.ID, service.Type})
	if err != nil {
		return "", err
	}
	return res, nil
}

func (service *HashIDService) Decode(c *gin.Context) (int, error) {
	dep := dependency.FromContext(c)
	res, err := dep.HashIDEncoder().Decode(service.HashID, service.Type)
	if err != nil {
		return 0, err
	}

	return res, nil
}

type (
	BsEncodeService struct {
		Bool []int `json:"bool"`
	}
	BsEncodeParamCtx struct{}
	BsEncodeRes      struct {
		Hex string
		B64 []byte
	}
)

func (service *BsEncodeService) Encode(c *gin.Context) (*BsEncodeRes, error) {
	bs := &boolset.BooleanSet{}
	for _, v := range service.Bool {
		boolset.Set(v, true, bs)
	}

	res, err := bs.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return &BsEncodeRes{
		Hex: hex.EncodeToString(res),
		B64: res,
	}, nil
}

type (
	BsDecodeService struct {
		Code string `json:"code"`
	}
	BsDecodeParamCtx struct{}
	BsDecodeRes      struct {
		Bool []int `json:"bool"`
	}
)

func (service *BsDecodeService) Decode(c *gin.Context) (*BsDecodeRes, error) {
	bs, err := boolset.FromString(service.Code)
	if err != nil {
		return nil, err
	}

	res := []int{}
	for i := 0; i < len(*bs)*8; i++ {
		if bs.Enabled(i) {
			res = append(res, i)
		}
	}

	return &BsDecodeRes{
		Bool: res,
	}, nil
}

type (
	FetchWOPIDiscoveryService struct {
		Endpoint string `form:"endpoint" binding:"required"`
	}
	FetchWOPIDiscoveryParamCtx struct{}
)

func (s *FetchWOPIDiscoveryService) Fetch(c *gin.Context) (*types.ViewerGroup, error) {
	dep := dependency.FromContext(c)
	requestClient := dep.RequestClient(request2.WithContext(c), request2.WithLogger(dep.Logger()))
	content, err := requestClient.Request("GET", s.Endpoint, nil).CheckHTTPResponse(http.StatusOK).GetResponse()
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "WOPI endpoint id unavailable", err)
	}

	vg, err := wopi.DiscoveryXmlToViewerGroup(content)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Failed to parse WOPI response", err)
	}

	return vg, nil
}

type (
	TestSMTPService struct {
		Settings map[string]string `json:"settings" binding:"required"`
		To       string            `json:"to" binding:"required,email"`
	}
	TestSMTPParamCtx struct{}
)

func (s *TestSMTPService) Test(c *gin.Context) error {
	port, err := strconv.Atoi(s.Settings["smtpPort"])
	if err != nil {
		return serializer.NewError(serializer.CodeParamErr, "Invalid SMTP port", err)
	}

	d := mail.NewDialer(s.Settings["smtpHost"], port, s.Settings["smtpUser"], s.Settings["smtpPass"])
	d.SSL = false
	if setting.IsTrueValue(s.Settings["smtpEncryption"]) {
		d.SSL = true
	}
	d.StartTLSPolicy = mail.OpportunisticStartTLS

	sender, err := d.Dial()
	if err != nil {
		return serializer.NewError(serializer.CodeInternalSetting, "Failed to connect to SMTP server: "+err.Error(), err)
	}

	m := mail.NewMessage()
	m.SetHeader("From", s.Settings["fromAdress"])
	m.SetAddressHeader("Reply-To", s.Settings["replyTo"], s.Settings["fromName"])
	m.SetHeader("To", s.To)
	m.SetHeader("Subject", "Cloudreve SMTP Test")
	m.SetBody("text/plain", "This is a test email from Cloudreve.")

	err = mail.Send(sender, m)
	if err != nil {
		return serializer.NewError(serializer.CodeInternalSetting, "Failed to send test email: "+err.Error(), err)
	}

	return nil
}

func ClearEntityUrlCache(c *gin.Context) {
	dep := dependency.FromContext(c)
	dep.KV().Delete(manager.EntityUrlCacheKeyPrefix)
}
