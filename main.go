package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
)

type ConversionRequest struct {
	URL      string   `json:"url"`
	Selector string   `json:"selector"`
	Filters  []string `json:"filters"` // 改为切片以支持多个过滤器
}

type ConversionResponse struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

func convertHandler(w http.ResponseWriter, r *http.Request) {
	var req ConversionRequest

	if r.Method == "GET" {
		req.URL = r.URL.Query().Get("url")
		req.Selector = r.URL.Query().Get("selector")
		req.Filters = r.URL.Query()["filters"] // 获取多个filters参数
	} else if r.Method == "POST" {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	resp, err := http.Get(req.URL)
	if err != nil {
		http.Error(w, "Failed to fetch URL", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		http.Error(w, "Failed to parse HTML", http.StatusInternalServerError)
		return
	}

	var selection *goquery.Selection
	if req.Selector != "" {
		selection = doc.Find(req.Selector)
		log.Printf("Selected %d elements with selector: %s", selection.Length(), req.Selector)
	} else {
		selection = doc.Selection
		log.Println("No selector specified, using entire document")
	}

	// 打印初始HTML内容长度
	initialHtml, _ := selection.Html()
	log.Printf("Initial HTML length: %d", len(initialHtml))

	// 直接在selection上应用过滤器
	for _, filter := range req.Filters {
		beforeLength := selection.Length()
		selection.Find(filter).Remove()
		afterLength := selection.Length()
		log.Printf("Filter '%s' removed %d elements", filter, beforeLength-afterLength)
	}

	// 获取过滤后的HTML
	filteredHtml, err := selection.Html()
	if err != nil {
		http.Error(w, "Failed to extract filtered HTML", http.StatusInternalServerError)
		return
	}
	log.Printf("Filtered HTML length: %d", len(filteredHtml))

	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(filteredHtml)
	if err != nil {
		http.Error(w, "Failed to convert to Markdown", http.StatusInternalServerError)
		return
	}

	// 处理相对路径链接
	markdown = processRelativeLinks(markdown, req.URL)

	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	filename := parsedURL.Host + strings.ReplaceAll(parsedURL.Path, "/", "_")
	if filename == "" {
		filename = "index"
	}
	filename = strings.TrimSuffix(filename, filepath.Ext(filename)) + ".md"
	filename = strings.Map(func(r rune) rune {
		if r == '?' || r == '#' || r == ':' {
			return '_'
		}
		return r
	}, filename)

	response := ConversionResponse{
		Filename: filename,
		Content:  markdown,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 新增函数用于处理相对路径链接
func processRelativeLinks(markdown, baseURL string) string {
	lines := strings.Split(markdown, "\n")
	parsedBaseURL, _ := url.Parse(baseURL)

	for i, line := range lines {
		if strings.Contains(line, "](") && (strings.Contains(line, "../") || strings.Contains(line, "./")) {
			startIdx := strings.Index(line, "](") + 2
			endIdx := strings.Index(line[startIdx:], ")")
			if endIdx != -1 {
				link := line[startIdx : startIdx+endIdx]
				if strings.HasPrefix(link, "../") || strings.HasPrefix(link, "./") {
					absoluteURL := resolveRelativeURL(parsedBaseURL, link)
					lines[i] = strings.Replace(line, link, absoluteURL, 1)
				}
			}
		}
	}

	return strings.Join(lines, "\n")
}

func resolveRelativeURL(baseURL *url.URL, relPath string) string {
	relURL, err := url.Parse(relPath)
	if err != nil {
		return relPath
	}
	return baseURL.ResolveReference(relURL).String()
}

func main() {
	http.HandleFunc("/convert", convertHandler)
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// http://localhost:8080/convert?url=https://www.wangmingchang.com/5482.html&selector=.content
// http://localhost:8080/convert?url=https://www.wangmingchang.com/5482.html&selector=.content&filters=.post-actions&filters=.post-copyright&filters=.action-share&filters=.article-tags&filters=.article-nav&filters=.relates

//x小程序
// http://localhost:8080/convert?url=https://developers.weixin.qq.com/miniprogram/dev/api/ad/wx.createInterstitialAd.html&filters=.sidebar&filters=.navbar&filters=.subnavbar&filters=.footer&filters=.fixed-translate&selector=.main-container

// https://htmltomd.jianshe2.com/convert?url=https://developers.weixin.qq.com/miniprogram/dev/api/ad/wx.createInterstitialAd.html&filters=.sidebar&filters=.navbar&filters=.subnavbar&filters=.footer&filters=.fixed-translate&selector=.main-container
