package dns

import (
	"encoding/json"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

type AkahelpProvider struct{}

func NewAkahelpProvider() *AkahelpProvider {
	return &AkahelpProvider{}
}

func (a *AkahelpProvider) Provider() string {
	return "akahelp"
}

func (a *AkahelpProvider) GetMyInternetInfomation() (*InternetInfomation, error) {
	// 1. 读取系统的本地 DNS 配置
	config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return nil, errors.Wrap(err, "读取 resolv.conf 失败")
	}
	if len(config.Servers) == 0 {
		return nil, errors.New("未找到任何 DNS server")
	}

	server := config.Servers[0] + ":" + config.Port

	// 2. 构造一个 DNS 查询（TXT 记录）
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn("whoami.ipv4.akahelp.net"), dns.TypeTXT)

	// 3. 发起查询
	c := new(dns.Client)
	in, _, err := c.Exchange(m, server)
	if err != nil {
		return nil, errors.Wrap(err, "查询失败")
	}

	answerData, _ := json.Marshal(in)

	retData := NewInternetInfomationFromLocal()

	retData.Provider = a.Provider()
	retData.DetectData = string(answerData)

	// 4. 打印结果
	for _, ans := range in.Answer {
		if t, ok := ans.(*dns.TXT); ok {
			if len(t.Txt) >= 2 {
				if t.Txt[0] == "ns" {
					retData.PublicLocalDns = t.Txt[1]
				}
			}
		}
	}

	return retData, nil
}
