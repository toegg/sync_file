package main

import (
    "file_sync/econfig"
    "file_sync/eservice"
	"github.com/flopp/go-findfont"
    "log"
	"os"
	"strings"
)

func init() {
	//设置中文字体:解决中文乱码问题
	fontPaths := findfont.List()
	for _, path := range fontPaths {
		if strings.Contains(path, "msyh.ttf") || strings.Contains(path, "simhei.ttf") || strings.Contains(path, "simsun.ttc") || strings.Contains(path, "simkai.ttf") {
			os.Setenv("FYNE_FONT", path)
			break
		}
	}
}

func main() {
	log.Println("打开界面中...")
	//读取配置
	list, keyList, err := econfig.GetIniProject()
	if err != nil {
		log.Fatalf("Init Ini Cfg Err:%v", err)
	}
	//创建服务
	eservice.Eservice = eservice.NewSyncService(list, keyList)

	//打开gui
	myWindow := eservice.Eservice.NewGui()
	myWindow.ShowAndRun()
}

