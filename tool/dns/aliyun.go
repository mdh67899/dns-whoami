package dns

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type AliyunProvider struct{}

func NewAliyunProvider() *AliyunProvider {
	return &AliyunProvider{}
}

func (a *AliyunProvider) Provider() string {
	return "aliyun"
}

type AliDnsDetectResponse struct {
	RetCode     int    `json:"retCode"`
	Description string `json:"description"`
	Content     struct {
		LocalIP     string `json:"localIp"`
		LDNS        string `json:"ldns"`
		IPOperator  string `json:"ipOperator"`
		IPProvince  string `json:"ipProvince"`
		IPCountry   string `json:"ipCountry"`
		DNSOperator string `json:"dnsOperator"`
		DNSProvince string `json:"dnsProvince"`
		DNSCountry  string `json:"dnsCountry"`
	} `json:"content"`
}

// GetMyInternetInfomation 从阿里云 CDN 检测接口获取当前互联网连接信息。
//
// 该方法通过调用阿里云的 DNS 检测接口，获取当前设备的公网 IP 地址、本地 DNS 服务器信息、
// 运营商及地理位置等数据，并封装为 InternetInfomation 结构体返回。
//
// 返回：
//   - *InternetInfomation: 包含公网 IP、本地 DNS、运营商等信息的结构体。
//   - error: 如果请求或解析过程中发生错误，返回错误信息。
//
// 注意：
//   - 该方法依赖于阿里云的 DNS 检测接口，需要确保网络连接正常。
//   - 返回的 InternetInfomation 结构体中包含请求的 Trace 信息和原始响应数据，便于调试。
//
// 示例：
//
//	info, err := provider.GetMyInternetInfomation()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Public IP: %s\n", info.PublicIpAddress)
func (a *AliyunProvider) GetMyInternetInfomation() (*InternetInfomation, error) {
	//https://be149f43-ceec-4199-9710-cdafed1df369.dns-detect.alicdn.com/api/detect/DescribeDNSLookup?cb=jQuery110109824510113197696_1756536536791&_=1756536536792
	aliDnsDelectUrlTpl := fmt.Sprintf("https://%s.dns-detect.alicdn.com/api/detect/DescribeDNSLookup?_=%d", uuid.New().String(), time.Now().UnixMilli())

	// Create a Resty Client
	client := resty.New()

	resp, err := client.R().
		EnableTrace().
		Get(aliDnsDelectUrlTpl)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, errors.Errorf("failed to get internet information from aliyun cdn detect url: %s, status code: %d", aliDnsDelectUrlTpl, resp.StatusCode())
	}

	respBody := resp.Body()

	//log.Info(string(respBody))

	/*
		{
		  "retCode": 200,
		  "description": "",
		  "content": {
		    "localIp": "58.246.230.75",
		    "ldns": "58.247.118.233",
		    "ipOperator": "unicom",
		    "ipProvince": "shanghai",
		    "ipCountry": "cn",
		    "dnsOperator": "unicom",
		    "dnsProvince": "shanghai",
		    "dnsCountry": "cn"
		  }
		}
	*/

	result := &AliDnsDetectResponse{}
	err = json.Unmarshal(respBody, result)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal response body: %s", string(respBody))
	}

	retData := NewInternetInfomationFromLocal()

	retData.Provider = a.Provider()
	retData.PublicIpAddress = result.Content.LocalIP
	retData.PublicLocalDns = result.Content.LDNS
	retData.TraceInfo = resp.Request.TraceInfo()
	retData.DetectData = string(respBody)

	return retData, nil
}
