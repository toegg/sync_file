### 介绍

同样，该工具适用于多个项目不同版本的维护，文件更新和新增的同步(自动创建目录)，支持自动提交svn。

### 升级迭代

之前的[文件同步工具](https://blog.csdn.net/toegg/article/details/114394439)，依赖chrome和http包，有时候js加载页面不太稳定，所以有空闲就升级迭代。  
新版本用的跨平台 GUI 工具包[fyne.io/fyne](https://github.com/fyne-io/fyne), 提供了各种控件和布局等，很齐全，基本要用的都能找到，也有提供demo指导和借鉴使用，具体看包文档。

### 同步机制
同步多个目标项目时，以目标项目对应文件的md5为基准，如果都一样，只会随机其中一个目标项目调起工具对比一次，而其它目标项目则以对比后的文件内容覆盖式同步，减少多次调起繁琐操作。
当然也存在版本文件不同，则会调起每个目标项目文件对比，支持同个文件同步到不同版本的差异化。

### 工具链接 
**gitee**: [https://gitee.com/toegg/file_sync  ](https://gitee.com/toegg/file_sync)  
 
**github**: [https://github.com/toegg/sync_file](https://github.com/toegg/sync_file)  

### 展示

ui界面

警告弹框

### 前提
1. 下载安装BeyondCompare对比工具，工具请自行下载；

2. 下载安装svn，并支持控制台命令操作（windows下控制台输入svn不提示错误），不需要提交svn可忽略该点；

*注意：工具只支持window下使用，linux下需要重新打包即可*

### 配置文件
#### 配置文件conf.ini

* 在配置文件中[files]下添加对应的项目目录
```
格式："项目名" = "项目路径"
例子："test1" = "D:\golearn\src\file_sync\etest\test_sync1"
     "test2" = "D:\golearn\src\file_sync\etest\test_sync2"
```

* 在配置文件中[ext_open]下可添加允许同步的后缀名
```
放开可同步的后缀名文件，多个用|隔开
格式："ext_open_list" = ".xxx|.xxx|.xxx"
例子："ext_open_list" = ".go|.txt|.sql"
```

* 在配置文件中[others]下配置BeyondCompare工具绝对路径
```
格式："beyond_path" = "路径"
例子："beyond_path" = "F:\compare\Beyond CompareHA\BeyondCompare\BCompare.exe"
```

### 使用方法
```
启动exe，会显示gui界面，自动加载配置的项目列表
 1.左侧栏上方select下拉框，选择对应的源项目，默认选第一个
 2.左侧栏中间的svn自动提交框，可勾选并填写svn提交备注，同步后会自动提交svn
 3.左侧栏下方input框，输入所需同步的文件名，不需要带目录，文件跟文件之间需换行，如下：
   test.go
   test.txt
   test.sql
 4.勾选右侧栏所需要同步的目标项目
 5.点击开始同步按钮即可
 6.有差异的会打开对比工具对比，对比完成后点关闭则继续往下执行程序
 7.同步结束后右上角关闭即可

注意：勾选提交svn，需要提交时间，会在log端有相关提交输出和错误提示，完成会输出"此次操作已结束"
```
