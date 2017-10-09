package installer

var managers = make(map[string]Manager)

func RegisterManager(m Manager) {
	managers[m.Name()] = m
}

func GetManager(name string) Manager {
	return managers[name]
}
