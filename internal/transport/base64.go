package transport

import "encoding/base64"

func stdBase64Encode(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
