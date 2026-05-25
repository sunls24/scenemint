package quota

import (
	"errors"

	"github.com/sunls24/gox/server"
)

func ResponseError(err error) error {
	if respErr, ok := InvalidFingerprintResponseError(err); ok {
		return respErr
	}
	return server.ErrMsg("额度服务暂时不可用").WithErr(err)
}

func InvalidFingerprintResponseError(err error) (error, bool) {
	if errors.Is(err, ErrInvalidFingerprint) {
		return server.ErrMsg("浏览器指纹不正确，请刷新页面后重试"), true
	}
	return nil, false
}
