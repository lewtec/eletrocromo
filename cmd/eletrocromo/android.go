package main

type androidCmd struct {
	create *androidCreateCmd
	init   *androidCreateCmd `cmd:"init"`
	build  *buildAndroidCmd
}

func (androidCmd) Description() string {
	return "Legacy Android commands. Prefer \"eletrocromo build android\" and \"eletrocromo build icons\"."
}
