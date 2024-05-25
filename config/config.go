package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"zswcat/configsdk/cache"

	"github.com/guonaihong/gout"
)

const configApp = "_config"

const (
	Prod = "prod"
	Test = "test"
	Dev  = "dev"
)

type ClientConf struct {
	EnvType     string `json:"env_type"`
	Host        string `json:"host"`
	AccessID    string `json:"access_id"`
	AccessToken string `json:"access_token"`
}

type Client[T any] struct {
	authCache   *cache.ExpiredCache[OpenApiAuth] // 权限缓存
	configCache *cache.ReloadCache[T]            // 配置缓存
}

func (client *Client[T]) Get() *T {
	c, _ := client.configCache.Get()
	return c
}

func NewOpenApiClient(conf *ClientConf) (*cache.ExpiredCache[OpenApiAuth], error) {
	return cache.NewExpiredCache[OpenApiAuth](func() (*OpenApiAuth, int64, error) {
		openApiAuth, err1 := getOpenApiAuth(conf.Host, configApp, conf.EnvType, conf.AccessID, conf.AccessToken)
		if err1 != nil {
			fmt.Println("刷新密钥失败")
			return nil, 0, err1
		}

		return openApiAuth, openApiAuth.ExpiredAt - 50, nil
	})
}

func NewConfigClient[T any](openApiClient *cache.ExpiredCache[OpenApiAuth], conf *ClientConf, namespace, name string) (*Client[T], error) {
	var client = &Client[T]{}

	client.authCache = openApiClient

	var err error
	// 数据10s刷新一次
	client.configCache, err = cache.NewReloadCache[T](func() (*T, error) {
		token, err1 := client.authCache.Get()

		if err1 != nil {
			fmt.Println("更新配置失败，原因: 更新密钥失败: " + err1.Error())
			return nil, err1
		}

		config, err1 := getConfig(conf.Host, namespace, name, conf.EnvType, token.JwtToken)
		if err1 != nil {
			fmt.Println("更新配置失败: " + err1.Error())
			return nil, err1
		}

		var res = new(T)
		err1 = json.Unmarshal([]byte(config), res)
		if err1 != nil {
			fmt.Println("更新配置失败, 数据格式异常: " + config + err1.Error())
			return res, err1
		}

		return res, nil
	}, true, 20*time.Second)

	return client, err
}

func getConfig(host, namespace, name, envType, token string) (string, error) {
	body := map[string]interface{}{
		"namespace": namespace,
		"name":      name,
		"env":       envType,
	}

	result := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Current string `json:"current"`
		} `json:"data"`
	}{}
	err := gout.
		POST(host + "/open_api/v1/cm/get_config_item_env").
		SetHeader(gout.H{
			"Content-Type": "application/json",
			"Token":        token,
		}).
		SetJSON(body).
		BindJSON(&result).
		Do()

	if err != nil {
		return "", err
	}

	// 判断结果是否正确
	if result.Code != 0 {
		return "", errors.New("get config err:" + result.Message)
	}

	return result.Data.Current, nil
}
