package tool

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"
)

const LF = "\n"

var httpClient *http.Client

func DoGet(domain string, apiPath string, queries map[string]string, headers map[string]string) ([]byte, error) {
	method := http.MethodGet
	url := buildUrl(domain, apiPath, queries)
	return exec(method, url, headers, nil)
}

func DoPost(domain string, apiPath string, body interface{}, queries map[string]string, headers map[string]string) ([]byte, error) {
	method := http.MethodPost
	url := buildUrl(domain, apiPath, queries)
	return exec(method, url, headers, body)
}

func DoPatch(domain string, apiPath string, body interface{}, queries map[string]string, headers map[string]string) ([]byte, error) {
	method := http.MethodPatch
	url := buildUrl(domain, apiPath, queries)
	return exec(method, url, headers, body)
}

func DoPut(domain string, apiPath string, body interface{}, queries map[string]string, headers map[string]string) ([]byte, error) {
	method := http.MethodPut
	url := buildUrl(domain, apiPath, queries)
	return exec(method, url, headers, body)
}

func DoDelete(domain string, apiPath string, queries map[string]string, headers map[string]string) ([]byte, error) {
	method := http.MethodDelete
	url := buildUrl(domain, apiPath, queries)
	return exec(method, url, headers, nil)
}

func exec(method string, url string, headers map[string]string, httpBody interface{}) ([]byte, error) {
	request, err := http.NewRequest(method, url, nil)
	if httpBody != nil {
		jsonBody, _ := json.Marshal(httpBody)
		reader := bytes.NewReader(jsonBody)
		if err != nil {
			fmt.Println(err.Error())
		}
		request, err = http.NewRequest(method, url, reader)
	}
	if err != nil {
		fmt.Println(err.Error())
	}
	request.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		request.Header.Add(k, v)
	}
	res, err := getHttpClient().Do(request)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	if res.StatusCode >= 300 {
		fmt.Println(map[string]string{"status_code": fmt.Sprint(res.StatusCode), "error_body": string(body)})
		return nil, errors.New(string(body))
	}

	return body, nil
}

func getHttpClient() *http.Client {
	if httpClient != nil {
		return httpClient
	}
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	httpClient = &http.Client{
		Transport: tr,
	}
	return httpClient
}

func buildUrl(domain string, apiPath string, queries map[string]string) string {
	queryString := buildQueryString(queries)
	formatString := "%s%s?%s"
	if queryString == "" {
		formatString = "%s%s%s"
	}
	url := fmt.Sprintf(formatString, domain, apiPath, queryString)
	return url
}

func buildQueryString(queries map[string]string) string {
	queryString := ""
	if queries == nil {
		return queryString
	}
	var keys []string
	for k := range queries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		itemString := fmt.Sprintf("%s=%s", k, queries[k])
		queryString = queryString + itemString + "&"
	}
	queryString = strings.Trim(queryString, "&")
	return queryString
}
