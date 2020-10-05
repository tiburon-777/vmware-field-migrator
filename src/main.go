package main

import (
	"flag"
	"log"
	"main/models"
	"main/vmware"
)

var Conf models.Conf

func init() {
	flag.StringVar(&Conf.Server, "vcenter", "", "VMWare VCenter address")
	flag.StringVar(&Conf.Login, "login", "", "Login")
	flag.StringVar(&Conf.Domain, "domain", "", "Domain")
	flag.StringVar(&Conf.Pass, "pass", "", "Password")
	flag.IntVar(&Conf.Threads, "threads", 5, "Number of parallel VMWare sessions")
	flag.StringVar(&Conf.FieldProject, "fproject", "autoProject", "Field name for project accociation")
	flag.StringVar(&Conf.FieldExpire, "fexpire", "autoExpire", "Field name for expiration date")
}

func main() {
	flag.Parse()
	err := vmware.FieldMigrator(Conf)
	if err != nil {
		log.Fatal(err.Error())
	}
}
