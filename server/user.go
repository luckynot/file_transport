package server

type user struct {
	// 用户名
	name string
	// 其他信息
	other string
}

// login 用户登陆
func login(name, pw string) (*user, bool) {
	u := &user{name, ""}
	return u, true
}
