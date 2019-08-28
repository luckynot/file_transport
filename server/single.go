package server

import (
	"io"
	"log"
	"net"
	"os"
	"strconv"
)

type singleFileServer struct {
	conn    net.Conn    // 连接
	uid     string      // 唯一id
	idx     int         // 分拆序号
	size    int64       // 该文件的大小
	fn      string      // 本地文件名
	allowed bool        // 是否允许上传
	fs      *fileServer // 总的文件服务
}

// receive 接收单个文件
func (sfs *singleFileServer) reveive() {
	fs, ok := allfsGet(sfs.uid)
	if !ok {
		log.Printf("获取总文件服务失败,uid:%s\n", sfs.uid)
		return
	}
	ok = fs.add(sfs)
	if !ok {
		log.Printf("文件的序号错误, uid:%s,index:%d\n", sfs.uid, sfs.idx)
		return
	}
	sfs.fs = fs
	sfs.setName()
	sfs.setSize()
	sfs.receiveFile()
}

// setName 设置拆分文件在本地存储的文件名
func (sfs *singleFileServer) setName() {
	sfs.fn = sfs.fs.singlefilename(sfs.idx)
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
	// 打开文件，不存在时创建，存在时追加写
	fp, err := os.OpenFile(sfs.fn, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0766)
	if err != nil {
		log.Printf("打开文件失败, %s\n", err)
		writeBufferTimeOut(sfs.conn, []byte("server exception"))
		return
	}
	defer fp.Close()
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
	var buf []byte
	var n int
	// 从buffer中读取数据，写入文件
	for sfs.allowed && size < sfs.size {
		buf, n, err = readBufferTimeOut(sfs.conn)
		if err != nil {
			if err == io.EOF {
				log.Println("拆分文件上传读取EOF!")
				break
			}
			log.Printf("拆分文件上传读取错误, %s\n", err)
			return
		}
		if len(buf) == 0 {
			continue
		}
		_, err = fp.Write(buf[:n])
		if err != nil {
			log.Printf("拆分文件上传写入文件错误, %s\n", err)
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
			log.Printf("%s大小是0", fn)
			return 0, nil
		}
		log.Printf("获取%s大小错误, %s\n", fn, err)
		return 0, err
	}
	log.Printf("%s大小是%d\n", fn, fInfo.Size())
	return fInfo.Size(), nil
}

// stop 停止文件上传
func (sfs *singleFileServer) stop() {
	sfs.allowed = false
}
