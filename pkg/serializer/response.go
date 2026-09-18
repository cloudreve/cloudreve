package serializer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"
)

// Response 基础序列化器
type Response struct {
	Code            int         `json:"code"`
	Data            interface{} `json:"data,omitempty"`
	AggregatedError interface{} `json:"aggregated_error,omitempty"`
	Msg             string      `json:"msg"`
	Error           string      `json:"error,omitempty"`
	CorrelationID   string      `json:"correlation_id,omitempty"`
}

// NewResponseWithGobData 返回Data字段使用gob编码的Response
func NewResponseWithGobData(c context.Context, data interface{}) Response {
	var w bytes.Buffer
	encoder := gob.NewEncoder(&w)
	if err := encoder.Encode(data); err != nil {
		return ErrWithDetails(c, CodeInternalSetting, "Failed to encode response content", err)
	}

	return Response{Data: w.Bytes()}
}

// DecodeGob 将 Response 正文解码至目标指针
func (r *Response) DecodeGob(target interface{}) {
	src := r.Data.(string)
	raw := make([]byte, len(src)*len(src)/base64.StdEncoding.DecodedLen(len(src)))
	base64.StdEncoding.Decode(raw, []byte(src))
	decoder := gob.NewDecoder(bytes.NewBuffer(raw))
	decoder.Decode(target)
}
