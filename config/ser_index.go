package config

import (
	"errors"
	"fmt"
	"github.com/guonaihong/gout"
	"github.com/zswcat/configsdk/cache"
	"time"
)

const serApp = "_ser"

type SerIndexClient[T any] struct {
	authCache     *cache.ExpiredCache[OpenApiAuth] // 权限缓存
	configCache   *cache.ReloadCache[T]            // 配置缓存
	serIndexCache SerIndexCache                    // 实例ID配置
}

type SerIndexCache struct {
	IndexID int    `json:"index_id"`
	Token   string `json:"token"`
}

func (client *SerIndexClient[T]) Get() *T {
	c, _ := client.configCache.Get()
	return c
}

func NewSerOpenApiClient(conf *ClientConf) (*cache.ExpiredCache[OpenApiAuth], error) {
	return cache.NewExpiredCache[OpenApiAuth](func() (*OpenApiAuth, int64, error) {
		openApiAuth, err1 := getOpenApiAuth(conf.Host, serApp, conf.EnvType, conf.AccessID, conf.AccessToken)
		if err1 != nil {
			fmt.Println("刷新密钥失败")
			return nil, 0, err1
		}

		return openApiAuth, openApiAuth.ExpiredAt - 50, nil
	})
}

func NewSerIndexClient[T any](openApiClient *cache.ExpiredCache[OpenApiAuth], conf *ClientConf, ser string, mapperFunc func(index int) T) (*SerIndexClient[T], error) {
	var client = &SerIndexClient[T]{}

	client.authCache = openApiClient

	var err error
	// 数据10s刷新一次
	client.configCache, err = cache.NewReloadCache[T](func() (*T, error) {
		token, err1 := client.authCache.Get()

		if err1 != nil {
			fmt.Println("更新配置失败，原因: 更新密钥失败: " + err1.Error())
			return nil, err1
		}

		serIndexCache, err1 := getSerIndex(conf.Host, ser, conf.EnvType, token.JwtToken, client.serIndexCache)
		if err1 != nil {
			fmt.Println("更新serIndex失败: " + err1.Error())
			return nil, err1
		}

		client.serIndexCache = serIndexCache

		t := mapperFunc(serIndexCache.IndexID)

		return &t, nil
	}, true, 20*time.Second)

	return client, err
}

func getSerIndex(host, ser, envType, token string, serIndexCache SerIndexCache) (SerIndexCache, error) {
	body := map[string]interface{}{
		"serve_name": ser,
		"env_type":   envType,
		"index_id":   serIndexCache.IndexID,
		"token":      serIndexCache.Token,
	}

	result := struct {
		Code    int           `json:"code"`
		Message string        `json:"message"`
		Data    SerIndexCache `json:"data"`
	}{}
	err := gout.
		POST(host + "/open_api/v1/ser/get_serve_index").
		SetHeader(gout.H{
			"Content-Type": "application/json",
			"Token":        token,
		}).
		SetJSON(body).
		BindJSON(&result).
		Do()

	if err != nil {
		return SerIndexCache{}, err
	}

	// 判断结果是否正确
	if result.Code != 0 {
		return SerIndexCache{}, errors.New("get ser_index err:" + result.Message)
	}

	return result.Data, nil
}
