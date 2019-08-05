package main

import (
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

const (
	fileName = "client.file"
)

func init() {
	log.SetPrefix("[CLIENT]")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func main() {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:10000", time.Second*3)
	if err != nil {
		log.Fatalf("connect to server error, %s\n", err)
	}
	defer conn.Close()
	clientDeal(conn)
}

func clientDeal(conn net.Conn) {
	// 先告诉服务端开始传输文件
	writeToServer(conn, []byte("start"))
	off := getOffset(conn)
	// 打开文件
	fp, err := os.OpenFile(fileName, os.O_RDONLY, 0755)
	if err != nil {
		log.Fatalf("open file error, %s\n", err)
	}
	defer fp.Close()
	// 文件指针移动到偏移位置
	_, err = fp.Seek(off, 0)
	if err != nil {
		log.Fatalf("move file pointer error, %s\n", err)
	}
	buf := make([]byte, 100)
	for {
		n, err := fp.Read(buf)
		if err != nil {
			if err == io.EOF {
				writeToServer(conn, []byte("finish"))
				log.Println("read EOF")
				return
			}
			log.Fatalf("read file error, %s\n", err)
		}
		log.Printf("read file success, content:%s\n", string(buf[:n]))
		writeToServer(conn, buf[:n])
	}
}

func writeToServer(conn net.Conn, content []byte) {
	_, err := conn.Write(content)
	if err != nil {
		log.Fatalf("write to server error, %s\n", err)
	}
	log.Printf("write to server success, content:%s", string(content))
}

func getOffset(conn net.Conn) int64 {
	buf := make([]byte, 8)
	n, err := conn.Read(buf)
	if err != nil {
		log.Fatalf("get size from server error, %s\n", err)
	}
	size, err := strconv.ParseInt(string(buf[:n]), 10, 64)
	if err != nil {
		log.Fatalf("trans string to int error, %s\n", err)
	}
	return size
}
