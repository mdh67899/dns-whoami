package probe

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

// ServerType 表示被拨测 DNS 服务器的来源类别。
type ServerType string

const (
	ServerDHCP   ServerType = "dhcp"   // 公司 DHCP/内网下发的 DNS
	ServerLocal  ServerType = "local"  // 本机系统配置的 local dns
	ServerPublic ServerType = "public" // 公共 DNS，如阿里云 223.5.5.5
)

// Server 描述一个被拨测的 DNS 服务器。
type Server struct {
	Address string     `json:"address"` // host:port
	Type    ServerType `json:"type"`
}

// Answer 是一条解析结果记录。
type Answer struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	TTL   uint32 `json:"ttl"`
	Value string `json:"value"`
}

// Result 是一次针对单个 DNS 服务器的拨测结果。
type Result struct {
	Timestamp  time.Time  `json:"timestamp"`
	Domain     string     `json:"domain"`
	QType      string     `json:"qtype"`
	Server     string     `json:"server"`
	ServerType ServerType `json:"server_type"`
	RCode      string     `json:"rcode"`
	Truncated  bool       `json:"truncated"`
	Answers    []Answer   `json:"answers"`
	RttMs      float64    `json:"rtt_ms"`
	Error      string     `json:"error,omitempty"`
}

// Prober 负责向一组 DNS 服务器并发拨测指定域名。
type Prober struct {
	Servers []Server
	Timeout time.Duration
}

// Query 并发向所有服务器查询同一域名，返回每个服务器的一条结果。
func (p *Prober) Query(domain, qtype string) []Result {
	qtypeCode, ok := dns.StringToType[strings.ToUpper(qtype)]
	if !ok {
		qtypeCode = dns.TypeA
	}

	results := make([]Result, len(p.Servers))
	var wg sync.WaitGroup
	for i, server := range p.Servers {
		wg.Add(1)
		go func(i int, server Server) {
			defer wg.Done()
			results[i] = p.queryOne(server, domain, qtypeCode)
		}(i, server)
	}
	wg.Wait()

	return results
}

func (p *Prober) queryOne(server Server, domain string, qtype uint16) Result {
	res := Result{
		Timestamp:  time.Now(),
		Domain:     domain,
		QType:      dns.TypeToString[qtype],
		Server:     server.Address,
		ServerType: server.Type,
		Answers:    []Answer{},
	}

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), qtype)
	m.RecursionDesired = true

	timeout := p.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	c := &dns.Client{Net: "udp", Timeout: timeout}
	in, rtt, err := c.Exchange(m, server.Address)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.RttMs = float64(rtt.Microseconds()) / 1000.0

	// UDP 被截断时回退到 TCP 重试。
	if in.Truncated {
		res.Truncated = true
		tc := &dns.Client{Net: "tcp", Timeout: timeout}
		tcpIn, tcpRtt, tcpErr := tc.Exchange(m, server.Address)
		if tcpErr != nil {
			res.Error = errors.Wrap(tcpErr, "tcp retry after truncation").Error()
			return res
		}
		in = tcpIn
		res.RttMs = float64(tcpRtt.Microseconds()) / 1000.0
	}

	res.RCode = rcodeToString(in.Rcode)
	res.Answers = extractAnswers(in.Answer)
	if res.RCode != "NOERROR" {
		res.Error = fmt.Sprintf("rcode=%s", res.RCode)
	}

	return res
}

func rcodeToString(rcode int) string {
	if s, ok := dns.RcodeToString[rcode]; ok {
		return s
	}
	return fmt.Sprintf("RCODE%d", rcode)
}

func extractAnswers(records []dns.RR) []Answer {
	out := make([]Answer, 0, len(records))
	for _, rr := range records {
		h := rr.Header()
		ans := Answer{
			Name: h.Name,
			Type: dns.TypeToString[h.Rrtype],
			TTL:  h.Ttl,
		}

		switch v := rr.(type) {
		case *dns.A:
			ans.Value = v.A.String()
		case *dns.AAAA:
			ans.Value = v.AAAA.String()
		case *dns.CNAME:
			ans.Value = v.Target
		case *dns.NS:
			ans.Value = v.Ns
		case *dns.PTR:
			ans.Value = v.Ptr
		case *dns.TXT:
			ans.Value = strings.Join(v.Txt, " ")
		case *dns.MX:
			ans.Value = fmt.Sprintf("%d %s", v.Preference, v.Mx)
		default:
			ans.Value = rr.String()
		}

		out = append(out, ans)
	}
	return out
}
