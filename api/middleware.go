package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ckeyer/sloth/admin"
	"github.com/ckeyer/sloth/gh"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
)

func CorsHandle(ctx *gin.Context) {
	ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type,Limt,Offset,Origin,Accept,X-Signature")
	ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE")
	ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	ctx.Writer.Header().Set("Access-Control-Max-Age", fmt.Sprint(24*time.Hour/time.Second))

	if ctx.Request.Method == "OPTIONS" {
		GinMessage(ctx, 200, "ok")
	}
}

func GinLogger(ctx *gin.Context) {
	start := time.Now()
	ctx.Next()

	logent := log.WithFields(log.Fields{
		"Method": ctx.Request.Method,
		"URL":    ctx.Request.URL.Path,
		"Remote": ctx.Request.RemoteAddr,
		"Status": ctx.Writer.Status(),
		"Period": fmt.Sprintf("%.6f", time.Now().Sub(start).Seconds()),
	})

	for _, prefix := range []string{API_PREFIX, WEB_HOOKS} {
		if strings.HasPrefix(ctx.Request.URL.Path, prefix) {
			logent.Info("bye jack.")
			return
		}
	}
	logent.Debug("bye jack.")
}

func MWRequireLogin(ctx *gin.Context) {
	xsign := ctx.Request.Header.Get("X-Signature")
	if xsign == "" {
		var err error
		xsign, err = ctx.Cookie("x-signature")
		if err != nil {
			GinError(ctx, 401, "Not Found Header X-Signature")
			return
		}
	}

	signSli := strings.Split(xsign, ":")
	if len(signSli) != 3 {
		GinError(ctx, 401, "Invalid Header X-Signature")
		return
	}

	apiKey, timestamp, sign := signSli[0], signSli[1], signSli[2]
	db := ctx.MustGet(CtxKeyMgoDB).(*mgo.Database)

	ua, err := admin.AuthSignature(db, apiKey, timestamp, sign)
	if err != nil {
		GinError(ctx, 401, "Invalid signature content")
		return
	}

	ctx.Set(CtxKeyUserAuth, ua)
}

// 需要管理员权限，需要在使用 MWRequireLogin 之后
func MWRequireAdmin(ctx *gin.Context) {
	cua, ok := ctx.Get(CtxKeyUserAuth)
	if !ok {
		log.Errorf("show use MWRequireLogin before")
		GinError(ctx, 401, "need login.")
		return
	}
	ua := cua.(*admin.UserAuth)

	db := ctx.MustGet(CtxKeyMgoDB).(*mgo.Database)
	u, err := admin.GetUser(db, ua.UserID)
	if err != nil {
		GinError(ctx, 500, "cannot find user.")
		return
	}

	if !u.IsAdmin() {
		GinError(ctx, 403, "required admin role.")
		return
	}

	ctx.Set(CtxKeyUser, u)
}

func MWAuthGithubServer(rw http.ResponseWriter, req *http.Request) {
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
		log.WithFields(log.Fields{
			"header":  req.Header.Get("X-Hub-Signature"),
			"compute": expectedMAC,
		}).Warn("Invalid X-Hub-Signature.")
	}
	log.Debugf("github server auth passing")
}

func MWLoadGithubApp(ctx *gin.Context) {
	db := ctx.MustGet(CtxKeyMgoDB).(*mgo.Database)
	ghappK := &gh.App{
		ClientID:     "gh_client_id",
		ClientSecret: "gh_client_secret",
		CallbackURL:  "gh_callback_url",
	}
	cid, err := admin.GetValue(db, ghappK.ClientID)
	if err != nil {
		GinError(ctx, 500, "require settings.", ghappK.ClientID)
		return
	}
	sec, err := admin.GetValue(db, ghappK.ClientSecret)
	if err != nil {
		GinError(ctx, 500, "require settings.", ghappK.ClientSecret)
		return
	}
	callback, err := admin.GetValue(db, ghappK.CallbackURL)
	if err != nil {
		GinError(ctx, 500, "require settings.", ghappK.CallbackURL)
		return
	}

	ctx.Set(CtxKeyGithubApp, &gh.App{
		ClientID:     cid,
		ClientSecret: sec,
		CallbackURL:  callback,
	})
}
