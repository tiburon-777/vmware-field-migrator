package app

import (
	"main/vmware/client"
)

type App struct {
	VmwareClient client.Pool
	Config       Conf
}

func NewApp(conf Conf) (App, error) {
	vmClient, err := client.NewPool(conf, 10)
	if err != nil {
		return App{}, err
	}
	return App{VmwareClient: vmClient, Config: conf}, nil
}
