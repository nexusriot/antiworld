package processor

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"

	cf "github.com/nexusriot/antiworld3/internal/config"
	"github.com/nexusriot/antiworld3/internal/net"
	"github.com/nexusriot/antiworld3/internal/utils"
)

const (
	userAgent = "Chrome/39.0.2171.95 Safari/537.36"
)

type StopIterationError struct {
}

func (e *StopIterationError) Error() string {
	return fmt.Sprintf("stop iteration")
}

type DownloadFailedError struct {
}

func (e *DownloadFailedError) Error() string {
	return fmt.Sprintf("download failed")
}

type FileInfo struct {
	FileLink string
	FileName string
}

type Processor struct {
	Net *net.Net
	Cfg *cf.Config
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func NewProcessor(cfg *cf.Config, net *net.Net) *Processor {
	return &Processor{
		Net: net,
		Cfg: cfg,
	}
}

func (p *Processor) TotalPages() int {
	libUrl := p.Cfg.BaseUrl + "/lib"
	log.Infof("getting total pages, url: %s", libUrl)
	req, err := http.NewRequest("GET", libUrl, nil)
	if err != nil {
		log.Fatalf("failed create request to enumerate pages %s", err.Error())
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := p.Net.Client.Do(req)
	if err != nil {
		log.Fatalf("failed do request to enumerate pages (url: %s) %s", libUrl, err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatalf("failed create document reader to enumerate pages (url: %s) %s", libUrl, err.Error())
	}
	lis := doc.Find("div .pagination").Find("li")
	aNodes := lis.Find("a").Nodes
	ref := aNodes[len(aNodes)-2].Attr[0].Val
	fields := strings.FieldsFunc(strings.TrimSpace(ref), utils.SplitFunc)
	maxPages, err := strconv.Atoi(fields[1])
	if err != nil {
		log.Fatalf("failed to convert to enumerate pages %s %s", fields[1], err.Error())
	}
	log.Infof("Got total pages: %d", maxPages)
	return maxPages
}

func (p *Processor) ProcessPages(totalPages int, pages chan int, links chan *FileInfo) {

	closeChans := func() {
		close(links)
		close(pages)
	}

	for pag := 1; pag <= totalPages; pag++ {
		pageUrl := fmt.Sprintf("%s/lib/%d", p.Cfg.BaseUrl, pag)
		log.Infof("processing page %d: %s", pag, pageUrl)
		req, err := http.NewRequest("GET", pageUrl, nil)
		if err != nil {
			log.Fatalf("failed to create request for %s", pageUrl)
		}
		req.Header.Set("User-Agent", userAgent)
		resp, err := p.Net.Client.Do(req)

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.Fatalf("failed to create document reader for %s", pageUrl)
		}

		var pageLinks []string
		pageLinks = make([]string, 0)
		doc.Find(".news_body").Each(func(i int, s *goquery.Selection) {
			s.Find("a").Each(func(i int, s *goquery.Selection) {
				class, _ := s.Attr("id")
				matched, _ := regexp.MatchString(`download_book_*`, class)
				if matched {
					href, ok := s.Attr("href")
					if ok {
						pageLinks = append(pageLinks, href)
					}
				}
			})
		})
		for _, v := range pageLinks {
			fileInfo, err := p.download(v)
			if err != nil {
				if _, ok := err.(*StopIterationError); ok {
					closeChans()
					return
				}
				if _, ok := err.(*DownloadFailedError); ok {
					continue
				}
			}
			log.Printf("downloaded file %s using link %s", fileInfo.FileName, fileInfo.FileLink)
			links <- fileInfo
			pages <- pag
		}
	}
	closeChans()
}

func (p *Processor) download(fileLink string) (*FileInfo, error) {
	log.Infof("downloading file %s", fileLink)
	req, err := http.NewRequest("GET", p.Cfg.BaseUrl+fileLink, nil)
	if err != nil {
		log.Errorf("failed to create request for file %s because of %s", fileLink, err.Error())
		return nil, &DownloadFailedError{}
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := p.Net.Client.Do(req)

	if err != nil {
		log.Errorf("failed to complete request for file %s because of %s", fileLink, err.Error())
		return nil, &DownloadFailedError{}
	}
	defer resp.Body.Close()

	after := func(value string, a string) string {
		pos := strings.LastIndex(value, a)
		if pos == -1 {
			return ""
		}
		adjustedPos := pos + len(a)
		if adjustedPos >= len(value) {
			return ""
		}
		return value[adjustedPos:len(value)]
	}
	fileName := after(resp.Header["Content-Disposition"][0], "filename=")
	if fileName == "" {
		log.Errorf("failed to get file name for %s", fileLink)
		return nil, &DownloadFailedError{}
	}
	localFileName := p.Cfg.DownloadFolder + fileName
	log.Infof("file name for %s: %s", fileLink, localFileName)

	if fileExists(localFileName) {
		log.Infof("file %s exists, task completed", localFileName)
		return nil, &StopIterationError{}
	}
	out, err := os.Create(localFileName)
	if err != nil {
		log.Fatalf("failed to create local file %s because of %s", localFileName, err.Error())
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Fatalf("failed to copy string: %s", err.Error())
	}
	return &FileInfo{
		FileLink: fileLink,
		FileName: fileName,
	}, nil
}
