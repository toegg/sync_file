package econfig

import (
	"errors"
	"fmt"
	"gopkg.in/ini.v1"
	"os"
	"strings"
)

const (
	SYNC_SUCCESS     = "同步完成"
	SYNC_FAIL        = "同步失败"
	SYNC_FINISH_WAIT = "同步完成，在提交svn"
	SYNC_ING         = "正在同步中"
	SOURCE_EMPTY     = "源项目路径为空"
	DEST_EMPRY       = "目标项目为空，没勾选"
	NOT_ALLOW_EXT    = "存在不允许同步的后缀文件"
	SYNC_FILE_EMPTY  = "同步文件为空"
)

// 文件的信息
type FileInfo struct {
	Name       string                  //文件名
	Path       string                  //文件的路径(不包括项目路径)
	Byte       string                  //源文件内容
	ByteMd5Map map[string][]SameMd5Dir //目标项目该文件的md5
	//分组形式，文件md5相同的目标项目为一组，只比较一次，其它覆盖式同步
	//map[文件md5]目标项目列表
	CreateDirList map[string][]string //需要新增目录的目标项目Map map[目标项目]新增目录列表
}

// 同组md5的目标目录
type SameMd5Dir struct {
	FilePath string
	Dir      string
}

// 项目信息
type Project struct {
	Name    string //项目名称
	Path    string //项目路径
	Keyword string //关键字
}

// 获取配置的项目列表
func GetIniProject() (map[string]string, []string, error) {
	cfg, err := ini.Load("./conf.ini")
	if err != nil {
		return nil, nil, err
	}
	list := cfg.Section("files").KeysHash()
	keyList := cfg.Section("files").KeyStrings()
	return list, keyList, err
}

// 获取放开的后缀名
func GetAllowExts() ([]string, error) {
	cfg, err := ini.Load("./conf.ini")
	var list []string
	if err != nil {
		return nil, err
	}

	extOpens := cfg.Section("ext_open").Key("ext_open_list").String()
	list = strings.Split(extOpens, "|")

	return list, nil
}

// 获取Beyond Compare工具路径
func GetComparePath() (string, error) {
	cfg, _ := ini.Load("./conf.ini")
	path := cfg.Section("others").Key("beyond_path").String()
	if path == "" {
		fmt.Println("BeyondCompare工具路径缺失")
		return "", errors.New("BeyondCompare工具路径缺失")
	}
	if _, err := os.Stat(path); err != nil {
		fmt.Println("BeyondCompare工具路径缺失")
		return "", err
	}
	return path, nil
}
