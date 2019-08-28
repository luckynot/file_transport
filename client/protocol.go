package client

import (
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
)

// login 登陆服务器,成功返true
func login(conn net.Conn, usr, pw string) (bool, error) {
	loginStr := fmt.Sprintf("login %s %s", usr, pw)
	err := writeBufferTimeOut(conn, []byte(loginStr))
	if err != nil {
		log.Printf("发送登陆信息到服务端失败, login:%s, err:%s\n", loginStr, err)
		return false, err
	}
	buf, n, err := readBufferTimeOut(conn)
	if err != nil {
		return false, err
	}
	res := string(buf[:n])
	switch res {
	case "success":
		return true, nil
	case "fail":
		return false, nil
	default:
		log.Printf("登陆返回协议错误, res:%s\n", res)
		return false, nil
	}
}

// splitScheme 从服务端获取拆分方案
func (cli *client) splitScheme() error {
	upStr := fmt.Sprintf("big %s %d", cli.fn, cli.tsize)
	if err := writeBufferTimeOut(cli.conn, []byte(upStr)); err != nil {
		return err
	}
	buf, n, err := readBufferTimeOut(cli.conn)
	if err != nil {
		return err
	}
	schemeStr := string(buf[:n])
	scheme := strings.Split(schemeStr, " ")
	if len(scheme) != 2 {
		log.Printf("拆分协议错误, scheme:%s\n", schemeStr)
		return fmt.Errorf("protocol error")
	}
	ssize, err := strconv.ParseInt(scheme[0], 10, 64)
	if err != nil {
		log.Printf("拆分协议错误, scheme:%s, err:%s\n", schemeStr, err)
		return err
	}
	cli.ssize = ssize
	cli.uid = scheme[1]
	return nil
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
			log.Printf("从buffer中读取错误, %s\n", err)
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
		log.Printf("写入buffer失败, content:%s, %s\n", string(content), err)
		return err
	}
	return nil
}
