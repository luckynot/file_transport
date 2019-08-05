package main

import (
	"io"
	"log"
	"net"
	"os"
	"strconv"
)

const (
	fileName = "server.file"
)

func init() {
	log.SetPrefix("[SERVER]")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func main() {
	l, err := net.Listen("tcp", "127.0.0.1:10000")
	if err != nil {
		log.Fatalf("Can't listen to port 10000, %s\n", err)
	}
	defer l.Close()
	log.Println("waiting accept....")
	conn, err := l.Accept()
	if err != nil {
		log.Fatalf("accept error, %s\n", err)
	}
	defer conn.Close()
	serverDeal(conn)
}

// 服务端处理
func serverDeal(conn net.Conn) {
	var start = false
	for {
		var buf = make([]byte, 100)
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				log.Println("read EOF!")
				return
			}
			log.Fatalf("conn read error, %s\n", err)
		}
		log.Printf("read from client, %s\n", string(buf[:n]))
		if !start && "start" == string(buf[:n]) {
			size := getFileSize()
			_, err = conn.Write([]byte(strconv.FormatInt(size, 10)))
			if err != nil {
				log.Fatalf("send file's size to client error, %s\n", err)
			}
			start = true
			continue
		} else if "finish" == string(buf[:n]) {
			log.Println("receive finish signal")
			return
		}
		writeFile(buf[:n])
	}
}

// 获取文件大小，断点续传用
func getFileSize() int64 {
	fInfo, err := os.Stat(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("file size is 0")
			return int64(0)
		}
		log.Fatalf("get file size error, %s\n", err)
	}
	log.Printf("file size is %d\n", fInfo.Size())
	return fInfo.Size()
}

// 写文件
func writeFile(content []byte) {
	if len(content) == 0 {
		return
	}
	// 打开文件，不存在时创建，存在时追加写
	fp, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)
	if err != nil {
		log.Fatalf("open file error, %s\n", err)
	}
	defer fp.Close()
	_, err = fp.Write(content)
	if err != nil {
		log.Fatalf("write file error, %s\n", err)
	}
	log.Printf("write file success, content:%s\n", string(content))
}
