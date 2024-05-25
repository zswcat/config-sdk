package config

import (
	"config-sdk/str"
	"errors"
	"fmt"
	"time"
)

type OpenApiAuth struct {
	JwtToken  string `json:"jwt_token"`
	ExpiredAt int64  `json:"expired_at"`
}

func getOpenApiAuth(host, application, envType, accessID, accessToken string) (*OpenApiAuth, error) {
	timestamp := time.Now().Add(10 * time.Second).Unix()
	body := map[string]interface{}{
		"application":               application,
		"env_type":                  envType,
		"access_id":                 accessID,
		"access_secret_result_sign": str.Get32Md5(fmt.Sprintf("%s%d", accessToken, timestamp)),
		"timestamp":                 timestamp,
	}

	result := struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    OpenApiAuth `json:"data"`
	}{}
	err := gout.
		POST(host + "/open_api/v1/oaa/get_access_jwt").
		SetJSON(body).
		BindJSON(&result).
		Do()

	if err != nil {
		return nil, err
	}

	// 判断结果是否正确
	if result.Code != 0 {
		return nil, errors.New("access token api err:" + result.Message)
	}

	return &result.Data, nil
}
