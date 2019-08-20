package server

import "sync"

// 存储所有的file server, key=uid, value=&fileServer
var allfs sync.Map

// allfsAdd 存储file server
// 如果不存在，则存储，并返回true
// 如果存在，则不存储，并返回false
func allfsAdd(fs *fileServer) bool {
	_, ok := allfs.LoadOrStore(fs.uid, fs)
	return !ok
}

// allfsGet 根据uid获取file server
func allfsGet(uid string) (*fileServer, bool) {
	val, ok := allfs.Load(uid)
	if !ok {
		return nil, false
	}
	return val.(*fileServer), true
}

// allfsDelete 删除uid对应的file server
func allfsDelete(uid string) {
	allfs.Delete(uid)
}
