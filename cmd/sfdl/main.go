// sfdl - SF轻小说下载器
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"SF/internal/api"
	"SF/internal/auth"
	"SF/internal/downloader"
	"SF/internal/exporter"
)

const cookieFile = "cookie.txt"

func main() {
	// 初始化 API 客户端
	client := api.NewClient()

	// 初始化认证管理器
	authMgr := auth.NewManager(client)
	fmt.Println("正在初始化...")
	if err := authMgr.InitNonce(); err != nil {
		fmt.Printf("初始化失败: %v\n", err)
		return
	}
	fmt.Println("初始化完成")

	// 加载或创建 cookie
	loadCookie(client)

	// 登录检查
	for authMgr.CheckLogin() {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("输入手机号: ")
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)

		fmt.Print("输入密码: ")
		password, _ := reader.ReadString('\n')
		password = strings.TrimSpace(password)

		cookie, err := authMgr.Login(username, password)
		if err != nil {
			fmt.Println("登录失败:", err)
			continue
		}

		client.SetHeader("cookie", cookie)
		os.WriteFile(cookieFile, []byte(cookie), 0644)
		fmt.Println("登录成功")
	}

	// 输入小说ID
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("输入小说ID: ")
	novelID, _ := reader.ReadString('\n')
	novelID = strings.TrimSpace(novelID)

	// 更新 User-Agent
	client.SetHeader("user-agent", fmt.Sprintf("boluobao/5.2.16(android;35)/OPPO/%s/OPPO", strings.ToLower(auth.GetDeviceToken())))

	// 获取小说信息
	dl := downloader.NewDownloader(client, authMgr)
	info, err := dl.GetCatalog(novelID)
	if err != nil {
		fmt.Printf("获取小说信息失败: %v\n", err)
		return
	}

	fmt.Printf("标题: %s\n作者: %s\n", info.Title, info.Author)

	// 显示卷信息
	for i, vol := range info.Volumes {
		fmt.Printf("%d: %s (%d章)\n", i+1, vol.Title, len(vol.Chapters))
	}

	// 选择下载格式
	fmt.Println("\n请选择下载格式:")
	fmt.Println("1: TXT")
	fmt.Println("2: EPUB")
	fmt.Print("输入选项 (1/2): ")
	format, _ := reader.ReadString('\n')
	format = strings.TrimSpace(format)

	fmt.Println("\n开始下载全部卷...")

	switch format {
	case "2":
		// EPUB格式
		ePubExporter := exporter.NewEPUBExporter(client.GetHeader("user-agent"))
		if err := ePubExporter.Export(info, dl); err != nil {
			fmt.Printf("生成EPUB失败: %v\n", err)
		}
	default:
		// TXT格式
		txtExporter := exporter.NewTXTExporter()
		if err := txtExporter.Export(info, dl); err != nil {
			fmt.Printf("保存TXT失败: %v\n", err)
		}
	}
}

// loadCookie 从文件加载 cookie
func loadCookie(client *api.Client) {
	if _, err := os.Stat(cookieFile); os.IsNotExist(err) {
		client.SetHeader("cookie", "")
		os.WriteFile(cookieFile, []byte(""), 0644)
		fmt.Println("cookie.txt 已创建")
	} else {
		data, _ := os.ReadFile(cookieFile)
		client.SetHeader("cookie", string(data))
		fmt.Println("读取到 cookie:", string(data))
	}
}
