package eservice

import (
	"context"
	"encoding/json"
	"errors"
	"file_sync/econfig"
	"file_sync/eutil"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
)

var service *SyncService

//工具服务
type SyncService struct {
	projects map[string]econfig.Project     //ini配置所有的项目Map
	is_beyond_compare bool					//是否使用beyond compare对比工具

	//并发锁
	sync_lock sync.Mutex                    //同步锁
	sync_status bool                        //同步状态

	//单次操作需要重置的内容
	dir string 								//源目录
	dir_list []string						//同步的目标目录
	file_list []*econfig.FileInfo			//同步的文件
	error_ch chan string  					//同步项目channel
	error_dir []string                      //同步失败项目列表
	svn_commit string						//svn提交备注
	svn_error_ch chan string				//svn提交channel
	svn_error_dir []string                  //svn提交失败项目列表(同步已经完成，svn失败)
}

//创建服务
func NewSyncService(list  map[string]string, sync_type string) *SyncService{
	service := new(SyncService)
	projects := make(map[string]econfig.Project)
	for name,val := range list{
		var path, keyword string
		vals := strings.Split(val, "|")
		if len(vals) == 2 {
			path = strings.Trim(vals[0],"\"")
			keyword = vals[1]
		}else{
			path = vals[0]
		}
		projects[path] = econfig.Project{name, path, keyword}
	}
	if sync_type == "2" {
		service.is_beyond_compare = true
	}
	service.projects = projects
	service.error_ch = make(chan string)
	service.svn_error_ch = make(chan string)
	return service
}

//重置服务
func (s *SyncService) ResetService() {
	if !s.sync_status{
		s.dir_list = s.dir_list[0:0]
		s.file_list = s.file_list[0:0]
		s.svn_commit = ""
		s.error_dir = s.error_dir[0:0]
		s.svn_error_dir = s.svn_error_dir[0:0]
	}
}

//请求获取配置的文件名称和目录
func (s *SyncService) GetFilesList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	var list = make(map[string]string)
	for _, pj := range s.projects{
		list[pj.Name] = pj.Path
	}
	b, err := json.Marshal(list)
	if err != nil{
		w.Write([]byte(""))
		return
	}
	w.Write([]byte(b))
}

//触发同步处理
func (s *SyncService) HandleSync(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	s.sync_lock.Lock()
	defer s.sync_lock.Unlock()

	//判断当前是否还在同步中
	if s.sync_status{
		s.returnAjax(w, econfig.SYNC_ING)
		return
	}
	s.sync_status = true

	//----解析参数
	r.ParseForm()
	//源目录
	dir := eutil.GetString(r.Form["dir"])
	if dir == ""{
		s.returnAjax(w,econfig.SOURCE_EMPTY)
		s.sync_status = false
		return
	}

	//目标项目目录
	projectsArgs := eutil.GetString(r.Form["projects"])
	if projectsArgs == "" {
		s.returnAjax(w,econfig.DEST_EMPRY)
		s.sync_status = false
		return
	}
	projectsList := strings.Split(projectsArgs, ",")

	//同步的文件
	filesList := strings.Split(eutil.GetString(r.Form["files"]), "\n")
	err := s.parse_files(dir, filesList, len(projectsList))
	if err != nil{
		s.returnAjax(w,econfig.NOT_ALLOW_EXT)
		s.sync_status = false
		return
	}
	if len(s.file_list) <= 0 {
		s.returnAjax(w,econfig.SYNC_FILE_EMPTY)
		s.sync_status = false
		return
	}

	//是否提交svn和svn提交备注
	svnselect := eutil.GetString(r.Form["svnselect"])
	var svncommit string
	if svnselect == "true" {
		svncommit = eutil.GetString(r.Form["svncommit"])
	}

	s.dir = dir
	s.dir_list = projectsList
	s.svn_commit = svncommit
	s.start_sync(w, r)
}

//开始同步操作
func (s *SyncService) start_sync(w http.ResponseWriter, r *http.Request) {
	//before操作
	for _, projectDir := range s.dir_list {
		go s.start_sync_before(projectDir)
	}
	//等待before操作完成
	for i:=0; i< len(s.dir_list); i++{
		<- s.error_ch
	}

	//执行同步文件操作
	ctx, cancel := context.WithCancel(context.Background())
	go func(waitCtx context.Context){
WAIT:
		for{
			select{
			case <- waitCtx.Done():
				break WAIT
			case dir := <- s.error_ch:
				s.error_dir = append(s.error_dir, dir)
			}
		}
	}(ctx)
	if s.is_beyond_compare {
		CallCompareSync(s.dir, s.file_list, s.dir_list, s.error_ch)
	}else{
		s.call_sync()
	}
	cancel()
	if len(s.error_dir) > 0 {
		s.returnAjax(w, econfig.SYNC_FAIL)
		s.sync_status = false
		return
	}

	//提交svn
	if s.svn_commit != ""{
		s.returnAjax(w, econfig.SYNC_FINISH_WAIT)

		for _, projectDir := range s.dir_list {
			pj, ok:= s.projects[projectDir]
			var name string
			if ok {
				name = pj.Name
			}else{
				name = projectDir
			}
			go CommitSvn(name, projectDir, s.svn_commit, s.file_list, s.svn_error_ch)
		}

		//等待提交完成
		go s.wait_svn_finish()

		return
	}else{
		s.returnAjax(w, econfig.SYNC_SUCCESS)
		s.sync_status = false
		return
	}
}

//开始同步的before操作
//更新项目svn，筛选需要新创建的目录和文件byte内容等
func (s *SyncService) start_sync_before(projectDir string) {
	//svn更新处理
	UpdateSvn(projectDir)

	var byte_md5 string
	for _, file := range s.file_list{
		fileName := file.Name
		filePath := projectDir + file.Path + "\\" + fileName
		if !eutil.FileExists(filePath) {
			if _, err := os.Stat(projectDir + file.Path); err != nil {
				//筛选新增的多层级目录
				s.filter_svn_need_create_dir(file, projectDir, projectDir + file.Path)
			}
			byte_md5 = eutil.Md5(file.Byte)
		}else{
			bytes, _ := ioutil.ReadFile(filePath)
			byte_md5 = eutil.Md5(string(bytes))
		}
		if same_dirs, ok :=file.ByteMd5Map[byte_md5]; ok{
			file.ByteMd5Map[byte_md5] = append(same_dirs, econfig.SameMd5Dir{FilePath: filePath, Dir:projectDir})
		}else{
			file.ByteMd5Map[byte_md5] = []econfig.SameMd5Dir{econfig.SameMd5Dir{FilePath: filePath, Dir:projectDir}}
		}
	}
	s.error_ch <- ""
}

//直接覆盖式同步
func (s *SyncService) call_sync(){
	for _, dir := range s.dir_list {
		for _, file := range s.file_list {
			fileName := file.Name
			filePath := dir + file.Path + "\\" + fileName
			err := eutil.WriteToFile(filePath, dir + file.Path, []byte(file.Byte))
			s.check_sync_err(err, dir)
		}
	}
}

//解析需要同步的文件
func (s *SyncService) parse_files(dir string, filesList []string, Cap int) error {
	var files []*econfig.FileInfo

	//获取允许的后缀名
	allowExts, err := econfig.GetAllowExts()
	if err != nil{
		return err
	}

	for _, file := range filesList{
		if eutil.IsAllowExt(file, allowExts) {
			file := &econfig.FileInfo{Name:file, ByteMd5Map: make(map[string][]econfig.SameMd5Dir, Cap), CreateDirList:make(map[string][]string, Cap)}
			s.get_file_info(dir, dir, file)
			if file.Byte != "" {
				files = append(files, file)
			}
		}else{
			return errors.New("not allow ext")
		}
	}
	s.file_list = files
	return nil
}

//获取同步文件的信息
func (s *SyncService) get_file_info(dirs string, nowdir string, file *econfig.FileInfo) {
	rd, _ := ioutil.ReadDir(nowdir)
	for _, fi := range rd {
		if fi.Name() == file.Name {
			bytes, _ := ioutil.ReadFile(nowdir + "\\" + fi.Name())
			file.Byte = string(bytes)
			file.Path = strings.TrimPrefix(nowdir, dirs)
			return
		} else {
			if fi.IsDir() {
				s.get_file_info(dirs, nowdir + "\\" + fi.Name(), file)
			}
		}
	}
}

//筛选需要新增的目录(svn提交用)
func (s *SyncService) filter_svn_need_create_dir(file *econfig.FileInfo, dir, path string){
	paths := strings.Split(path, "\\")

	if len(paths) <= 0{
		return
	}

	file.CreateDirList[dir] = []string{}
	var basePath string
	for key, val := range paths {
		if key > 0 {
			basePath += "\\" + val
		}else{
			basePath += val
		}

		if !eutil.FileExists(basePath) {
			dirList := file.CreateDirList[dir]
			file.CreateDirList[dir] = append(dirList, basePath)
		}
	}
}

//等待svn执行完毕
func (s *SyncService) wait_svn_finish() {
	for i:=0; i< len(s.dir_list); i++{
		result:= <- s.svn_error_ch
		if result != ""{
			s.svn_error_dir = append(s.svn_error_dir, result)
		}
	}
	if len(s.svn_error_dir) <= 0 {
		goto OVER
	}

	fmt.Printf("\n\n提交或编译失败的项目有如下：\n")
	for _, dir := range s.svn_error_dir{
		fmt.Printf("【%v】 ",dir)
	}
OVER:
	s.sync_status = false
	s.ResetService()
	fmt.Printf("\n\n本次执行操作结束！！！\n")
}

//返回ajax
func (s *SyncService) returnAjax(w http.ResponseWriter, return_type int) {
	b, _ := json.Marshal(econfig.ReturnCommon{Type: return_type})
	w.Write([]byte(b))
}

func (s *SyncService) check_sync_err(err error, dir string){
	if err != nil{
		fmt.Println("check_sync_error Err:", err)
		s.error_ch <- dir
	}
}




