package eutil

import(
	"testing"
)

func TestIsAllowExt (t *testing.T){
	allowExts := []string{".txt", ".sql", ".go"}
	txtRes := IsAllowExt("test.txt", allowExts)
	sqlRes := IsAllowExt("test.sql", allowExts)
	goRes := IsAllowExt("test.go", allowExts)
	if !txtRes{
		t.Error(".txt not allow")
	}
	if !sqlRes{
		t.Error(".sql not allow")
	}
	if !goRes{
		t.Error(".go not allow")
	}
}

func TestGetString(t *testing.T) {
	var num int = 1
	if GetString(num) != "1" {
		t.Error("int trans fail")
	}
	var str string = "abc"
	if GetString(str) != "abc" {
		t.Error("str trans fail")
	}
	var strArray []string = []string{"abc", "d", "ef"}
	if GetString(strArray) != "abcdef" {
		t.Error("[]string trans fail")
	}
	var byte []byte = []byte{65,66,67}
	if GetString(byte) != "ABC" {
		t.Error("[]string trans fail")
	}
}

func TestFileExists(t *testing.T) {
	if !FileExists("../econfig/econfig.go"){
		t.Error("FileExists fail")
	}
}

func TestWriteToFile(t *testing.T) {
	err := WriteToFile("../etest/test_sync1/test.txt", "../etest/test_sync1", []byte("abc 123"))
	if err !=nil{
		t.Errorf("WriteToFile Err:%v", err)
	}
}