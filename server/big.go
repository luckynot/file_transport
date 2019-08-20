package server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

const singleMaxSize = int64(1024 * 1024)

type fileServer struct {
	usr *user
	// 唯一id
	uid string
	// 连接
	conn net.Conn
	// 文件大小
	size int64
	// 文件个数
	num int
	// 文件名
	fn string
	// 单个拆分文件处理服务
	split []*singleFileServer
	mu    sync.Mutex
}

// upload 上传大文件
func (fs *fileServer) upload() {
	// 生成唯一id
	fs.genID()
	// 存储fs
	ok := allfsAdd(fs)
	// 防止用户多处登陆并发上传
	if !ok {
		log.Printf("store file server error,uid:%s\n", fs.uid)
		return
	}
	// 计算文件拆分方案
	fs.calSplitNum()
	// 回复客户端文件拆分方案
	err := fs.sendSplit()
	if err != nil {
		return
	}
	// 监听客户端的操作
	fs.listenOp()
}

// genId 生成这次上传的唯一id:远程地址_文件名
func (fs *fileServer) genID() {
	fs.uid = fmt.Sprintf("%s_%s", fs.usr.name, fs.fn)
}

// calSplitNum 计算文件应该拆分的个数
// 规定每个文件的最大大小进行拆分
func (fs *fileServer) calSplitNum() {
	num := fs.size / singleMaxSize
	if fs.size%singleMaxSize != 0 {
		num++
	}
	fs.num = int(num)
	fs.split = make([]*singleFileServer, num)
}

// splitFile 回复客户端文件拆分方案
func (fs *fileServer) sendSplit() error {
	res := fmt.Sprintf("%d %s", fs.num, fs.uid)
	err := writeBufferTimeOut(fs.conn, []byte(res))
	if err != nil {
		log.Println("write the scheme to client error")
	}
	return err
}

// listenOp 监听客户端的操作指令
func (fs *fileServer) listenOp() {
	// 允许错误指令的次数为10次
	var errTime = 0
	for errTime < 10 {
		buffer, n, err := readBuffer(fs.conn)
		if err != nil {
			if err == io.EOF {
				fs.stopAll()
				return
			}
			log.Printf("listen opration from client fail, %s\n", buffer[:n])
			errTime++
			continue
		}
		op := string(buffer[:n])
		opType, uid, fidx, err := analyzeOp(op)
		if err != nil {
			errTime++
			continue
		}
		switch opType {
		case stopType:
			if uid != fs.uid {
				log.Printf("[stop] uid invalid, %s\n", uid)
				errTime++
				continue
			}
			fs.stop(int(fidx))
			break
		case endType:
			if uid != fs.uid {
				log.Printf("[end] uid invalid, %s\n", uid)
				errTime++
				continue
			}
			ok := fs.end(int(fidx))
			if ok {
				// 组装文件
				if fs.assembFile() {
					writeBufferTimeOut(fs.conn, []byte("success"))
				} else {
					writeBufferTimeOut(fs.conn, []byte("assemb error"))
				}
				return
			}
			break
		default:
			log.Printf("op type invalid, %d\n", opType)
			errTime++
			continue
		}
	}
	fs.stopAll()
}

// add 添加single file server
func (fs *fileServer) add(sfs *singleFileServer) bool {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.split[sfs.idx] != nil {
		return false
	}
	// 超出拆分文件数量
	if len(fs.split) <= sfs.idx {
		return false
	}
	fs.split[sfs.idx] = sfs
	return true
}

// stopAll 停止这次文件的上传
func (fs *fileServer) stopAll() {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	allfsDelete(fs.uid)
	for _, sfs := range fs.split {
		if sfs == nil {
			continue
		}
		sfs.stop()
	}
}

// stop 停止单个拆分文件上传
func (fs *fileServer) stop(idx int) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	sfs := fs.split[idx]
	fs.split[idx] = nil
	sfs.stop()
}

// end 单个拆分文件上传结束
func (fs *fileServer) end(idx int) bool {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	sfs := fs.split[idx]
	if sfs != nil {
		sfs.stop()
		fs.split[idx] = nil
	}
	return fs.checkSuc()
}

// checkSuc 检查是否所有拆分文件上传成功
func (fs *fileServer) checkSuc() bool {
	// 校验是否所有文件上传结束
	var sum int64
	var fn string
	for i := 0; i < fs.num; i++ {
		fn = fmt.Sprintf("%s_%d", fs.uid, i)
		size, err := getFileSize(fn)
		if err != nil {
			continue
		}
		sum += size
	}
	return sum == fs.size
}

// assembFile 组装拆分的文件
// 不需要加锁，因为一个文件只会有一个fileServer
func (fs *fileServer) assembFile() bool {
	var fp *os.File
	res, err := os.OpenFile(fs.uid, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		log.Printf("assemb file error, uid:%s\n", fs.uid)
		return false
	}
	defer res.Close()
	bw := bufio.NewWriter(res)
	buf := make([]byte, 1024)
	// 解析文件名
	var fns = make([]string, fs.num)
	for i := 0; i < fs.num; i++ {
		fns[i] = fmt.Sprintf("%s_%d", fs.uid, i)
	}
	for _, fn := range fns {
		fp, err = os.OpenFile(fn, os.O_RDONLY, 0755)
		if err != nil {
			log.Printf("assemb file error, uid:%s\n", fs.uid)
			return false
		}
		defer fp.Close()
		br := bufio.NewReader(fp)
		for {
			n, err := br.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
				return false
			}
			bw.Write(buf[:n])
		}
		if err := bw.Flush(); err != nil {
			return false
		}
	}
	// 合并成功后，删除拆分的文件
	for _, fn := range fns {
		if err := os.Remove(fn); err != nil {
			log.Printf("remove file error, file name=%s\n", fn)
		}
	}
	return true
}
