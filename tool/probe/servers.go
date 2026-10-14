package probe

import (
	"bufio"
	"net"
	"os/exec"
	"runtime"
	"strings"

	"github.com/miekg/dns"
)

// NormalizeServer 将 DNS 服务器地址统一为 host:port 形式，缺省端口为 53。
func NormalizeServer(server string) string {
	server = strings.TrimSpace(server)
	if server == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(server); err == nil {
		return server
	}
	return net.JoinHostPort(server, "53")
}

// DiscoverLocalDNSServers 返回本机系统当前生效的 DNS 服务器（host:port）。
//
// macOS 优先通过 `scutil --dns` 读取系统解析器（/etc/resolv.conf 在 macOS 上
// 往往不是真实来源）；其他平台及回退场景读取 /etc/resolv.conf。
func DiscoverLocalDNSServers() []string {
	if runtime.GOOS == "darwin" {
		if servers := discoverFromScutil(); len(servers) > 0 {
			return servers
		}
	}
	return discoverFromResolvConf()
}

func discoverFromResolvConf() []string {
	config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return nil
	}
	port := config.Port
	if port == "" {
		port = "53"
	}
	var servers []string
	for _, s := range config.Servers {
		servers = appendUnique(servers, net.JoinHostPort(s, port))
	}
	return servers
}

func discoverFromScutil() []string {
	out, err := exec.Command("scutil", "--dns").Output()
	if err != nil {
		return nil
	}

	var servers []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "nameserver[") {
			continue
		}
		_, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		servers = appendUnique(servers, NormalizeServer(value))
	}
	return servers
}

func appendUnique(list []string, item string) []string {
	item = strings.TrimSpace(item)
	if item == "" {
		return list
	}
	for _, v := range list {
		if v == item {
			return list
		}
	}
	return append(list, item)
}
