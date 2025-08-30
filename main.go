package main

import (
	"whoami/pkg"
	"whoami/tool/dns"
)

var (
	log = pkg.AppLog
)

func main() {
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
