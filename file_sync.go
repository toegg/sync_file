package main

import (
    "errors"
    "fmt"
    "net/http"
    "net/url"

    "time"
    "strings"
    "github.com/zserge/lorca"
    "encoding/json"
    "io/ioutil"
    "os"
    "gopkg.in/ini.v1"
    "file_sync/common"
    "file_sync/esvn"
)

//初始化
func init() {
    cfg,err := ini.Load("./conf.ini")
    if err != nil{
        fmt.Println("error:",err)
        return
    }
    list := cfg.Section("files").KeysHash()
    for name,val := range list{
        var path, keyword string
        vals := strings.Split(val, "|")
        if len(vals) == 2 {
            path = strings.Trim(vals[0],"\"")
            keyword = vals[1]
        }else{
            path = vals[0]
        }
        common.ProjectMap[path] = common.Project{name, path, keyword}
    }
}

func main() {
    fmt.Println("open gui")
    http.HandleFunc("/get_files_list", func(w http.ResponseWriter, r *http.Request){
        get_files_list(w, r)
    })
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        common.FsyncLock.Lock()
        defer common.FsyncLock.Unlock()
        b, isLock := handle_sync(w, r)
        common.ErrorDirs = common.ErrorDirs[0:0]
        if isLock{
           common.FsyncStatus = false
        }
        w.Write([]byte(b))
    })
    s := &http.Server{
        Addr: "localhost:8080",
        ReadTimeout:    55 * time.Second,
        WriteTimeout:   55 * time.Second,
        MaxHeaderBytes: 1 << 20,
    }

    go s.ListenAndServe()

    ui, _ := lorca.New("", "", 650, 400)
    defer ui.Close()

    bytes, _ := ioutil.ReadFile("./file_sync.html")
    html := string(bytes)

    ui.Load("data:text/html," + url.PathEscape(html))

    <-ui.Done()
}
//请求获取文件名称和目录
func get_files_list(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    var list = make(map[string]string)
    for _, pj := range common.ProjectMap{
       list[pj.Name] = pj.Path
    }
    b, err := json.Marshal(list)
    if err != nil{
      w.Write([]byte(""))
      return
    }
    w.Write([]byte(b))
}

//请求同步处理
func handle_sync(w http.ResponseWriter, r *http.Request) (b []byte, isLock bool) {
    w.Header().Set("Access-Control-Allow-Origin", "*")

    //判断同步锁状态
    if common.FsyncStatus{
        b, _ := json.Marshal(common.ReturnCommon{Type: 8})
        return b, false
    }
    common.FsyncStatus = true

    r.ParseForm()
    dir := get_string(r.Form["dir"])
    filesList := strings.Split(get_string(r.Form["files"]), "\n")
    projectsArgs := get_string(r.Form["projects"])
    projectsList := strings.Split(projectsArgs, ",")
    svnselect := get_string(r.Form["svnselect"])
    svncommit := ""
    if svnselect == "true" {
        svncommit = get_string(r.Form["svncommit"])
    }
    autobuild := ""
    if svnselect == "true" && svncommit != ""{
        autobuild = get_string(r.Form["autobuild"])
    }

    //分离hrl和erl
    erlFiles, err := filter_files(dir, filesList, len(projectsList))

    if dir == "" {
        b, _ = json.Marshal(common.ReturnCommon{Type: 4})
        return b, true
    }else if len(erlFiles) <= 0{
        b, _ = json.Marshal(common.ReturnCommon{Type: 3})
        return b, true
    }else if projectsArgs == ""{
        b, _ = json.Marshal(common.ReturnCommon{Type: 5})
        return b, true
    }else if err != nil {
        b, _ = json.Marshal(common.ReturnCommon{Type: 6})
        return b, true
    }else{
        //开始同步
        for _, projectDir := range projectsList {
            go project_sync(projectDir, erlFiles, svncommit, autobuild)
        }

        //等待同步完成
        for i:=0; i< len(projectsList); i++{
            <- common.Count
        }

        if len(common.ErrorDirs) > 0 {
            b, _ = json.Marshal(common.ReturnCommon{Type: 2})
            return b, true
            return
        }

        if svncommit != ""{
            b, _ = json.Marshal(common.ReturnCommon{Type: 7})
            //等待提交完成
            go wait_svn_finish(len(projectsList))
            return b, false
        }else{
            b, _ = json.Marshal(common.ReturnCommon{Type: 1})
            return b, true
         }
    }
}

//等待svn执行完毕
func wait_svn_finish(pjLen int) {
    for i:=0; i< pjLen; i++{
        <- common.BuildCount
    }
    failLen := len(common.FailProject)
    if failLen <= 0 {
        goto OVER
    }

    fmt.Printf("\n\n提交或编译失败的项目有如下：\n")
    for _, val := range common.FailProject{
        fmt.Printf("【%v】 ",val)
    }
OVER:
    fmt.Printf("\n\n本次执行操作结束！！！\n")
    common.FailProject = common.FailProject[0:0]
    common.FsyncStatus = false
}

//筛选文件
func filter_files(dir string, filesList []string, Cap int) ([]*common.FileInfo, error) {

    var erlFiles []*common.FileInfo
    var errs error

    //获取放开的后缀名
    cfg, err := ini.Load("./conf.ini")
    var list []string
    if err == nil{
        extOpens := cfg.Section("files").Key("ext_open_list").String()
        list = strings.Split(extOpens, "|")
    }

    for _, file := range filesList{
        isAllowExt := is_in_ext_open(file, list)
        if isAllowExt {
            files := &common.FileInfo{Name:file, CreateDirList:make(map[string][]string, Cap)}
            get_file_bytes(dir, dir, files)
            if files.Info != "" {
                erlFiles = append(erlFiles, files)
            }
        }else{
            errs = errors.New("not allow ext")
        }
    }
    return erlFiles, errs
}

//开始同步
func project_sync(projectDir string, erlFiles []*common.FileInfo, svncommit, build string) {
    //svn更新处理
    esvn.UpdateSvn(projectDir)

    for _, file := range erlFiles{
        write_file_bytes(projectDir, file)
    }

    //svn提交处理
    go esvn.CommitSvn(svncommit, build, projectDir, erlFiles)

    common.Count <- 1
}

//写入目标项目文件
func write_file_bytes(dir string, file *common.FileInfo) {
    fileName := file.Name
    fileBytes := file.Info
    filePath := dir + file.Path + "\\" + fileName

    if fileExists(filePath) { //直接写入
        f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
        defer f.Close()
        check_err(err, dir)
        _, err1 := f.Write([]byte(fileBytes))
        check_err(err1, dir)
    }else{ //创建写入
        if _, err := os.Stat(dir + file.Path); err != nil {
            //筛选新增的多层级目录
            filter_create_dir(file, dir, dir + file.Path)
            err = os.MkdirAll(dir + file.Path, 0711)
            check_err(err, dir)
        }
        f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
        defer f.Close()
        check_err(err, dir)
        _, err1 := f.Write([]byte(file.Info))
        check_err(err1, dir)
    }
    return
}

//筛选需要创建的目录(svn提交用)
func filter_create_dir(file *common.FileInfo, dir, path string){
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

        if !fileExists(basePath) {
            dirList := file.CreateDirList[dir]
            file.CreateDirList[dir] = append(dirList, basePath)
        }
    }
}

//获取源项目文件bytes
func get_file_bytes(dirs string, nowdir string, files *common.FileInfo) {
    rd, _ := ioutil.ReadDir(nowdir)
    for _, fi := range rd {
        if fi.Name() == files.Name {
            bytes, _ := ioutil.ReadFile(nowdir + "\\" + fi.Name())
            str := string(bytes)  
            files.Info = str
            files.Path = strings.TrimPrefix(nowdir, dirs)
            return       
        } else {
            if fi.IsDir() {
                get_file_bytes(dirs, nowdir + "\\" + fi.Name(), files)
            }
        }
    }
}

//检测错误
func check_err(err error, dir string){
    if err != nil{
        fmt.Println(err)
        common.ErrorDirs = append(common.ErrorDirs, dir)
        common.Count <- 1
        return 
    }
}

//判断文件是否为允许同步的后缀名
func is_in_ext_open(file string, extOpens []string) bool{
    for _, v := range extOpens {
        if strings.Contains(file, v){
            return true
        }
    }
    return false
}

//转为string
func get_string(v interface{}) string {
    switch result := v.(type) {
    case string:
        return result
    case []string:
        return strings.Join(result, "")
    case []byte:
        return string(result)
    default:
        if v != nil {
            return fmt.Sprint(result)
        }
    }
    return ""
}

//文件是否存在
func fileExists(path string) (bool) {
    _, err := os.Stat(path)
    if err == nil {
        return true
    }
    if os.IsNotExist(err) {
        return false
    }
    return false
}
