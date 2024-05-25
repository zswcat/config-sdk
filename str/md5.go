package str

import (
	"crypto/md5"
	"encoding/hex"
)

func Get32Md5(str string) string {
	h1 := md5.New()
	h1.Write([]byte(str))
	return hex.EncodeToString(h1.Sum(nil))
}
