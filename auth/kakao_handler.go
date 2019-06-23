package auth

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/buger/jsonparser"
	"github.com/gin-gonic/gin"
	"github.com/itslaves/rentalgames-server/common/sessions"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

func KakaoOAuthCallback(c *gin.Context) {
	session := sessions.Session(c)

	webAddr := viper.GetString("web.addr")

	oauthConfig := &oauth2.Config{
		ClientID:     viper.GetString("oauth_kakao_client_id"),
		ClientSecret: viper.GetString("oauth_kakao_client_secret"),
		Endpoint: oauth2.Endpoint{
			AuthURL:  viper.GetString("oauth.kakao.authorizeURL"),
			TokenURL: viper.GetString("oauth.kakao.tokenURL"),
		},
		RedirectURL: viper.GetString("oauth.kakao.redirectURL"),
	}

	token, err := oauthConfig.Exchange(context.TODO(), c.Query("code"))
	if err != nil {
		// TODO: 에러 유형 파라미터로 전달
		location := fmt.Sprintf("http://%s/error", webAddr)
		c.Redirect(http.StatusFound, location)
	}
	if !token.Valid() {
		// TODO: 에러 유형 파라미터로 전달
		location := fmt.Sprintf("http://%s/error", webAddr)
		c.Redirect(http.StatusFound, location)
	}

	// TODO: MySQL DB 연동 처리
	dbUserID := "itslaves"
	// END MYSQL DB 연동

	// 세션의 유저 ID와 디비의 유저 ID가 동일하면 가입된 사용자로 판단
	alreadyJoined := false
	if sessionUserID, exist := session.Values["userID"]; exist {
		alreadyJoined = dbUserID == sessionUserID
	}

	if alreadyJoined {
		// 가입된 사용자는 세션에 정보를 업데이트하고 웹페이지 메인으로 이동
		session.Values["userID"] = dbUserID
		session.Values["accessToken"] = token.AccessToken
		session.Values["refreshToken"] = token.RefreshToken
		session.Save(c.Request, c.Writer)

		location := fmt.Sprintf("http://%s/", webAddr)
		c.Redirect(http.StatusFound, location)
	} else {
		// 리소스 서버로부터 사용자 정보를 가져온 뒤 회원가입 페이지로 이동
		client := oauthConfig.Client(context.TODO(), token)
		resp, err := client.Get(viper.GetString("oauth.kakao.userProfileURL"))
		if err != nil {
			// TODO: 에러 유형 파라미터로 전달
			location := fmt.Sprintf("http://%s/error", webAddr)
			c.Redirect(http.StatusFound, location)
		}
		defer resp.Body.Close()
		result, _ := ioutil.ReadAll(resp.Body)

		id, _ := jsonparser.GetInt(result, "id")
		nickname, _ := jsonparser.GetString(result, "properties", "nickname")
		profileImage, _ := jsonparser.GetString(result, "properties", "profile_image")
		gender, _ := jsonparser.GetString(result, "kakao_account", "gender")
		email, _ := jsonparser.GetString(result, "kakao_account", "email")

		params := url.Values{}
		params.Set("id", strconv.Itoa(int(id)))
		params.Set("nickname", nickname)
		params.Set("profileImage", profileImage)
		params.Set("gender", gender)
		params.Set("email", email)

		location := fmt.Sprintf("http://%s/join?%s", webAddr, params.Encode())
		c.Redirect(http.StatusFound, location)
	}
}
