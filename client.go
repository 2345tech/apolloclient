package apolloclient

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	_delimiter      = "\n"
	_question       = "?"
	_defaultCluster = "default"
	_httpTimeout    = 60 * time.Second
)

type (
	GetConfigParam struct {
		AppID      string // 应用ID，必填
		Cluster    string // 集群名称，选填
		Namespace  string // 命名空间，必填
		Secret     string // 授权密钥，选填
		ReleaseKey string // 上一次的releaseKey，选填
		ClientIP   string // 应用部署机器IP，选填
	}

	GetNotificationsParam struct {
		AppID         string         // 应用ID，必填
		Cluster       string         // 集群名称，必填
		Secret        string         // 授权密钥，必填
		Notifications []Notification // 通知列表，必填
	}

	ConfigData struct {
		AppID      string            `json:"appId"`          // 应用ID
		Cluster    string            `json:"cluster"`        // 集群名称
		Namespace  string            `json:"namespaceName"`  // 命名空间
		Configs    map[string]string `json:"configurations"` // 配置项列表
		ReleaseKey string            `json:"releaseKey"`     // 当前的releaseKey
	}

	Notification struct {
		Namespace      string `json:"namespaceName"`  // 命名空间，必填
		NotificationID int64  `json:"notificationId"` // 通知ID，必填
	}
)

type Client struct {
	HttpClient *http.Client
	Request    *http.Request
	BaseURL    *url.URL
}

func NewClient(serverAddr string, httpClient *http.Client, request *http.Request) (*Client, error) {
	baseURL, err := url.Parse(serverAddr)
	if err != nil {
		return nil, err
	}

	return &Client{
		HttpClient: httpClient,
		Request:    request,
		BaseURL:    baseURL,
	}, nil
}

// GetConfig 通过不带缓存的Http接口从Apollo读取配置
func (c *Client) GetConfig(param *GetConfigParam) (ConfigData, error) {
	configData := ConfigData{}
	if err := c.checkGetConfigParam(param); err != nil {
		return configData, err
	}

	// URL: {config_server_url}/configs/{appId}/{clusterName}/{namespaceName}?releaseKey={releaseKey}&ip={clientIp}
	finalURL := *c.BaseURL
	finalURL.Path += fmt.Sprintf("/configs/%s/%s/%s", param.AppID, param.Cluster, param.Namespace)

	query := url.Values{}
	if param.ClientIP != "" {
		query.Add("ip", param.ClientIP)
	}

	if param.ReleaseKey != "" {
		query.Add("releaseKey", param.ReleaseKey)
	}

	response, err := c.requestApollo(param.AppID, param.Secret, finalURL, query)
	if err != nil {
		return configData, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotModified {
		return configData, nil
	}

	err = json.NewDecoder(response.Body).Decode(&configData)
	if err != nil {
		return configData, err
	}

	return configData, nil
}

// GetConfigCache 通过带缓存的Http接口从Apollo读取配置
func (c *Client) GetConfigCache(param *GetConfigParam) (ConfigData, error) {
	configData := ConfigData{}
	if err := c.checkGetConfigParam(param); err != nil {
		return configData, err
	}

	// URL: {config_server_url}/configfiles/json/{appId}/{clusterName}/{namespaceName}?ip={clientIp}
	finalURL := *c.BaseURL
	finalURL.Path += fmt.Sprintf("/configfiles/json/%s/%s/%s", param.AppID, param.Cluster, param.Namespace)

	query := url.Values{}
	if param.ClientIP != "" {
		query.Add("ip", param.ClientIP)
	}

	response, err := c.requestApollo(param.AppID, param.Secret, finalURL, query)
	if err != nil {
		return configData, err
	}
	defer response.Body.Close()

	configs := make(map[string]string)
	err = json.NewDecoder(response.Body).Decode(&configs)
	if err != nil {
		return configData, err
	}

	configData.AppID = param.AppID
	configData.Cluster = param.Cluster
	configData.Namespace = param.Namespace
	configData.Configs = configs
	return configData, nil
}

// GetNotifications 应用感知配置更新（默认最长阻塞60秒）
func (c *Client) GetNotifications(param *GetNotificationsParam) (bool, []Notification, error) {
	notifications := make([]Notification, 0)
	err := c.checkGetNotificationsParam(param)
	if err != nil {
		return false, notifications, err
	}

	data, err := json.Marshal(param.Notifications)
	if err != nil {
		return false, notifications, err
	}

	// URL: {config_server_url}/notifications/v2?appId={appId}&_cluster={clusterName}&notifications={notifications}
	finalURL := *c.BaseURL
	finalURL.Path += "/notifications/v2"

	query := url.Values{}
	query.Add("appId", param.AppID)
	query.Add("cluster", param.Cluster)
	query.Add("notifications", string(data))

	response, err := c.requestApollo(param.AppID, param.Secret, finalURL, query)
	if err != nil {
		return false, notifications, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotModified {
		return false, notifications, nil
	}

	err = json.NewDecoder(response.Body).Decode(&notifications)
	if err != nil {
		return false, notifications, err
	}

	return true, notifications, nil
}

// requestApollo 发起请求调用Apollo Config Service
func (c *Client) requestApollo(appId, secret string, finalURL url.URL, query url.Values) (*http.Response, error) {
	rawQuery := query.Encode()
	if rawQuery != "" {
		finalURL.RawQuery = rawQuery
	}

	var request *http.Request
	var err error
	if c.Request == nil {
		request, err = http.NewRequest(http.MethodGet, finalURL.String(), nil)
		if err != nil {
			return nil, err
		}
	} else {
		if request, err = http.NewRequestWithContext(c.Request.Context(), http.MethodGet, finalURL.String(), nil); err != nil {
			return nil, err
		}
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("Accept-Charset", "utf-8")

	if secret != "" {
		now := time.Now()
		sign, err := c.signature(secret, finalURL.String(), now)
		if err != nil {
			return nil, err
		}

		request.Header.Set("Authorization", fmt.Sprintf("ConfigService %s:%s", appId, sign))
		request.Header.Set("Timestamp", strconv.FormatInt(now.UnixNano()/int64(time.Millisecond), 10))
	}

	httpClient := c.HttpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
		httpClient.Timeout = _httpTimeout
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("httpClient.Do error %v, url is %v", err, finalURL.String())
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNotModified {
		defer response.Body.Close()
		return nil, fmt.Errorf("http request failed with status: %s", response.Status)
	}

	return response, nil
}

func (c *Client) signature(secret, uri string, timestamp time.Time) (string, error) {
	var err error
	uri, err = url2PathWithQuery(uri)
	if err != nil {
		return "", err
	}

	src := strconv.FormatInt(timestamp.UnixNano()/int64(time.Millisecond), 10) + _delimiter + uri
	h := hmac.New(sha1.New, []byte(secret))
	_, err = h.Write([]byte(src))
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func url2PathWithQuery(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	pathWithQuery := u.Path

	if len(u.RawQuery) > 0 {
		pathWithQuery += _question + u.RawQuery
	}

	return pathWithQuery, nil
}

func (c *Client) checkGetConfigParam(param *GetConfigParam) error {
	if param == nil {
		return errors.New("GetConfigParam nil")
	}

	if param.AppID == "" {
		return errors.New("GetConfigParam.AppID nil")
	}

	if param.Cluster == "" {
		param.Cluster = _defaultCluster
	}

	if param.Namespace == "" {
		return errors.New("GetConfigParam.Namespace nil")
	}

	if param.ClientIP != "" {
		ip := net.ParseIP(param.ClientIP)
		if ip.To4() == nil {
			return errors.New("GetConfigParam.ClientIP is not a ipv4")
		}
	}

	return nil
}

func (c *Client) checkGetNotificationsParam(param *GetNotificationsParam) error {
	if param == nil {
		return errors.New("GetNotificationsParam nil")
	}

	if param.AppID == "" {
		return errors.New("GetNotificationsParam.AppID nil")
	}

	if param.Cluster == "" {
		param.Cluster = _defaultCluster
	}

	if len(param.Notifications) == 0 {
		return errors.New("GetNotificationsParam.Notifications nil")
	}

	for _, n := range param.Notifications {
		if n.Namespace == "" {
			return errors.New("GetNotificationsParam.Notifications.Namespace nil")
		}
	}

	return nil
}
