package main

import (
    "errors"
    "fmt"
    "net/http"
    "net/url"
    "os/exec"

    "time"
    "strings"
    "github.com/zserge/lorca"
    "encoding/json"
    "io/ioutil"
    "os"
    "gopkg.in/ini.v1"
)

//文件的信息
type fileInfo struct{
    name string
    info string
    path string
    is_write int
    is_create int
}

//返回json
type ReturnCommon struct {
    Type int
}

//项目
type project struct{
    name string         //项目名称
    path string          //项目路径
}

//项目的map
var projectMap = make(map[string]project)

var errorDirs []string

var count = make(chan int)

//初始化
func init() {
    cfg,err := ini.Load("./conf.ini")
    if err != nil{
        fmt.Println("error:",err)
        return
    }
    list := cfg.Section("files").KeysHash()
    for name, path := range list{
        projectMap[path] = project{name, path}
    }
}

func main() {
    fmt.Println("open gui")
    http.HandleFunc("/get_files_list", func(w http.ResponseWriter, r *http.Request){
        get_files_list(w, r)
    })
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        handle_sync(w, r)
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
    header(w)
    var list = make(map[string]string)
    for _, pj := range projectMap{
        list[pj.name] = pj.path
    }
    b, err := json.Marshal(list)
    if err != nil{
       w.Write([]byte(""))
       return
    }
    w.Write([]byte(b))
}


//请求同步处理
func handle_sync(w http.ResponseWriter, r *http.Request) {
    header(w)
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

    //筛选文件列表
    fileInfoList, err := filter_files(dir, filesList)

    var b []byte
    if dir == "" {
        b, _ = json.Marshal(ReturnCommon{Type: 4})
    }else if len(fileInfoList) <= 0{
        b, _ = json.Marshal(ReturnCommon{Type: 3}) 
    }else if projectsArgs == ""{
        b, _ = json.Marshal(ReturnCommon{Type: 5}) 
    }else if err != nil {
        b, _ = json.Marshal(ReturnCommon{Type: 6})
    }else{
        //开始同步
        for _, projectDir := range projectsList {
            go project_sync(count, projectDir, fileInfoList, svncommit)
        }

        for i:=0; i< len(projectsList); i++{
            <- count
        }

        if len(errorDirs) > 0 {
            b, _ = json.Marshal(ReturnCommon{Type: 2})
        }else{
            b, _ = json.Marshal(ReturnCommon{Type: 1})
        }
    }
    //返回结果
    w.Write([]byte(b))

}

//筛选文件信息
func filter_files(dir string, filesList []string) ([]*fileInfo, error) {
    var filesInfoList []*fileInfo
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
            files := &fileInfo{name:file}
            get_file_bytes(dir, dir, files)
            if files.info != "" {
                filesInfoList = append(filesInfoList, files)
            }
        }else{
            errs = errors.New("not allow ext")
        }
    }

    return filesInfoList, errs
}

//同步
func project_sync(count chan int, projectDir string, fileInfoList []*fileInfo, svncommit string) {
    for _, file := range fileInfoList{
        write_file_bytes(projectDir, file)
        //写入失败了,尝试重新打开或创建
        if file.is_write != 1 {
            if _, err := os.Stat(projectDir + file.path); err != nil {
                err = os.MkdirAll(projectDir + file.path, 0711)
                check_err(err, projectDir)
            }
            f, err := os.OpenFile(projectDir + file.path + "\\" + file.name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
            check_err(err, projectDir)
            _, err1 := f.Write([]byte(file.info))
            check_err(err1, projectDir)
            file.is_create = 1
            f.Close()
        }
    }

    //svn提交处理
    commit_svn(svncommit, projectDir, fileInfoList)

    count <- 1
}

//svn提交处理
func commit_svn(svncommit string, projectDir string, allFiles []*fileInfo) {
    if svncommit != ""{
        var files []string
        var addFiles []string
        for _, file := range allFiles{
            files = append(files, projectDir + "\\" + file.path + "\\" + file.name)
            if file.is_create == 1 {
                addFiles = append(addFiles, projectDir + "\\" + file.path + "\\" + file.name)
            }
        }
        fmt.Printf("--------%v:start commit svn--------\n", projectDir)
        //新增的文件执行add
        if len(addFiles) > 0 {
        args1 := []string{"add"}
        args1 = append(args1, addFiles...)
        cmd1 := exec.Command("svn", args1...)
        cmd1.Dir = projectDir
        cmd1.Run()
        }

        //提交svn
        args := []string{"commit", "-m", svncommit}
        args = append(args, files...)
        cmd := exec.Command("svn", args...)
        cmd.Dir = projectDir
        out, err := cmd.Output()
        if err != nil {
            fmt.Println(err)
        }
        fmt.Println(string(out))
        fmt.Printf("--------%v:over commit svn--------\n", projectDir)
    }
}

//找文件并覆盖写入
func write_file_bytes(dir string, file *fileInfo) {
    rd, _ := ioutil.ReadDir(dir)
    fileName := file.name
    fileBytes := file.info
    for _, fi := range rd {
        if fi.Name() == fileName {
            f, err := os.OpenFile(dir + "\\" + fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
            check_err(err, dir)
            _, err1 := f.Write([]byte(fileBytes))  
            check_err(err1, dir)
            //标志文件写入完毕
            file.is_write = 1
            f.Close()
            return           
        } else {
            write_file_bytes(dir + "\\" + fi.Name(), file)
        }
    }  
}

//找文件读取bytes
func get_file_bytes(dirs string, nowdir string, files *fileInfo) {
    rd, _ := ioutil.ReadDir(nowdir)
    for _, fi := range rd {
        if fi.Name() == files.name {
            bytes, _ := ioutil.ReadFile(nowdir + "\\" + fi.Name())
            str := string(bytes)  
            files.info = str
            files.path = "\\" + strings.TrimLeft(nowdir, dirs)
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
        errorDirs = append(errorDirs, dir)
        count <- 1
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

func header(w http.ResponseWriter) {
    w.Header().Set("Access-Control-Allow-Origin", "*") //允许访问所有域
}