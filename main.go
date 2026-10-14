package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"whoami/pkg"
	"whoami/tool/dns"
	"whoami/tool/probe"
)

var (
	log = pkg.AppLog
)

func main() {
	var (
		domain      = flag.String("domain", "", "要拨测的域名（设置后进入拨测模式；为空则执行 whoami 功能）")
		serversFlag = flag.String("servers", "223.5.5.5,223.6.6.6", "公共 DNS 服务器列表，逗号分隔（host 或 host:port）")
		dhcpServer  = flag.String("dhcp-server", "", "公司 DHCP/内网 DNS 服务器列表，逗号分隔（host 或 host:port）")
		debugLocal  = flag.Bool("debug-local", false, "同时拨测本机系统配置的 local dns")
		qtype       = flag.String("qtype", "A", "查询记录类型，如 A/AAAA/CNAME")
		interval    = flag.Duration("interval", time.Minute, "拨测间隔")
		timeout     = flag.Duration("timeout", 5*time.Second, "单次查询超时时间")
		output      = flag.String("output", "dns_probe.jsonl", "拨测结果存储文件（JSON Lines）")
		once        = flag.Bool("once", false, "仅拨测一次后退出")
	)
	flag.Parse()

	if *domain == "" {
		runWhoami()
		return
	}

	runProbe(*domain, buildServers(*serversFlag, *dhcpServer, *debugLocal), *qtype, *interval, *timeout, *output, *once)
}

func runWhoami() {
	akahelpData, err := dns.NewAkahelpProvider().GetMyInternetInfomation()
	if err != nil {
		log.Error(err)
	} else {
		log.Info(akahelpData.ToJsonString())
	}

	data, err := dns.NewNetEaseProvider().GetMyInternetInfomation()
	if err != nil {
		log.Error(err)
	} else {
		log.Info(data.ToJsonString())
	}

	aliData, err := dns.NewAliyunProvider().GetMyInternetInfomation()
	if err != nil {
		log.Error(err)
	} else {
		log.Info(aliData.ToJsonString())
	}
}

func runProbe(domain string, servers []probe.Server, qtype string, interval, timeout time.Duration, output string, once bool) {
	if len(servers) == 0 {
		log.Fatal("没有可用的拨测 DNS 服务器")
	}

	for _, s := range servers {
		log.Infof("拨测目标: %-22s [%s]", s.Address, s.Type)
	}

	store, err := probe.NewStore(output)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	prober := &probe.Prober{Servers: servers, Timeout: timeout}

	runOnce := func() {
		results := prober.Query(domain, qtype)
		if err := store.Append(results); err != nil {
			log.Errorf("存储拨测结果失败: %v", err)
		}
		for _, r := range results {
			if r.Error != "" {
				log.Warnf("[%s] %s %s -> 失败: %s", r.ServerType, r.Server, r.RCode, r.Error)
				continue
			}
			log.Infof("[%s] %s -> %s (%d 条, %.1fms)", r.ServerType, r.Server, summarize(r.Answers), len(r.Answers), r.RttMs)
		}
	}

	runOnce()
	if once {
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Infof("开始定时拨测，间隔 %s，结果写入 %s", interval, output)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("收到退出信号，停止拨测")
			return
		case <-ticker.C:
			runOnce()
		}
	}
}

func buildServers(publicFlag, dhcpFlag string, debugLocal bool) []probe.Server {
	var servers []probe.Server
	seen := make(map[string]bool)

	add := func(raw string, stype probe.ServerType) {
		addr := probe.NormalizeServer(raw)
		if addr == "" || seen[addr] {
			return
		}
		seen[addr] = true
		servers = append(servers, probe.Server{Address: addr, Type: stype})
	}

	for _, s := range splitList(dhcpFlag) {
		add(s, probe.ServerDHCP)
	}
	for _, s := range splitList(publicFlag) {
		add(s, probe.ServerPublic)
	}
	if debugLocal {
		for _, s := range probe.DiscoverLocalDNSServers() {
			add(s, probe.ServerLocal)
		}
	}
	return servers
}

func splitList(raw string) []string {
	var out []string
	for _, s := range strings.Split(raw, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func summarize(answers []probe.Answer) string {
	if len(answers) == 0 {
		return "空应答"
	}
	var vals []string
	for _, a := range answers {
		vals = append(vals, fmt.Sprintf("%s(%s)", a.Value, a.Type))
	}
	return strings.Join(vals, ", ")
}
