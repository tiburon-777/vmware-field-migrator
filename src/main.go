package main

import (
	"flag"
	"log"
	"main/app"
	"main/vmware"
)

var Conf app.Conf

func init() {
	flag.StringVar(&Conf.Server, "VCenter", "", "VMWare VCenter address")
	flag.StringVar(&Conf.Login, "Login", "", "Login")
	flag.StringVar(&Conf.Domain, "Domain", "", "Domain")
	flag.StringVar(&Conf.Pass, "Password", "", "Password")
	flag.IntVar(&Conf.Threads, "Threads", 5, "Number of parallel VMWare sessions")
	flag.StringVar(&Conf.FieldProject, "FieldProject", "autoProject", "Field name for project accociation")
	flag.StringVar(&Conf.FieldExpire, "FieldExpire", "autoExpire", "Field name for expiration date")
}

func main() {
	flag.Parse()
	a, err := app.NewApp(Conf)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = vmware.FieldMigrator(a)
	if err != nil {
		log.Fatal(err.Error())
	}
}
