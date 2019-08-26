package client

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	fileName    = "client.file"
	defaultUser = "client"
	defaultPw   = "12345"
)

type client struct {
	conn  net.Conn       // 连接
	usr   string         // 用户名
	pw    string         // 密码
	uid   string         // 唯一id
	fn    string         // 文件名
	tsize int64          // 文件总大小
	ssize int64          // 单个拆分文件大小
	wg    sync.WaitGroup // 记录拆分文件上传协程
}

func init() {
	log.SetPrefix("[CLIENT]")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

// connServer 连接服务端
func connServer() (net.Conn, error) {
	errTime := 0
	for errTime < 10 {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:10000", time.Second*3)
		if err != nil {
			log.Printf("连接服务器失败, %s\n", err)
			errTime++
			time.Sleep(time.Second * 1)
			continue
		}
		log.Printf("连接成功, remote:%s, local:%s\n", conn.RemoteAddr().String(), conn.LocalAddr().String())
		return conn, nil
	}
	return nil, fmt.Errorf("连接超时")
}

// Upload 上传文件
func Upload(fn string) bool {
	// 获取文件大小
	size, err := getFileSize(fn)
	if err != nil || 0 == size {
		return false
	}
	conn, err := connServer()
	if err != nil {
		return false
	}
	defer conn.Close()
	cli := &client{
		conn:  conn,
		usr:   defaultUser,
		pw:    defaultPw,
		fn:    fn,
		tsize: size,
	}
	ok, err := login(cli.conn, cli.usr, cli.pw)
	if err != nil || !ok {
		log.Printf("登陆失败, fn:%s\n", fn)
		return false
	}
	if err = cli.splitScheme(); err != nil {
		return false
	}
	fnum := cli.fileNum()
	for {
		cli.wg.Add(fnum)
		for i := 0; i < fnum; i++ {
			go cli.uploadSplitFile(i)
		}
		cli.wg.Wait()
		if cli.endUpload() {
			break
		}
	}
	return true
}

// getFileSize 获取文件大小
func getFileSize(fn string) (int64, error) {
	fInfo, err := os.Stat(fn)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("%s大小为0\n", fn)
			return 0, nil
		}
		log.Printf("获取%s大小错误, %s\n", fn, err)
		return 0, err
	}
	log.Printf("%s大小为：%d\n", fn, fInfo.Size())
	return fInfo.Size(), nil
}

// fileNum 获取拆分文件的个数
func (cli *client) fileNum() int {
	fnum := cli.tsize / cli.ssize
	if cli.tsize%cli.ssize != 0 {
		return int(fnum) + 1
	}
	return int(fnum)
}

// uploadSplitFile 上传分拆文件
func (cli *client) uploadSplitFile(idx int) {
	defer cli.wg.Done()
	fp, err := os.OpenFile(cli.fn, os.O_RDONLY, 0755)
	if err != nil {
		log.Printf("打开文件失败, idx:%d, err:%s\n", idx, err)
		return
	}
	defer fp.Close()
	// 拆分文件上传新建连接
	conn, err := connServer()
	if err != nil {
		return
	}
	defer conn.Close()
	ok, err := login(conn, cli.usr, cli.pw)
	if err != nil || !ok {
		log.Printf("登陆失败, idx:%d, err:%s\n", idx, err)
		return
	}
	ctn, err := ctnLoc(conn, cli.uid, idx)
	if err != nil {
		return
	}
	offset := int64(idx)*cli.ssize + ctn
	log.Printf("文件移动位置:%d, uid:%s, idx:%d\n", offset, cli.uid, idx)
	if _, err = fp.Seek(offset, 0); err != nil {
		log.Printf("移动文件指针失败,uid:%s, idx:%d, err:%s\n", cli.uid, idx, err)
		return
	}
	// todo 从服务端获取续传位置
	buf := make([]byte, 1024)
	var n int
	var totalSize = ctn
	for totalSize < cli.ssize {
		if n, err = fp.Read(buf); err != nil {
			if err == io.EOF {
				log.Printf("读取结束EOF, uid:%s, idx:%d\n", cli.uid, idx)
				break
			}
			log.Printf("文件读取错误, uid:%s, idx:%d, err:%s\n", cli.uid, idx, err)
			return
		}
		totalSize += int64(n)
		// 文件读取超过了分拆文件的大小
		if totalSize > cli.ssize {
			n -= int(totalSize - cli.ssize)
		}
		if err = writeBufferTimeOut(conn, buf[:n]); err != nil {
			log.Printf("写文件到net buffer错误, uid:%s, idx:%d, err:%s\n", cli.uid, idx, err)
			return
		}
	}
}

// ctnLoc 从服务端获取续传位置
func ctnLoc(conn net.Conn, uid string, idx int) (int64, error) {
	split := fmt.Sprintf("split %s %d", uid, idx)
	if err := writeBufferTimeOut(conn, []byte(split)); err != nil {
		log.Printf("传送分拆文件信息到服务端错误, uid:%s, idx:%d, err:%s\n", uid, idx, err)
		return 0, err
	}
	locB, n, err := readBufferTimeOut(conn)
	if err != nil {
		log.Printf("获取文件续传位置失败, uid:%s, idx:%d, err:%s\n", uid, idx, err)
		return 0, err
	}
	loc, err := strconv.ParseInt(string(locB[:n]), 10, 64)
	if err != nil {
		log.Printf("获取文件续传位置失败, uid:%s, idx:%d, loc:%s, err:%s\n", uid, idx, string(locB[:n]), err)
		return 0, err
	}
	return loc, nil
}

// endUpload 客户端结束上传
func (cli *client) endUpload() bool {
	// 客户端主连接向服务端发送当前id结束信号
	endStr := fmt.Sprintf("end %s %d", cli.uid, 0)
	if err := writeBufferTimeOut(cli.conn, []byte(endStr)); err != nil {
		log.Printf("发送结束信号到服务端失败, uid:%s, err:%s\n", cli.uid, err)
		return false
	}
	resB, n, err := readBufferTimeOut(cli.conn)
	if err != nil {
		log.Printf("获取结束结果失败, uid:%s, err:%s\n", cli.uid, err)
		return false
	}
	res := string(resB[:n])
	log.Printf("发送结束信号，服务端返回结果:%s\n", res)
	return res == "success"
}
