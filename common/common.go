package common

import "sync"

//文件的信息
type FileInfo struct{
	Name string
	Info string
	Path string								//文件的路径(不包括项目路径)
	CreateDirList map[string][]string		//需要新增的目录
}

//返回json
type ReturnCommon struct {
	Type int
}

//项目信息
type Project struct{
	Name string         	//项目名称
	Path string          	//项目路径
	Keyword string      	//关键字
}

//项目的map
var ProjectMap = make(map[string]Project)

//同步锁
var FsyncLock sync.Mutex
//同步状态
var FsyncStatus bool

//同步文件错误统计
var ErrorDirs []string
//同步文件计数器
var Count = make(chan int)

//svn或编译失败项目
var FailProject []string
//svn或编译计数器
var BuildCount = make(chan int)

