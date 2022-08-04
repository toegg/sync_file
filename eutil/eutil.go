package eutil

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

func IsAllowExt(file string, extOpens []string) bool{
	for _, v := range extOpens {
		if strings.Contains(file, v){
			return true
		}
	}
	return false
}

//转为string
func GetString(v interface{}) string {
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
func FileExists(path string) (bool) {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

//写入文件
func WriteToFile(filePath string, destDir string, byte []byte) error{
	if _, err := os.Stat(destDir); err != nil {
		err = os.MkdirAll(destDir, 0711)
		if err != nil{
			return err
		}
	}
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	defer f.Close()
	if err !=nil {
		return err
	}
	_, err = f.Write(byte)
	return err
}

//md5加密
func Md5(data string) string {
	h := md5.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}