package eservice

import (
	"context"
	"errors"
	"file_sync/econfig"
	"file_sync/eutil"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

var Eservice *SyncService

// 工具服务
type SyncService struct {
	projects       map[string]econfig.Project //ini配置所有的项目Map
	projectNameMap map[string]string          //项目名对应的path
	projectList    []string                   //项目名（排序显示用）

	//ui界面
	ui fyne.Window

	//并发锁
	sync_lock   sync.Mutex //同步锁
	sync_status bool       //同步状态

	//单次操作需要重置的内容
	dir           string              //源目录
	syncOns       map[string]int      //选中的同步项目
	file_list     []*econfig.FileInfo //同步的文件
	error_ch      chan string         //同步项目channel
	error_dir     []string            //同步失败项目列表
	svn           bool                //是否提交svn
	svn_commit    string              //svn提交备注
	svn_error_ch  chan string         //svn提交channel
	svn_error_dir []string            //svn提交失败项目列表(同步已经完成，svn失败)
}

// 创建服务
func NewSyncService(list map[string]string, keyList []string) *SyncService {
	service := new(SyncService)
	projects := make(map[string]econfig.Project)
	projectNameMap := make(map[string]string)
	projectList := []string{}
	for _, name := range keyList {
		val := list[name]
		var path, keyword string
		vals := strings.Split(val, "|")
		if len(vals) == 2 {
			path = strings.Trim(vals[0], "\"")
			keyword = vals[1]
		} else {
			path = vals[0]
		}
		projects[path] = econfig.Project{name, path, keyword}
		projectNameMap[name] = path
		projectList = append(projectList, name)
	}
	service.projects = projects
	service.projectNameMap = projectNameMap
	service.projectList = projectList
	service.error_ch = make(chan string)
	service.svn_error_ch = make(chan string)
	service.syncOns = make(map[string]int)
	return service
}

// 重置服务
func (s *SyncService) ResetService() {
	if !s.sync_status {
		s.file_list = s.file_list[0:0]
		s.svn_commit = ""
		s.error_dir = s.error_dir[0:0]
		s.svn_error_dir = s.svn_error_dir[0:0]
	}
}

// 打开Gui界面
func (s *SyncService) NewGui() fyne.Window {
	myApp := app.New()
	myApp.SetIcon(resourceLogoPng)
	myWindow := myApp.NewWindow("文件同步工具")
	s.ui = myWindow
	content := s.NewGuiContent()
	myWindow.SetContent(container.New(layout.NewGridWrapLayout(fyne.NewSize(700, 600)), content))
	return myWindow
}

// gui界面内容
func (s *SyncService) NewGuiContent() fyne.CanvasObject {
	//目标项目数据
	var targetDirNames = []fyne.CanvasObject{widget.NewLabel("目标项目")}
	var targetSyncs = []fyne.CanvasObject{widget.NewLabel("同步选项")}
	var projectNames []string
	for _, name := range s.projectList {
		project := s.projects[s.projectNameMap[name]]
		projectNames = append(projectNames, project.Name)
		targetDirNames = append(targetDirNames, widget.NewLabel(project.Name))
		targetSyncs = append(targetSyncs, widget.NewCheck("", func(on bool) {
			if s.sync_status {
				dialog.ShowError(errors.New(econfig.SYNC_ING), s.ui)
				return
			}
			if on {
				s.syncOns[project.Path] = 1
			} else {
				delete(s.syncOns, project.Path)
			}
		}),
		)
	}

	//源目录下拉框
	sourceComboBox := widget.NewSelect(projectNames, nil)
	if len(projectNames) > 0 {
		sourceComboBox.SetSelected(projectNames[0])
		s.dir = projectNames[0]
	}

	//svn
	svnCheck := func(on bool) {
		if s.sync_status {
			dialog.ShowError(errors.New(econfig.SYNC_ING), s.ui)
			return
		}
		if on {
			s.svn = true
		} else {
			s.svn = false
		}
	}
	//svn备注Text
	svnEntry := widget.NewMultiLineEntry()
	svnEntry.SetPlaceHolder("svn提交备注(XXX)，\n为空或没勾选则不提交")

	//同步文件Text
	fileEntry := widget.NewMultiLineEntry()
	fileEntry.SetPlaceHolder("test.go\n test.sql\n test.txt\n (填入需同步文件，文件间需换行)")

	//同步按钮
	syncButton := widget.Button{
		Text:       "开始同步",
		Importance: widget.HighImportance,
		OnTapped: func() {
			//判断当前是否还在同步中
			if s.sync_status {
				dialog.ShowError(errors.New(econfig.SYNC_ING), s.ui)
				return
			}
			s.sync_lock.Lock()
			//设置同步标识
			s.sync_status = true
			s.sync_lock.Unlock()
			s.dir = s.projectNameMap[sourceComboBox.Selected]
			s.svn_commit = svnEntry.Text
			go s.HandleSync(fileEntry.Text)
		},
	}

	content := container.NewGridWithColumns(2,
		container.NewVBox(
			container.NewGridWrap(fyne.NewSize(250, 50), widget.NewForm(widget.NewFormItem("源项目：", sourceComboBox))),
			widget.NewCheck("自动提交svn", svnCheck),
			container.NewGridWrap(fyne.NewSize(250, 70), svnEntry),
			widget.NewLabel("提交svn和自动编译注意:"),
			widget.NewLabel("【1】需√自动提交svn和填写备注;"),
			widget.NewLabel("【2】需√所需目标项目的同步选项;"),
			widget.NewLabel("提交svn只需【1】即可"),
			container.NewGridWrap(fyne.NewSize(250, 200), fileEntry),
			container.NewGridWrap(fyne.NewSize(100, 30), &syncButton),
		),
		container.NewScroll(
			container.NewGridWithColumns(2,
				container.NewVBox(
					targetDirNames...,
				),
				container.NewVBox(
					targetSyncs...,
				),
			),
		),
	)
	return content
}

// 触发同步处理
func (s *SyncService) HandleSync(fileListArgs string) {
	//----解析参数
	//源目录
	if s.dir == "" {
		dialog.ShowError(errors.New(econfig.SOURCE_EMPTY), s.ui)
		s.sync_status = false
		return
	}

	//目标项目目录
	if len(s.syncOns) <= 0 {
		dialog.ShowError(errors.New(econfig.DEST_EMPRY), s.ui)
		s.sync_status = false
		return
	}

	//同步的文件
	filesList1 := strings.Split(strings.TrimSpace(fileListArgs), "\n")
	filesList := []string{}
	for _, filename := range filesList1 {
		filesList = append(filesList, strings.TrimSpace(filename))
	}
	filesList = eutil.FilterSameInArray[string](filesList)
	err := s.parse_files(filesList)

	if err != nil {
		dialog.ShowError(errors.New(econfig.NOT_ALLOW_EXT), s.ui)
		s.sync_status = false
		return
	}

	if len(s.file_list) <= 0 {
		dialog.ShowError(errors.New(econfig.SYNC_FILE_EMPTY), s.ui)
		s.sync_status = false
		return
	}

	s.start_sync()
}

// 开始同步操作
func (s *SyncService) start_sync() {
	//before操作
	for projectDir, _ := range s.syncOns {
		go s.start_sync_before(projectDir)
	}
	//等待before操作完成
	for i := 0; i < len(s.syncOns); i++ {
		<-s.error_ch
	}

	//执行同步文件操作
	ctx, cancel := context.WithCancel(context.Background())
	go func(waitCtx context.Context) {
	WAIT:
		for {
			select {
			case <-waitCtx.Done():
				break WAIT
			case dir := <-s.error_ch:
				s.error_dir = append(s.error_dir, dir)
			}
		}
	}(ctx)
	CallCompareSync(s.dir, s.file_list, s.error_ch)
	cancel()
	if len(s.error_dir) > 0 {
		dialog.ShowError(errors.New(econfig.SYNC_FAIL), s.ui)
		s.sync_status = false
		return
	}

	//提交svn
	if s.svn && s.svn_commit != "" {
		dialog.ShowError(errors.New(econfig.SYNC_FINISH_WAIT), s.ui)

		for projectDir, _ := range s.syncOns {
			pj, ok := s.projects[projectDir]
			var name string
			if ok {
				name = pj.Name
			} else {
				name = projectDir
			}
			go CommitSvn(name, projectDir, s.svn_commit, s.file_list, s.svn_error_ch)
		}

		//等待提交完成
		go s.wait_svn_finish()

		return
	} else {
		dialog.ShowError(errors.New(econfig.SYNC_SUCCESS), s.ui)
		s.sync_status = false
		return
	}
}

// 开始同步的before操作
// 更新项目svn，筛选需要新创建的目录和文件byte内容等
func (s *SyncService) start_sync_before(projectDir string) {
	//svn更新处理
	UpdateSvn(projectDir)

	var byte_md5 string
	for _, file := range s.file_list {
		fileName := file.Name
		filePath := projectDir + file.Path + "\\" + fileName
		if !eutil.FileExists(filePath) {
			if _, err := os.Stat(projectDir + file.Path); err != nil {
				//筛选新增的多层级目录
				s.filter_svn_need_create_dir(file, projectDir, projectDir+file.Path)
			}
			byte_md5 = eutil.Md5(file.Byte)
		} else {
			bytes, _ := ioutil.ReadFile(filePath)
			byte_md5 = eutil.Md5(string(bytes))
		}
		if same_dirs, ok := file.ByteMd5Map[byte_md5]; ok {
			file.ByteMd5Map[byte_md5] = append(same_dirs, econfig.SameMd5Dir{FilePath: filePath, Dir: projectDir})
		} else {
			file.ByteMd5Map[byte_md5] = []econfig.SameMd5Dir{econfig.SameMd5Dir{FilePath: filePath, Dir: projectDir}}
		}
	}
	s.error_ch <- ""
}

// 解析需要同步的文件
func (s *SyncService) parse_files(filesList []string) error {
	var files []*econfig.FileInfo

	//获取允许的后缀名
	allowExts, err := econfig.GetAllowExts()
	if err != nil {
		return err
	}

	cap := len(s.syncOns)
	for _, file := range filesList {
		if eutil.IsAllowExt(file, allowExts) {
			file := &econfig.FileInfo{Name: file, ByteMd5Map: make(map[string][]econfig.SameMd5Dir, cap), CreateDirList: make(map[string][]string, cap)}
			s.get_file_info(s.dir, s.dir, file)
			if file.Byte != "" {
				files = append(files, file)
			}
		} else {
			return errors.New("not allow ext")
		}
	}
	s.file_list = files
	return nil
}

// 获取同步文件的信息
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
				s.get_file_info(dirs, nowdir+"\\"+fi.Name(), file)
			}
		}
	}
}

// 筛选需要新增的目录(svn提交用)
func (s *SyncService) filter_svn_need_create_dir(file *econfig.FileInfo, dir, path string) {
	paths := strings.Split(path, "\\")

	if len(paths) <= 0 {
		return
	}

	file.CreateDirList[dir] = []string{}
	var basePath string
	for key, val := range paths {
		if key > 0 {
			basePath += "\\" + val
		} else {
			basePath += val
		}

		if !eutil.FileExists(basePath) {
			dirList := file.CreateDirList[dir]
			file.CreateDirList[dir] = append(dirList, basePath)
		}
	}
}

// 等待svn执行完毕
func (s *SyncService) wait_svn_finish() {
	for i := 0; i < len(s.syncOns); i++ {
		result := <-s.svn_error_ch
		if result != "" {
			s.svn_error_dir = append(s.svn_error_dir, result)
		}
	}
	if len(s.svn_error_dir) <= 0 {
		goto OVER
	}

	fmt.Printf("\n\n提交或编译失败的项目有如下：\n")
	for _, dir := range s.svn_error_dir {
		fmt.Printf("【%v】 ", dir)
	}
OVER:
	s.sync_status = false
	s.ResetService()
	fmt.Printf("\n\n本次执行操作结束！！！\n")
}

func (s *SyncService) check_sync_err(err error, dir string) {
	if err != nil {
		fmt.Println("check_sync_error Err:", err)
		s.error_ch <- dir
	}
}
