package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/http2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	ROOT_URL        = "https://mangapill.com/manga/3258/one-piece-digital-colored-comics"
	MAX_CONCURRENCY = 16
)

var client = &http.Client{
	Transport: &http2.Transport{},
	Timeout:   30 * time.Second,
}

type ImageData struct {
	ChapterName string
	Filename    string
	URL         string
}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/118.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Edge/118.0.0.0",
}

func fetchHTML(url string) (*goquery.Document, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching %s: %d", url, resp.StatusCode)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

func scrapeChapters() []string {
	doc, err := fetchHTML(ROOT_URL)
	if err != nil {
		log.Fatal(err)
	}

	var links []string
	doc.Find("#chapters a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			fullLink := "https://mangapill.com" + href
			links = append([]string{fullLink}, links...)
		}
	})

	fmt.Println("Total Chapters Found:", len(links))
	return links
}

func scrapeAndDownload(chapterLink string, wg *sync.WaitGroup) {
	defer wg.Done()

	doc, err := fetchHTML(chapterLink)
	if err != nil {
		log.Println("Error fetching:", chapterLink)
		return
	}

	chapterTitle := cases.Title(language.English).String(strings.Replace(strings.Split(chapterLink, "/")[len(strings.Split(chapterLink, "/"))-1], "-", " ", -1))
	fmt.Println("📖 Scraping Chapter:", chapterTitle)

	imageCount := 0
	doc.Find("img.js-page").Each(func(i int, s *goquery.Selection) {
		imgSrc, exists := s.Attr("data-src")
		if exists && strings.Contains(imgSrc, "https://cdn.readdetectiveconan.com/") {
			ext := ".jpg"
			if strings.Contains(imgSrc, ".jpeg") {
				ext = ".jpeg"
			}

			pageNum := fmt.Sprintf("%03d%s", imageCount+1, ext)
			imageCount++

			downloadImage(ImageData{ChapterName: chapterTitle, Filename: pageNum, URL: imgSrc})
		}
	})

	fmt.Println("✅ Total Images Found in", chapterTitle, ":", imageCount)
}

func downloadImage(img ImageData) {
	folderPath := filepath.Join("One Piece", img.ChapterName)
	if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
		log.Println("❌ Failed to create folder:", folderPath, "Error:", err)
		return
	}

	filePath := filepath.Join(folderPath, img.Filename)
	fmt.Println("⬇️  Saving to:", filePath)

	// Retry logic with exponential backoff and random delay
	maxRetries := 5
	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", img.URL, nil)
		if err != nil {
			log.Println("❌ Failed to create request:", err)
			return
		}

		// Mimic a real browser
		req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
		req.Header.Set("Referer", ROOT_URL)

		resp, err := client.Do(req)
		if err != nil {
			log.Println("❌ Failed to download:", img.URL, "Error:", err)
			time.Sleep(time.Duration(rand.Intn(5)+attempt) * time.Second)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			file, err := os.Create(filePath)
			if err != nil {
				log.Println("❌ Failed to create file:", filePath, "Error:", err)
				return
			}
			defer file.Close()

			_, err = io.Copy(file, resp.Body)
			if err != nil {
				log.Println("❌ Error saving image:", err)
			}

			fmt.Printf("✅ Downloaded: %s/%s\n", img.ChapterName, img.Filename)
			return
		} else if resp.StatusCode == 403 {
			log.Println("🚫 Blocked! Retrying with delay...")
			time.Sleep(time.Duration(rand.Intn(10)+attempt*2) * time.Second)
		} else {
			log.Println("❌ Unexpected status:", resp.StatusCode, "URL:", img.URL)
		}
	}

	log.Println("❌ Max retries reached. Failed to download:", img.URL)
}

func main() {
	chapterLinks := scrapeChapters()
	var wg sync.WaitGroup

	for _, chapter := range chapterLinks {
		wg.Add(1)
		go scrapeAndDownload(chapter, &wg)
	}

	wg.Wait()
	fmt.Println("✅ All chapters downloaded successfully!")
}
