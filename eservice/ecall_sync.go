package eservice

import (
	"eggpackage/log"
	"file_sync/econfig"
	"file_sync/eutil"
	"io/ioutil"

	"os"
	"os/exec"
)

//调用Beyond Compare工具处理同步
func CallCompareSync(dir string, files []*econfig.FileInfo, dir_list []string, ch chan string){
	log.Elog.Info("\n正在启动Beyond Compare工具对比。。。")

	//工具路径
	compareDir, err := econfig.GetComparePath()
	if err != nil{
		ch <- dir
		return
	}

	for _, file := range files{
		sourceFiles := dir + file.Path + "\\" + file.Name
		source_md5 := eutil.Md5(file.Byte)
		for byte_md5, same_dirs := range file.ByteMd5Map{
			//md5相同，直接覆盖式同步
			if source_md5 == byte_md5 {
				for _, destFiles := range same_dirs{
					eutil.WriteToFile(destFiles.FilePath, destFiles.Dir + file.Path, []byte(file.Byte))
				}
				continue
			}

			//发起调用
			args := []string{sourceFiles, same_dirs[0].FilePath}
			cmd := exec.Command(compareDir, args...)
			pwd, _ :=os.Getwd()
			cmd.Dir = pwd
			err := cmd.Run()
			//这里排除13状态码
			if err != nil && (err.Error() == "exit status 100" || err.Error() == "exit status 103" || err.Error() == "exit status 104" ||
				err.Error() == "exit status 105" || err.Error() == "exit status 106" || err.Error() == "exit status 107") {
				ch <- dir
			}else{
				//同步到同组的其它目标目录文件
				for _, destFiles := range same_dirs{
					bytes, _ := ioutil.ReadFile(same_dirs[0].FilePath)
					eutil.WriteToFile(destFiles.FilePath, destFiles.Dir + file.Path, bytes)
				}
			}
		}
	}
}

