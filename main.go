package main

import (
	"whoami/tool/dns"

	"go.uber.org/zap"
)

var (
	logger, _ = zap.NewProduction()
	log       = logger.Sugar()
)

func main() {
	data, err := dns.NewNetEaseProvider().GetMyInternetInfomation()
	if err != nil {
		log.Error(err)
	} else {
		log.Info(data.ToJsonString())
	}
}
