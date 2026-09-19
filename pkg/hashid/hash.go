package hashid

import (
	"context"
	"errors"
)
import "github.com/speps/go-hashids"

// ID类型
const (
	ShareID  = iota // 分享
	UserID          // 用户
	FileID          // 文件ID
	FolderID        // 目录ID
	TagID           // 标签ID
	PolicyID        // 存储策略ID
	SourceLinkID
	GroupID
	EntityID
	AuditLogID
	NodeID
	TaskID
	DavAccountID
	PaymentID
)

var (
	// ErrTypeNotMatch ID类型不匹配
	ErrTypeNotMatch = errors.New("mismatched ID type.")
)

type Encoder interface {
	Encode(v []int) (string, error)
	Decode(raw string, t int) (int, error)
}

// ObjectIDCtx define key for decoded hash ID.
type (
	ObjectIDCtx struct{}
	EncodeFunc  func(encoder Encoder, uid int) string
)

type hashEncoder struct {
	h *hashids.HashID
}

func New(salt string) (Encoder, error) {
	hd := hashids.NewData()
	hd.Salt = salt
	h, err := hashids.NewWithData(hd)
	if err != nil {
		return nil, err
	}

	return &hashEncoder{h: h}, nil
}

func (e *hashEncoder) Encode(v []int) (string, error) {
	id, err := e.h.Encode(v)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (e *hashEncoder) Decode(raw string, t int) (int, error) {
	res, err := e.h.DecodeWithError(raw)
	if err != nil {
		return 0, err
	}

	if len(res) != 2 || res[1] != t {
		return 0, ErrTypeNotMatch
	}
	return res[0], nil
}

func encodeID(encoder Encoder, id, t int) string {
	res, _ := encoder.Encode([]int{id, t})
	return res
}

// EncodeUserID encode user id to hash id
func EncodeUserID(encoder Encoder, uid int) string {
	return encodeID(encoder, uid, UserID)
}

// EncodeGroupID encode group id to hash id
func EncodeGroupID(encoder Encoder, uid int) string {
	return encodeID(encoder, uid, GroupID)
}

// EncodePaymentID encode payment id to hash id
func EncodePaymentID(encoder Encoder, uid int) string {
	return encodeID(encoder, uid, PaymentID)
}

// EncodeFileID encode file id to hash id
func EncodeFileID(encoder Encoder, uid int) string {
	return encodeID(encoder, uid, FileID)
}

// EncodeAuditLogID encode audit log id to hash id
func EncodeAuditLogID(encoder Encoder, uid int) string {
	return encodeID(encoder, uid, AuditLogID)
}

// EncodeTaskID encode task id to hash id
func EncodeTaskID(encoder Encoder, uid int) string {
	return encodeID(encoder, uid, TaskID)
}

// EncodeEntityID encode entity id to hash id
func EncodeEntityID(encoder Encoder, id int) string {
	return encodeID(encoder, id, EntityID)
}

// EncodeNodeID encode node id to hash id
func EncodeNodeID(encoder Encoder, id int) string {
	return encodeID(encoder, id, NodeID)
}

// EncodePolicyID encode policy id to hash id
func EncodePolicyID(encoder Encoder, id int) string {
	return encodeID(encoder, id, PolicyID)
}

// EncodeShareID encode share id to hash id
func EncodeShareID(encoder Encoder, id int) string {
	return encodeID(encoder, id, ShareID)
}

// EncodeDavAccountID encode dav account id to hash id
func EncodeDavAccountID(encoder Encoder, id int) string {
	return encodeID(encoder, id, DavAccountID)
}

// EncodeSourceLinkID encode source link id to hash id
func EncodeSourceLinkID(encoder Encoder, id int) string {
	return encodeID(encoder, id, SourceLinkID)
}

func FromContext(c context.Context) int {
	return c.Value(ObjectIDCtx{}).(int)
}
