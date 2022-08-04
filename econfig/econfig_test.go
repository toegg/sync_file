package econfig

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Chdir("../")
	m.Run()
}

func TestGetIniProject(t *testing.T) {
	list, err := GetIniProject()
	if err != nil{
		t.Errorf("GetIniProject Err:%v", err.Error())
		return
	}
	if len(list) <= 0 {
		t.Error("GetIniProject is empty")
	}
}

func TestGetAllowExts(t *testing.T) {
	list, err := GetAllowExts()
	if err != nil{
		t.Errorf("GetAllowExts Err:%v", err.Error())
		return
	}
	if len(list) <= 0 {
		t.Error("GetAllowExts is empty")
	}
}

func TestGetComparePath(t *testing.T) {
	path, err := GetComparePath()
	if err != nil{
		t.Errorf("GetComparePath Err:%v", err.Error())
		return
	}
	if path == ""{
		t.Error("GetComparePath path is empty")
	}
}
