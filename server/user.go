package server

const (
	defaultUser = "client"
	defaultPw   = "12345"
)

type user struct {
	// 用户名
	name string
	// 其他信息
	other string
}

// login 用户登陆
func login(name, pw string) (*user, bool) {
	if name != defaultUser {
		return nil, false
	}
	if pw != defaultPw {
		return nil, false
	}
	u := &user{name, ""}
	return u, true
}
