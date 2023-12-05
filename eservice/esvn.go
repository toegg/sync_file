package eservice

//------------------------svn提交处理

import (
	"file_sync/econfig"
	"fmt"
	"os/exec"
)

// svn更新处理
func UpdateSvn(projectDir string) {
	args := []string{"update", projectDir}
	cmd := exec.Command("svn", args...)
	cmd.Dir = projectDir
	err := cmd.Run()
	if err != nil {
		fmt.Printf("\n--------【%v】:update svn ERROR--------%v\n", projectDir, err)
	}
}

// svn提交处理
func CommitSvn(name, projectDir, svncommit string, syncFiles []*econfig.FileInfo, svnCh chan string) {
	var files []string
	var addFiles []string
	for _, file := range syncFiles {
		//新增目录
		if dirList, ok := file.CreateDirList[projectDir]; ok {
			for _, val := range dirList {
				addFiles = append(addFiles, val)
				files = append(files, val)
			}
		}
		addFiles = append(addFiles, projectDir+file.Path+"\\"+file.Name)
		files = append(files, projectDir+file.Path+"\\"+file.Name)
	}

	//新增的文件执行add
	if len(addFiles) > 0 {
		args1 := []string{"add"}
		args1 = append(args1, addFiles...)
		cmd1 := exec.Command("svn", args1...)
		cmd1.Dir = projectDir
		fmt.Printf("\n--------【%v】:start commit svn--------\n\n新增命令：\nsvn %v", name, args1)
		cmd1.Run()
	}

	//文件执行commit
	args := []string{"commit", "-m", svncommit}
	args = append(args, files...)
	cmd := exec.Command("svn", args...)
	cmd.Dir = projectDir
	if len(addFiles) > 0 {
		fmt.Printf("\n提交命令：\nsvn %v\n\n正在执行中。。。。\n", args)
	} else {
		fmt.Printf("\n--------【%v】:start commit svn--------\n\n提交命令：\nsvn %v\n\n正在执行中。。。。\n", name, args)
	}
	out, err := cmd.Output()

	//输出结果
	if err != nil {
		fmt.Printf("\n【%v】执行结果：\ncommit error:%v\n检查备注是否有按规定加前缀如：【功能】XXX\n", name, err)
	} else {
		fmt.Printf("\n【%v】执行结果：\ncommit success result:\n%v\n", name, string(out))
	}
	fmt.Printf("\n--------【%v】:over commit svn--------\n", name)

	if err != nil {
		svnCh <- name
		return
	}

	svnCh <- ""

}
