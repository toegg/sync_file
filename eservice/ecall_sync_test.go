package eservice

import (
	"context"
	"file_sync/econfig"
	"file_sync/eutil"
	"os"

	"io/ioutil"
	"testing"
)

//测试调起Compare Beyond
func TestCallCompareSync(t *testing.T) {
	ch := make(chan string, 3)
	var ch_dir []string
	//go文件
	goBytes, _ := ioutil.ReadFile("../etest/test_sync1/test.go")
	goByteMd5Map := make(map[string][]econfig.SameMd5Dir)
	destGoBytes, _ := ioutil.ReadFile("../etest/test_sync2/test.go")
	goByteMd5Map[eutil.Md5(string(destGoBytes))] = []econfig.SameMd5Dir{
		econfig.SameMd5Dir{FilePath: "etest/test_sync2/test.go", Dir: "etest/test_sync2"},
	}

	//txt文件
	txtBytes, _ := ioutil.ReadFile("../etest/test_sync1/test.txt")
	txtByteMd5Map := make(map[string][]econfig.SameMd5Dir)
	destTxtBytes, _ := ioutil.ReadFile("../etest/test_sync2/test.txt")
	txtByteMd5Map[eutil.Md5(string(destTxtBytes))] = []econfig.SameMd5Dir{
		econfig.SameMd5Dir{FilePath: "etest/test_sync2/test.txt", Dir: "etest/test_sync2"},
	}

	//sql文件
	sqlBytes, _ := ioutil.ReadFile("../etest/test_sync1/test.sql")
	sqlByteMd5Map := make(map[string][]econfig.SameMd5Dir)
	destSqlBytes, _ := ioutil.ReadFile("../etest/test_sync2/test.sql")
	sqlByteMd5Map[eutil.Md5(string(destSqlBytes))] = []econfig.SameMd5Dir{
		econfig.SameMd5Dir{FilePath: "etest/test_sync2/test.sql", Dir: "etest/test_sync2"},
	}
	files := []*econfig.FileInfo{
		&econfig.FileInfo{Name: "test.go", Path: "", Byte: string(goBytes), ByteMd5Map: goByteMd5Map},
		&econfig.FileInfo{Name: "test.txt", Path: "", Byte: string(txtBytes), ByteMd5Map: txtByteMd5Map},
		&econfig.FileInfo{Name: "test.sql", Path: "", Byte: string(sqlBytes), ByteMd5Map: sqlByteMd5Map},
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func(waitCtx context.Context) {
	WAIT:
		for {
			select {
			case <-waitCtx.Done():
				break WAIT
			case dir := <-ch:
				ch_dir = append(ch_dir, dir)
			}
		}
	}(ctx)

	os.Chdir("../")
	CallCompareSync("etest/test_sync1", files, ch)
	cancel()
	if len(ch_dir) > 0 {
		t.Error("CallCompareSync fail")
	}

}