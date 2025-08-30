package dns

import (
	"bytes"
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"golang.org/x/net/html/charset"
)

type NetEaseProvider struct{}

func NewNetEaseProvider() *NetEaseProvider {
	return &NetEaseProvider{}
}

func (p *NetEaseProvider) Provider() string {
	return "NetEase"
}

func (p *NetEaseProvider) GetMyInternetInfomation() (*InternetInfomation, error) {
	// Create a Resty Client
	client := resty.New()

	resp, err := client.R().
		EnableTrace().
		Get("https://nstool.netease.com/")

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, errors.Errorf("failed to get internet information from NetEase url: %s, status code: %d", "https://nstool.netease.com/", resp.StatusCode())
	}

	//log.Infof("request trace: %v", resp.Request.TraceInfo())

	respBody := resp.Body()

	//log.Info(string(respBody))

	nstoolUrl := ""

	// 解析 HTML
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(respBody))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse NetEase response HTML: %s", string(respBody))
	}

	// 找 iframe 的 src
	doc.Find("iframe").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			nstoolUrl = src
		}
	})

	if nstoolUrl == "" {
		return nil, errors.Errorf("failed to find iframe src in NetEase response: %s", string(respBody))
	}

	newResp, err := client.R().Get(nstoolUrl)
	if err != nil {
		return nil, err
	}

	//log.Info(string(newResp.Body()))
	//log.Info(newResp.RawResponse.Header.Get("Content-Type"))

	reader, err := charset.NewReader(bytes.NewReader(newResp.Body()), newResp.RawResponse.Header.Get("Content-Type"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create charset reader for NetEase response")
	}
	body, _ := io.ReadAll(reader)

	resultList := strings.Split(string(body), "<br>")

	data := NewInternetInfomationFromLocal()

	data.Provider = p.Provider()
	data.TraceInfo = newResp.Request.TraceInfo()
	data.DetectData = string(body)

	for idx, _ := range resultList {
		switch {
		case strings.Contains(resultList[idx], "您的IP地址信息"):
			filedList := strings.Fields(resultList[idx])
			if len(filedList) >= 2 {
				data.PublicIpAddress = strings.TrimSpace(filedList[1])
			}

		case strings.Contains(resultList[idx], "您的DNS地址信息"):
			filedList := strings.Fields(resultList[idx])
			if len(filedList) >= 2 {
				data.PublicLocalDns = strings.TrimSpace(filedList[1])
			}

		}
	}

	return data, nil
}
