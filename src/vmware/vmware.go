package vmware

import (
	"log"
	"main/models"
	"main/vmware/client"
	"strings"
	"sync"
	"time"

	"github.com/vmware/govmomi/vim25/mo"
)

// Fan-ы.
func FieldMigrator(conf models.Conf) error {
	log.Println("Запущена миграция полей VMWare в ", conf.Threads, "воркерах")
	pool, err := client.NewPool(conf, 10)
	if err != nil {
		return err
	}
	vms, err := getAllVMs(pool)
	if err != nil {
		return err
	}

	vm, errs, wg, done := make(chan mo.VirtualMachine), make(chan error, len(vms)), sync.WaitGroup{}, make(chan interface{})
	defer func() {
		close(done)
		wg.Wait()
	}()
	wg.Add(conf.Threads)
	for i := 1; i <= conf.Threads; i++ {
		go func(done <-chan interface{}, vm chan mo.VirtualMachine, wg *sync.WaitGroup) {
			for {
				select {
				case <-done:
					wg.Done()
					return
				case v := <-vm:
					if err := migrateFields(conf, pool, v); err != nil {
						errs <- err
					}
				}
			}
		}(done, vm, &wg)
	}
	for _, t := range vms {
		vm <- t
	}
	log.Println("В процессе миграции, произошло", len(errs), "ошибок")
	return nil
}

// Сами функции, используемые в мультиплексорах.
func migrateFields(conf models.Conf, pool client.Pool, vm mo.VirtualMachine) error {
	log.Println("Мигрируем поля", vm.Summary.Config.Name)
	annotationModified, pkeysFromAnnotation, expireFromAnnotation := rebuildAnnotation(vm.Summary.Config.Annotation)

	// Берем клиента из пула
	c, err := pool.GetClient(30 * time.Second)
	if err != nil {
		return err
	}
	defer pool.PutClient(c)

	pkeyOriginal := getCustomFieldByName(c.Node.Ctx, c.Node.Govmomi.Client, vm.Summary.CustomValue, conf.FieldProject)
	if pkeyOriginal == "Нет данных" {
		pkeyOriginal = ""
	}
	expireOriginal := getCustomFieldByName(c.Node.Ctx, c.Node.Govmomi.Client, vm.Summary.CustomValue, conf.FieldExpire)

	pkeyFinal := composeFieldProject(pkeyOriginal, pkeysFromAnnotation)
	expireFinal := composeFieldExpire(expireOriginal, expireFromAnnotation)

	if pkeyFinal != pkeyOriginal {
		err := setVMCustomField(c.Node.Ctx, c.Node.Govmomi.Client, vm.Config.Uuid, conf.FieldProject, pkeyFinal)
		if err != nil {
			return err
		}
	}

	if pkeyFinal == "" {
		expireFinal = ""
	}

	if expireFinal != expireOriginal {
		err := setVMCustomField(c.Node.Ctx, c.Node.Govmomi.Client, vm.Config.Uuid, conf.FieldExpire, expireFinal)
		if err != nil {
			return err
		}
	}

	//// Вычитываем кастомные поля заново и проверяем, что все сохранилось
	vmChek, err := getVMByKey(c.Node.Ctx, c.Node.Govmomi.Client, vm.Config.Uuid)
	if err != nil {
		return err
	}
	contrPkey := getCustomFieldByName(c.Node.Ctx, c.Node.Govmomi.Client, vmChek.Summary.CustomValue, conf.FieldProject)
	if contrPkey == "Нет данных" {
		contrPkey = ""
	}
	contrExpire := getCustomFieldByName(c.Node.Ctx, c.Node.Govmomi.Client, vmChek.Summary.CustomValue, conf.FieldExpire)

	if contrPkey != pkeyFinal || contrExpire != expireFinal {
		log.Println("Миграция полей виртуалки", vm.Summary.Config.Name, "прошла не корректно:")
		log.Println("--- pkeyExpected:", pkeyFinal, "-- pkeyGeted:", contrPkey)
		log.Println("--- dateExpected:", expireFinal, "-- dateGeted:", contrExpire)
		return err
	}

	log.Println("Для виртуалки", vm.Summary.Config.Name, "установлено поле проекта:", pkeyFinal, "и дата истечения:", expireFinal)
	err = setVMAnnotation(c.Node.Ctx, c.Node.Govmomi.Client, vm.Config.Uuid, annotationModified)
	if err != nil {
		return err
	}
	return nil
}

func composeFieldProject(pkeysOriginal string, pkeysFromAnnotation string) string {
	pkeysOriginal = strings.Replace(pkeysOriginal, " ", "", -1)
	var pkeysSlice []string
	pkeysSlice = append(pkeysSlice, strings.Split(pkeysOriginal, ",")...)
	pkeysSlice = append(pkeysSlice, strings.Split(pkeysFromAnnotation, ",")...)

	// Дедупликация и удаление пустых значений
	pkeysSlice = deduplicate(pkeysSlice)

	return strings.Trim(strings.Join(pkeysSlice, ","), ",")
}

func composeFieldExpire(expireOriginal string, expireFromAnnotation string) string {
	var dateFinal string
	parsedExpireFromAnnotation, err := time.Parse("02.01.2006", expireFromAnnotation)
	if expireFromAnnotation != "" && err != nil {
		expireFromAnnotation = ""
	}
	_, err = time.Parse("02.01.2006", expireOriginal)
	if expireOriginal != "" && err != nil {
		expireOriginal = ""
	}

	//	В кастоме есть дата
	//			Отдаем кастом
	//	В кастоме пусто
	//		В аннотации пусто или старая дата
	//			Отдаем сегодня+месяц
	//		В аннотации свежая дата
	//			Отдаем из аннотации

	if expireOriginal != "" {
		return expireOriginal
	}
	if expireFromAnnotation == "" || time.Since(parsedExpireFromAnnotation) > 0 {
		currentTime := time.Now().AddDate(0, 1, 0)
		dateFinal = currentTime.Format("02.01.2006")
	} else {
		dateFinal = expireFromAnnotation
	}
	return dateFinal
}

func deduplicate(splice []string) []string {
	var res []string
	for _, v := range splice {
		var m = true
		for _, j := range res {
			if v == j {
				m = false
				break
			}
		}
		if m && v != "" {
			res = append(res, v)
		}
	}
	return res
}

func rebuildAnnotation(oldNote string) (newNote string, pkeys string, expire string) {
	for _, annotationLine := range strings.Split(oldNote, "\n") {
		p := strings.Split(annotationLine, ":")
		if p[0] == "До" {
			d := strings.Trim(p[1], " ")
			_, err := time.Parse("02.01.2006", d)
			if err == nil {
				expire = d
			}
			continue
		}
		if p[0] == "Проект" {
			pkeys = strings.Trim(p[1], " ")
			pkeys = strings.Replace(pkeys, "/", ",", -1)
			continue
		}
		if annotationLine != "" {
			newNote = newNote + annotationLine + "\n"
		}
	}
	return newNote, pkeys, expire
}
