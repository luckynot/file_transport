package server

import (
	"io"
	"log"
	"net"
	"strings"
)

const port = "10000"

func init() {
	log.SetPrefix("[SERVER]")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

// Start 服务端启动方法
func Start() {
	log.Println("服务器启动中")
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("端口监听错误 %s, %s\n", port, err)
	}
	defer l.Close()
	log.Println("等待连接....")
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("连接失败, %s\n", err)
			continue
		}
		log.Println("连接成功")
		go serverDeal(conn)
	}
}

// serverDeal 服务端处理
func serverDeal(conn net.Conn) {
	defer conn.Close()
	usr, ok := clientVerify(conn)
	if !ok {
		writeBufferTimeOut(conn, []byte("fail"))
		return
	}
	writeBufferTimeOut(conn, []byte("success"))
	buf, n, err := readBufferTimeOut(conn)
	if err != nil {
		return
	}
	opStr := string(buf[:n])
	opType, pstr, pint, err := analyzeOp(opStr)
	if err != nil {
		return
	}
	switch opType {
	case bigType:
		// 上传大文件
		// 文件名格式错误
		if strings.HasSuffix(pstr, "/") {
			log.Printf("文件名格式错误,name:%s\n", pstr)
			return
		}
		var fs = &fileServer{
			usr:  usr,
			conn: conn,
			size: pint,
			fn:   pstr,
		}
		fs.receive()
		break
	case splitType:
		// 验证用户是否正确
		if !strings.HasPrefix(pstr, usr.name+"_") {
			writeBufferTimeOut(conn, []byte("用户非法"))
			log.Printf("用户非法,usr:%s\n", usr.name)
			return
		}
		// 上传拆分的文件
		var sfs = &singleFileServer{
			conn:    conn,
			uid:     pstr,
			idx:     int(pint),
			allowed: true,
		}
		sfs.reveive()
		break
	default:
		log.Printf("操作类型错误, %d\n", opType)
		return
	}
}

// userVerify 用户验证，成功返回用户
func clientVerify(conn net.Conn) (*user, bool) {
	buf, n, err := readBufferTimeOut(conn)
	if err != nil {
		return nil, false
	}
	loginStr := string(buf[:n])
	name, pw, err := analyzeLogin(loginStr)
	if err != nil {
		return nil, false
	}
	return login(name, pw)
}

// readBufferTimeOut 从缓冲区读取字节，过期两秒
func readBufferTimeOut(conn net.Conn) ([]byte, int, error) {
	// conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	return readBuffer(conn)
}

// readBuffer 从缓冲区读取字节
func readBuffer(conn net.Conn) ([]byte, int, error) {
	var buf = make([]byte, 1000)
	n, err := conn.Read(buf)
	if err != nil {
		if err != io.EOF {
			log.Printf("buffer读取错误, %s\n", err)
		}
		return nil, 0, err
	}
	return buf, n, nil
}

// writeBufferTimeOut 写数据到缓冲区，过期两秒
func writeBufferTimeOut(conn net.Conn, content []byte) error {
	// conn.SetWriteDeadline(time.Now().Add(time.Second * 2))
	return writeBuffer(conn, content)
}

// writeBuffer 写数据到缓冲区
func writeBuffer(conn net.Conn, content []byte) error {
	_, err := conn.Write(content)
	if err != nil {
		log.Printf("buffer写入错误, content:%s, %s\n", string(content), err)
		return err
	}
	return nil
}
