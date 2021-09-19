package tool

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	url2 "net/url"
	"strings"
)

type GetRequest struct {
	Url      string
	Header   map[string]interface{}
	Query    map[string]interface{}
	Cookie   map[string]string
	Redirect bool
}

type PostRequest struct {
	Url      string
	Header   map[string]interface{}
	Query    map[string]interface{}
	Body     map[string]interface{}
	Cookie   map[string]string
	Redirect bool
}

type HttpOptions struct {
	RedirectCookieJar bool
}

type httP struct { //HTTP操作工具包
	DefaultHeader map[string]string //默认爬虫header
	Options       HttpOptions
}

var HTTP = httP{
	DefaultHeader: map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.212 Safari/537.36",
	},
}

// GenRequest 生成请求 底层函数
func (a *httP) GenRequest(Type string, url string, header map[string]interface{}, query map[string]interface{}, body map[string]interface{}, cookies map[string]string) (*http.Request, error) {
	//表单
	var form string
	if body != nil {
		var data = make(url2.Values)
		for k, v := range body {
			data[k] = []string{fmt.Sprint(v)}
		}
		form = data.Encode()

		if _, ok := header["Content-Type"]; !ok {
			if header == nil {
				header = make(map[string]interface{}, 1)
			}
			header["Content-Type"] = "application/x-www-form-urlencoded; charset=utf-8"
		}
	}

	req, err := http.NewRequest(Type, url, strings.NewReader(form))
	if err != nil {
		return nil, err
	}

	//请求头
	for k, v := range a.DefaultHeader {
		req.Header.Add(k, v)
	}
	for k, v := range header {
		req.Header.Add(k, fmt.Sprint(v))
	}

	//url参数
	q := req.URL.Query()
	for k, v := range query {
		q.Add(k, fmt.Sprint(v))
	}
	req.URL.RawQuery = q.Encode()

	//cookie
	for k, v := range cookies {
		req.AddCookie(&http.Cookie{
			Name:  k,
			Value: v,
		})
	}

	return req, nil
}

// DefaultReader 执行请求获得io reader的默认流程
func (a *httP) DefaultReader(Type string, url string, header map[string]interface{}, query map[string]interface{}, body map[string]interface{}, cookies map[string]string, redirect bool) (http.Header, io.ReadCloser, error) {
	req, e := a.GenRequest(Type, url, header, query, body, cookies)
	if e != nil {
		return nil, nil, e
	}

	var client = *http.DefaultClient

	if !redirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else if a.Options.RedirectCookieJar {
		jar, e := cookiejar.New(nil)
		if e != nil {
			return nil, nil, e
		}
		client.Jar = jar
		if cookies != nil {
			client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				u, _ := url2.Parse(url)
				for _, v := range jar.Cookies(u) {
					cookies[v.Name] = v.Value
				}
				return nil
			}
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	return resp.Header, resp.Body, nil
}

// PostReader 执行POST请求，获得io reader
func (a *httP) PostReader(r *PostRequest) (http.Header, io.ReadCloser, error) {
	return a.DefaultReader("POST", r.Url, r.Header, r.Query, r.Body, r.Cookie, r.Redirect)
}

// GetReader 执行GET请求，获得io reader
func (a *httP) GetReader(r *GetRequest) (http.Header, io.ReadCloser, error) {
	return a.DefaultReader("GET", r.Url, r.Header, r.Query, nil, r.Cookie, r.Redirect)
}

func (*httP) ReadResBodyToByte(i io.ReadCloser) ([]byte, error) {
	defer func() {
		_ = i.Close()
	}()
	return ioutil.ReadAll(i)
}

func (a *httP) ReadResBodyToString(i io.ReadCloser) (string, error) {
	d, e := a.ReadResBodyToByte(i)
	return string(d), e
}

// DecodeResBodyToMap 读取io reader中返回的json写入map
func (a *httP) DecodeResBodyToMap(i io.ReadCloser) (map[string]interface{}, error) {
	var t map[string]interface{}

	//读取
	data, err := a.ReadResBodyToByte(i)
	if err != nil {
		return nil, err
	}

	//解码
	if err = json.Unmarshal(data, &t); err != nil {
		return nil, err
	}

	return t, nil
}

// Post 表单请求快捷方式
func (a *httP) Post(r *PostRequest) (http.Header, map[string]interface{}, error) {
	d, b, e := a.PostReader(r)
	if e != nil {
		return nil, nil, e
	}
	c, e := a.DecodeResBodyToMap(b)
	return d, c, nil
}

// Get 表单请求快捷方式
func (a *httP) Get(r *GetRequest) (http.Header, map[string]interface{}, error) {
	d, b, e := a.GetReader(r)
	if e != nil {
		return nil, nil, e
	}
	c, e := a.DecodeResBodyToMap(b)
	return d, c, nil
}

func (a *httP) PostBytes(r *PostRequest) (http.Header, []byte, error) {
	d, b, e := a.PostReader(r)
	if e != nil {
		return nil, nil, e
	}
	c, e := a.ReadResBodyToByte(b)
	return d, c, nil
}

func (a *httP) GetBytes(r *GetRequest) (http.Header, []byte, error) {
	d, b, e := a.GetReader(r)
	if e != nil {
		return nil, nil, e
	}
	c, e := a.ReadResBodyToByte(b)
	return d, c, nil
}

func (a *httP) PostString(r *PostRequest) (http.Header, string, error) {
	d, b, e := a.PostReader(r)
	if e != nil {
		return nil, "", e
	}
	c, e := a.ReadResBodyToString(b)
	return d, c, nil
}

func (a *httP) GetString(r *GetRequest) (http.Header, string, error) {
	d, b, e := a.GetReader(r)
	if e != nil {
		return nil, "", e
	}
	c, e := a.ReadResBodyToString(b)
	return d, c, nil
}

func (a httP) DefaultGoquery(Type string, url string, header map[string]interface{}, query map[string]interface{}, body map[string]interface{}, cookie map[string]string, redirect bool) (*goquery.Document, error) {
	_, resp, e := a.DefaultReader(Type, url, header, query, body, cookie, redirect)
	if e != nil {
		return nil, e
	}
	d, e := goquery.NewDocumentFromReader(resp)
	_ = resp.Close()
	return d, e
}

func (a httP) GetGoquery(r *GetRequest) (*goquery.Document, error) {
	return a.DefaultGoquery("GET", r.Url, r.Header, r.Query, nil, r.Cookie, r.Redirect)
}

func (a httP) PostGoquery(r *PostRequest) (*goquery.Document, error) {
	return a.DefaultGoquery("POST", r.Url, r.Header, r.Query, r.Body, r.Cookie, r.Redirect)
}
