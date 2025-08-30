package dns

import (
	"encoding/json"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

var (
	logger, _ = zap.NewProduction()
	log       = logger.Sugar()
)

type InternetInfomation struct {
	Provider        string   // 监测工具所使用的dns监测工具提供商
	IpAddress       string   // 监测工具所在网络的公网出口ip
	DnsServer       []string // 监测工具所使用的本地局域网的dns服务器
	PublicIpAddress string   // 监测工具所使用的公共dns服务器解析到的公网出口ip
	PublicLocalDns  string   // 监测工具所在网络的互联网服务商localdns

	TraceInfo  resty.TraceInfo `json:"-"` // 监测工具所使用的dns监测工具提供商的请求追踪信息
	DetectData string          `json:"-"` // 监测工具返回的原始数据
}

func (i *InternetInfomation) ToJsonString() string {
	data, _ := json.Marshal(i)
	return string(data)
}
