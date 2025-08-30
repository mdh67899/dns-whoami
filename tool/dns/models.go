package dns

import (
	"encoding/json"
	"net"
	"whoami/pkg"

	"github.com/go-resty/resty/v2"
	"github.com/miekg/dns"
)

var (
	log = pkg.AppLog
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

func NewInternetInfomationFromLocal() *InternetInfomation {
	// 1. 读取系统的本地 DNS 配置
	config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		log.Errorf("读取 resolv.conf 失败: %v", err)
		return &InternetInfomation{}
	}
	if len(config.Servers) == 0 {
		log.Warnf("读取 resolv.conf 失败: %v", err)
		return &InternetInfomation{}
	}

	retData := &InternetInfomation{
		Provider:  "local",
		DnsServer: config.Servers,
	}

	// 2. 读取本地网卡的 IP 地址
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Errorf("net.InterfaceAddrs 失败: %v", err)
		return retData
	}

	for _, addr := range addrs {
		// 取出 IP 地址
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			// 只要 IPv4
			if ipNet.IP.To4() != nil {
				retData.IpAddress = ipNet.IP.String()
			}
		}
	}

	return retData
}
