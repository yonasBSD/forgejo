package incoming

import (
	"fmt"

	"github.com/emersion/go-sasl"

	"crypto/hmac"
	"crypto/md5"
)

const CRAM_MD5 = "CRAM-MD5"

type cramMD5Client struct {
	Username string
	Secret   string
}

func (client *cramMD5Client) Start() (mech string, ir []byte, err error) {
	mech = CRAM_MD5
	return
}

func (client *cramMD5Client) Next(challenge []byte) (response []byte, err error) {
	hash := hmac.New(md5.New, []byte(client.Secret))
	hash.Write(challenge)
	s := make([]byte, 0, hash.Size())
	return []byte(fmt.Sprintf("%s %x", client.Username, hash.Sum(s))), nil
}

func NewCramMD5Client(username, secret string) sasl.Client {
	return &cramMD5Client{username, secret}
}
