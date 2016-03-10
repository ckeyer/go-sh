package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"net/http"
)

func MWHello(rw http.ResponseWriter, req *http.Request, ctx *RequestContext) {
	log.Debugf("hello %s:%s", ctx.req.Method, ctx.req.URL.Path)
}

func WMAuthGithubServer(rw http.ResponseWriter, req *http.Request, ctx *RequestContext) {
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Error("first read body error, ", err)
		return
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(data))

	key := []byte("asdf")
	mac := hmac.New(sha1.New, key)
	mac.Write(data)
	expectedMAC := mac.Sum(nil)
	if fmt.Sprintf("sha1=%x", expectedMAC) != req.Header.Get("X-Hub-Signature") {
		log.Warningf("the webhooks' sha1 from github should be %s, but now is %x",
			req.Header.Get("X-Hub-Signature"), expectedMAC)
	}
	log.Debugf("github server auth passing")
}