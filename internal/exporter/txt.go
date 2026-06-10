// Package exporter 提供 TXT 导出功能
package exporter

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"SF/internal/downloader"
)

// TXTExporter 处理 TXT 导出
type TXTExporter struct{}

// NewTXTExporter 创建新的 TXT 导出器
func NewTXTExporter() *TXTExporter {
	return &TXTExporter{}
}

// Export 导出小说为 TXT 格式
func (e *TXTExporter) Export(info *downloader.NovelInfo, dl *downloader.Downloader) error {
	var content strings.Builder
	content.WriteString(info.Title + "\n\n")
	content.WriteString("作者: " + info.Author + "\n\n")

	for i, vol := range info.Volumes {
		fmt.Printf("\n正在下载: %s\n", vol.Title)
		content.WriteString(vol.Title + "\n\n")

		for _, chapID := range vol.Chapters {
			chap, err := dl.DownloadChapter(chapID)
			if err != nil {
				fmt.Printf("  %s 下载失败: %v\n", chapID, err)
				continue
			}
			fmt.Printf("  %s 已下载\n", chap.Title)
			content.WriteString(chap.Title + "\n")
			content.WriteString(chap.Content + "\n\n")
		}

		if i < len(info.Volumes)-1 {
			content.WriteString("\n")
		}
	}

	// 清理标题中的非法字符
	titleClean := regexp.MustCompile(`[\\/:*?"<>|]`).ReplaceAllString(info.Title, " ")
	filename := fmt.Sprintf("%s.txt", titleClean)

	if err := os.WriteFile(filename, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("保存失败: %v", err)
	}

	fmt.Printf("\n已保存为 TXT: %s\n", filename)
	return nil
}
