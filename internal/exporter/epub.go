// Package exporter 提供 EPUB 导出功能
package exporter

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bmaupin/go-epub"

	"SF/internal/downloader"
	"SF/internal/utils"
)

// EPUBExporter 处理 EPUB 导出
type EPUBExporter struct {
	userAgent string
}

// NewEPUBExporter 创建新的 EPUB 导出器
func NewEPUBExporter(userAgent string) *EPUBExporter {
	return &EPUBExporter{userAgent: userAgent}
}

// Export 导出小说为 EPUB 格式
func (e *EPUBExporter) Export(info *downloader.NovelInfo, dl *downloader.Downloader) error {
	epubBook := epub.NewEpub(info.Title)
	epubBook.SetAuthor(info.Author)

	// 下载并设置封面
	var coverPath string
	if info.Cover != "" {
		fmt.Println("正在下载封面...")
		var err error
		coverPath, err = e.downloadCover(info.Cover)
		if err == nil {
			internalCoverPath, err := epubBook.AddImage(coverPath, "cover.jpg")
			if err == nil {
				epubBook.SetCover(internalCoverPath, "")
				fmt.Println("封面已添加")
			} else {
				fmt.Printf("添加封面到EPUB失败: %v\n", err)
			}
		} else {
			fmt.Printf("封面下载失败: %v\n", err)
		}
	}

	// 用于追踪已下载的图片，避免重复下载
	downloadedImages := make(map[string]string)
	imageCounter := 0
	var tempImageFiles []string

	for _, vol := range info.Volumes {
		// 添加卷标题页
		if vol.Title != "" {
			volTitleHTML := fmt.Sprintf(`<h1 style="text-align:center; margin-top: 40%%;">%s</h1>`, utils.HTMLEscape(vol.Title))
			_, err := epubBook.AddSection(volTitleHTML, vol.Title, "", "")
			if err != nil {
				fmt.Printf("  添加卷标题 %s 失败: %v\n", vol.Title, err)
			}
		}

		// 为每卷创建章节
		for _, chapID := range vol.Chapters {
			chap, err := dl.DownloadChapter(chapID)
			if err != nil {
				fmt.Printf("  %s 下载失败: %v\n", chapID, err)
				continue
			}
			fmt.Printf("  %s 已下载\n", chap.Title)

			// 处理内容
			escapedContent := utils.HTMLEscape(chap.Content)
			processedContent := utils.ConvertImgTags(utils.RestoreEscapedImgTags(escapedContent))

			// 嵌入图片
			processedContent, err = e.embedImages(epubBook, processedContent, downloadedImages, &imageCounter, &tempImageFiles)
			if err != nil {
				fmt.Printf("  嵌入图片失败: %v\n", err)
			}

			htmlContent := fmt.Sprintf(`<h1>%s</h1><p>%s</p>`,
				utils.HTMLEscape(chap.Title),
				strings.ReplaceAll(processedContent, "\n", "</p><p>"))

			_, err = epubBook.AddSection(htmlContent, chap.Title, "", "")
			if err != nil {
				fmt.Printf("  添加章节 %s 失败: %v\n", chap.Title, err)
				continue
			}
		}
	}

	// 清理标题中的非法字符
	titleClean := regexp.MustCompile(`[\\/:*?"<>|]`).ReplaceAllString(info.Title, " ")
	filename := fmt.Sprintf("%s.epub", titleClean)

	if err := epubBook.Write(filename); err != nil {
		return fmt.Errorf("保存EPUB失败: %v", err)
	}

	// 清理临时文件
	if coverPath != "" {
		os.Remove(coverPath)
	}
	for _, tempFile := range tempImageFiles {
		os.Remove(tempFile)
	}

	fmt.Printf("\n已保存为 EPUB: %s\n", filename)
	return nil
}

// embedImages 下载HTML中的外部图片，嵌入EPUB，并替换src为本地路径
func (e *EPUBExporter) embedImages(epubBook *epub.Epub, htmlContent string, downloadedImages map[string]string, counter *int, tempFiles *[]string) (string, error) {
	imgRegex := regexp.MustCompile(`<img[^>]+src=["']([^"']+)["'][^>]*>`)
	matches := imgRegex.FindAllStringSubmatch(htmlContent, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		fullTag := match[0]
		imgURL := match[1]
		if !strings.HasPrefix(imgURL, "http://") && !strings.HasPrefix(imgURL, "https://") {
			continue
		}

		var internalPath string
		if existingPath, ok := downloadedImages[imgURL]; ok {
			internalPath = existingPath
		} else {
			tempFile, err := e.downloadImageToTemp(imgURL)
			if err != nil {
				fmt.Printf("    下载图片失败 %s: %v\n", imgURL, err)
				continue
			}
			*tempFiles = append(*tempFiles, tempFile)

			ext := filepath.Ext(imgURL)
			if ext == "" {
				ext = ".jpg"
			}
			if idx := strings.Index(ext, "?"); idx != -1 {
				ext = ext[:idx]
			}
			if ext == "" {
				ext = ".jpg"
			}

			*counter++
			internalName := fmt.Sprintf("img_%04d%s", *counter, ext)

			internalPath, err = epubBook.AddImage(tempFile, internalName)
			if err != nil {
				fmt.Printf("    添加图片到EPUB失败 %s: %v\n", imgURL, err)
				continue
			}
			downloadedImages[imgURL] = internalPath
			fmt.Printf("    已嵌入图片: %s\n", internalName)
		}

		newTag := regexp.MustCompile(`src=["'][^"']+["']`).ReplaceAllString(fullTag, fmt.Sprintf(`src="%s"`, internalPath))
		htmlContent = strings.Replace(htmlContent, fullTag, newTag, 1)
	}

	return htmlContent, nil
}

// downloadImageToTemp 下载图片到临时文件
func (e *EPUBExporter) downloadImageToTemp(url string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", e.userAgent)
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Referer", "https://book.sfacg.com/")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("状态码: %d", resp.StatusCode)
	}

	file, err := os.CreateTemp("", "img_*")
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(file.Name())
		return "", err
	}

	return file.Name(), nil
}

// downloadCover 下载封面图片
func (e *EPUBExporter) downloadCover(url string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", e.userAgent)
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Referer", "https://book.sfacg.com/")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("下载封面失败，状态码: %d", resp.StatusCode)
	}

	file, err := os.CreateTemp("", "cover.jpg")
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(file.Name())
		return "", err
	}

	return file.Name(), nil
}
