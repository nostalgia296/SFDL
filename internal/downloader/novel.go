// Package downloader 提供小说目录和章节下载功能
package downloader

import (
	"fmt"
	"strconv"
	"time"

	"SF/internal/api"
	"SF/internal/auth"
	"SF/internal/utils"
)

// Volume 表示小说的一卷
type Volume struct {
	Title    string
	Chapters []string
}

// Chapter 表示单个章节
type Chapter struct {
	Title   string
	Content string
	ID      string
}

// NovelInfo 包含小说的基本信息
type NovelInfo struct {
	Title   string
	Author  string
	Cover   string
	Volumes []Volume
}

// Downloader 处理小说下载
type Downloader struct {
	client *api.Client
	auth   *auth.Manager
}

// NewDownloader 创建新的下载器
func NewDownloader(client *api.Client, authMgr *auth.Manager) *Downloader {
	return &Downloader{
		client: client,
		auth:   authMgr,
	}
}

// GetCatalog 获取小说目录信息
func (d *Downloader) GetCatalog(novelID string) (*NovelInfo, error) {
	info := &NovelInfo{}

	d.auth.RefreshSecurityHeader()

	// 获取标题和作者
	resp, err := d.client.Request("GET", api.BuildURL(fmt.Sprintf("/novels/%s?expand=bigNovelCover", novelID)), nil)
	if err != nil || api.GetHTTPCode(resp) != 200 {
		return nil, fmt.Errorf("标题获取失败")
	}

	if data, ok := resp["data"].(map[string]interface{}); ok {
		info.Title = api.GetString(data, "novelName")
		info.Author = api.GetString(data, "authorName")
		if expand, ok := data["expand"].(map[string]interface{}); ok {
			info.Cover = api.GetString(expand, "bigNovelCover")
		}
	}

	// 获取目录
	d.auth.RefreshSecurityHeader()
	resp, err = d.client.Request("GET", api.BuildURL(fmt.Sprintf("/novels/%s/dirs?expand=originNeedFireMoney", novelID)), nil)
	if err != nil || api.GetHTTPCode(resp) != 200 {
		return nil, fmt.Errorf("目录获取失败")
	}

	if data, ok := resp["data"].(map[string]interface{}); ok {
		if volumeList, ok := data["volumeList"].([]interface{}); ok {
			for _, v := range volumeList {
				if volume, ok := v.(map[string]interface{}); ok {
					vol := Volume{
						Title: api.GetString(volume, "title"),
					}
					if chapterList, ok := volume["chapterList"].([]interface{}); ok {
						for _, c := range chapterList {
							if chapter, ok := c.(map[string]interface{}); ok {
								if chapID, ok := chapter["chapId"].(float64); ok {
									vol.Chapters = append(vol.Chapters, strconv.Itoa(int(chapID)))
								}
							}
						}
					}
					info.Volumes = append(info.Volumes, vol)
				}
			}
		}
	}

	return info, nil
}

// DownloadChapter 下载单个章节
func (d *Downloader) DownloadChapter(chapterID string) (*Chapter, error) {
	var chap Chapter

	for t := 0; t < 5; t++ {
		d.auth.RefreshSecurityHeader()

		resp, err := d.client.Request("GET", api.BuildURL(fmt.Sprintf("/Chaps/%s?expand=content%%2Cexpand.content", chapterID)), nil)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		code := api.GetHTTPCode(resp)
		if code == 200 {
			if data, ok := resp["data"].(map[string]interface{}); ok {
				chap.Title = api.GetString(data, "title")
				chap.ID = chapterID

				var content string
				if c, ok := data["content"].(string); ok {
					content = c
				}
				if expand, ok := data["expand"].(map[string]interface{}); ok {
					if c, ok := expand["content"].(string); ok {
						content += c
					}
				}

				// 字符替换
				chap.Content = utils.ReplaceChars(content)
				return &chap, nil
			}
		} else if code == 403 {
			return nil, fmt.Errorf("未订阅该章节")
		}

		time.Sleep(500 * time.Millisecond)
	}

	return nil, fmt.Errorf("下载失败")
}
