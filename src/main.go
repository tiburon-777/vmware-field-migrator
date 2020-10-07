package main

import (
	"flag"
	"io/ioutil"
	"log"
	"main/models"
	"main/vmware"
	"os"
	"strings"
)

var Conf models.Conf
var fName string

func init() {
	flag.StringVar(&Conf.Server, "vcenter", "", "VMWare VCenter address")
	flag.StringVar(&Conf.Login, "login", "", "Login")
	flag.StringVar(&Conf.Domain, "domain", "", "Domain")
	flag.StringVar(&Conf.Pass, "pass", "", "Password")
	flag.IntVar(&Conf.Threads, "threads", 5, "Number of parallel VMWare sessions")
	flag.StringVar(&Conf.FieldProject, "fproject", "autoProject", "Field name for project accociation")
	flag.StringVar(&Conf.FieldExpire, "fexpire", "autoExpire", "Field name for expiration date")
	flag.StringVar(&fName, "origins", "origins.txt", "File with origins, delimeters by ';'")
}

func main() {
	flag.Parse()

	// Открываем файл с образцами и перегоняем все в мапу
	Conf.Origins = make(map[string]int)
	fOrigins, err := os.Open(fName)
	if err != nil {
		log.Fatalf("не удалось открыть файл с образцами")
	}
	defer fOrigins.Close()
	b, err := ioutil.ReadAll(fOrigins)
	if err != nil {
		log.Fatalf("не удалось прочитать файл с образцами")
	}
	for _, v := range strings.Split(string(b), ";") {
		Conf.Origins[v] = 1
	}

	// Запускаем мигратор
	err = vmware.FieldMigrator(Conf)
	if err != nil {
		log.Fatal(err.Error())
	}
}
