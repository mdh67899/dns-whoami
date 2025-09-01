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
