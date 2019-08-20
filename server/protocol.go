package server

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

const (
	bigType   = 1 // 大文件类型
	splitType = 2 // 拆分文件类型
	stopType  = 3 // 停止上传
	endType   = 4 // 上传结束
)

// analyzeOp 解析客户端的操作请求
// 上传大文件：big {file_name} {file_size}
// 上传拆分后的文件：split {unique_id} {file_index}
// 停止上传文件：stop {unique_id} {file_index}
// 上传完成：end {unique_id} {file_index}
func analyzeOp(opStr string) (int, string, int64, error) {
	opArr := strings.Split(opStr, " ")
	if len(opArr) != 3 {
		log.Printf("protocol is error, %s\n", opStr)
		return 0, "", 0, fmt.Errorf("protocol error")
	}
	var t int
	switch opArr[0] {
	case "big":
		t = bigType
		break
	case "split":
		t = splitType
		break
	case "stop":
		t = stopType
		break
	case "end":
		t = endType
		break
	default:
		log.Printf("protocol is error, %s\n", opStr)
		return 0, "", 0, fmt.Errorf("protocol error")
	}
	pint, err := strconv.ParseInt(opArr[2], 10, 64)
	if err != nil {
		log.Printf("fail to convert str to integer, %s, %s\n", opArr[2], err)
		return 0, "", 0, err
	}
	return t, opArr[1], pint, nil
}

// analyzeLogin 用户登陆解析
// 协议：login {user} {pw}
// return:user, pw
func analyzeLogin(loginStr string) (string, string, error) {
	loginArr := strings.Split(loginStr, " ")
	if len(loginArr) != 3 || loginArr[0] != "login" {
		log.Printf("protocol is error. %s\n", loginStr)
		return "", "", fmt.Errorf("protocol error")
	}
	return loginArr[1], loginArr[2], nil
}
