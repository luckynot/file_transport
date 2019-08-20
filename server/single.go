package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
)

type singleFileServer struct {
	// 连接
	conn net.Conn
	// 唯一id
	uid string
	// 分拆序号
	idx int
	// 该文件的大小
	size int64
	// 本地文件名
	fn string
	// 是否允许上传
	allowed bool
	// 总的文件服务
	fs *fileServer
}

// upload 上传单个文件
func (sfs *singleFileServer) upload() {
	fs, ok := allfsGet(sfs.uid)
	if !ok {
		log.Printf("get file server error,uid:%s\n", sfs.uid)
		return
	}
	ok = fs.add(sfs)
	if !ok {
		log.Printf("file index error, uid:%s,index:%d\n", sfs.uid, sfs.idx)
		return
	}
	sfs.fs = fs
	sfs.setName()
	sfs.setSize()
	sfs.receiveFile()
}

// setName 设置拆分文件在本地存储的文件名
func (sfs *singleFileServer) setName() {
	sfs.fn = fmt.Sprintf("%s_%d", sfs.uid, sfs.idx)
}

// setSize 设置文件的大小
func (sfs *singleFileServer) setSize() {
	// 最后一个index
	if sfs.idx == sfs.fs.num-1 {
		sfs.size = sfs.fs.size - singleMaxSize*int64(sfs.idx)
	} else {
		sfs.size = singleMaxSize
	}
}

// receiveFile 接收文件
func (sfs *singleFileServer) receiveFile() {
	size, err := getFileSize(sfs.fn)
	if err != nil {
		writeBufferTimeOut(sfs.conn, []byte("server exception"))
		return
	}
	// 回复客户端断点续传位置
	err = writeBufferTimeOut(sfs.conn, []byte(strconv.FormatInt(size, 10)))
	if err != nil {
		return
	}
	// 打开文件，不存在时创建，存在时追加写
	fp, err := os.OpenFile(sfs.fn, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)
	if err != nil {
		log.Printf("open file error, %s\n", err)
		writeBufferTimeOut(sfs.conn, []byte("server exception"))
		return
	}
	defer fp.Close()
	var buf []byte
	var n int
	// 从buffer中读取数据，写入文件
	for sfs.allowed && size < sfs.size {
		buf, n, err = readBufferTimeOut(sfs.conn)
		if err != nil {
			if err == io.EOF {
				log.Println("read EOF!")
				break
			}
			log.Printf("conn read error, %s\n", err)
			return
		}
		if len(buf) == 0 {
			continue
		}
		_, err = fp.Write(buf[:n])
		if err != nil {
			log.Printf("write file error, %s\n", err)
			writeBufferTimeOut(sfs.conn, []byte("server exception"))
			return
		}
		size += int64(n)
	}
}

// getFileSize 获取文件大小
func getFileSize(fn string) (int64, error) {
	fInfo, err := os.Stat(fn)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("file size is 0")
			return 0, nil
		}
		log.Printf("get file size error, %s\n", err)
		return 0, err
	}
	log.Printf("file size is %d\n", fInfo.Size())
	return fInfo.Size(), nil
}

// stop 停止文件上传
func (sfs *singleFileServer) stop() {
	sfs.allowed = false
}
