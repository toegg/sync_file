package main

import (
    "file_sync/econfig"
    "file_sync/eservice"
    "fmt"
    "github.com/zserge/lorca"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "time"
)

func main() {
    //读取配置
    list, err :=econfig.GetIniProject()
    if err != nil{
        log.Fatalf("Init Ini Cfg Err:%v", err)
    }

    //标准输入获取同步类型
    var sync_type string
    fmt.Println("覆盖式同步输入1，调起BeyondCompare同步输入2")
    fmt.Scanln(&sync_type)

    //创建服务
    service := eservice.NewSyncService(list, sync_type)

    fmt.Println("open gui")
    http.HandleFunc("/get_files_list", func(w http.ResponseWriter, r *http.Request){
        service.GetFilesList(w, r)
    })
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        service.HandleSync(w, r)
        service.ResetService()
    })
    s := &http.Server{
        Addr: "localhost:8080",
        ReadTimeout:    55 * time.Second,
        WriteTimeout:   55 * time.Second,
        MaxHeaderBytes: 1 << 20,
    }
    go s.ListenAndServe()

    //启动界面
    ui, _ := lorca.New("", "", 650, 400)
    defer ui.Close()
    bytes, _ := ioutil.ReadFile("./file_sync.html")
    html := string(bytes)
    ui.Load("data:text/html," + url.PathEscape(html))

    <-ui.Done()
}

