package vmware

import (
	"context"
	"errors"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"log"
	"main/app"
	"strings"
	"sync"
	"time"
)

// Fan-ы
func FieldMigrator(app app.App) error {
	log.Println("Запущена миграция полей VMWare в ", app.Config.Threads, "воркерах")
	vms, err := getAllVMs(app)
	if err != nil {
		return err
	}

	vm, errs, wg, done := make(chan mo.VirtualMachine), make(chan error, len(vms)), sync.WaitGroup{}, make(chan interface{})
	defer func() {
		close(done)
		wg.Wait()
	}()
	wg.Add(app.Config.Threads)
	for i := 1; i <= app.Config.Threads; i++ {
		go func(done <-chan interface{}, vm chan mo.VirtualMachine, wg *sync.WaitGroup) {
			for {
				select {
				case <-done:
					wg.Done()
					return
				case v := <-vm:
					if err := migrateFields(app, v); err != nil {
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

// Сами функции, используемые в мультиплексорах
func migrateFields(app app.App, vm mo.VirtualMachine) error {
	var (
		annotationOriginal  string
		annotationModified  string
		dateFromAnnotation  string
		pkeysFromAnnotation string
		dateOriginal        string
		dateFinal           string
		pkeyOriginal        string
		pkeyFinal           string
	)
	log.Println("Мигрируем поля", vm.Summary.Config.Name)
	// Читаем Annotation в слайс
	// Перебираем слайс, выдираем наши поля и собираем слайс обратно в строку
	annotationOriginal = vm.Summary.Config.Annotation
	for _, annotationLine := range strings.Split(annotationOriginal, "\n") {
		p := strings.Split(annotationLine, ":")
		if p[0] == "До" {
			d := strings.Trim(p[1], " ")
			_, err := time.Parse("02.01.2006", d)
			if err == nil {
				dateFromAnnotation = d
				continue
			}
		}
		if p[0] == "Проект" {
			pkeysFromAnnotation = strings.Trim(p[1], " ")
			pkeysFromAnnotation = strings.Replace(pkeysFromAnnotation, "/", ",", -1)
			continue
		}
		annotationModified = annotationModified + annotationLine + "\n"
	}
	c, err := app.VmwareClient.GetClient(30 * time.Second)
	if err != nil {
		return err
	}
	defer app.VmwareClient.PutClient(c)

	// Проверяем поле URMSExpirationDate
	dateOriginal = getCustomFieldByName(c.Node.Ctx, c.Node.Govmomi.Client, vm.Summary.CustomValue, app.Config.FieldExpire)
	_, err = time.Parse("02.01.2006", dateOriginal)
	if err != nil {
		// Если не заплнено
		parsedDateFromAnnotation, _ := time.Parse("02.01.2006", dateFromAnnotation)
		if dateFromAnnotation != "" && time.Since(parsedDateFromAnnotation) < 0 {
			// Если есть дата в Annotation и она не в прошлом, ставим ее
			dateFinal = dateFromAnnotation
		} else {
			// Иначе, ставим Now+месяц
			currentTime := time.Now().AddDate(0, 1, 0)
			dateFinal = currentTime.Format("02.01.2006")
		}
	}
	// Если заполнена, ничего не делаем

	// Проверяем поле URMSProjectKey
	if pkeyOriginal = getCustomFieldByName(c.Node.Ctx, c.Node.Govmomi.Client, vm.Summary.CustomValue, app.Config.FieldProject); pkeyOriginal == "Нет данных" {
		pkeyOriginal = ""
	}
	srcPkeys := strings.Join([]string{strings.Replace(pkeyOriginal, " ", "", -1), strings.Replace(pkeysFromAnnotation, " ", "", -1)}, ",")

	pkeyArray := []string{}
	dbProjects, err := dbase.ListProjects(app.DB, "", false)
	if srcPkeys != "" {
		for _, p := range strings.Split(srcPkeys, ",") {
			addPkey := true
			for _, j := range pkeyArray {
				if p == j {
					addPkey = false
					continue
				}
			}
			if addPkey && dbProjects[p].Pkey != "" {
				pkeyArray = append(pkeyArray, p)
			}
		}
	}
	pkeyFinal = strings.Join(pkeyArray, ",")

	// Логика взаимосвязи полей
	if pkeyFinal == "" {
		dateFinal = ""
	}

	if dateFinal != dateOriginal {
		err = setVMCustomField(c.Node.Ctx, c.Node.Govmomi.Client, vm.ExtensibleManagedObject.Self.Value, app.Config.FieldExpire, dateFinal)
		if err != nil {
			return err
		}
		//		log.Println("Для виртуалки",vm.Summary.Config.Name, "установили URMSExpirationDate -", dateFinal)
	}
	if pkeyFinal != pkeyOriginal {
		err = setVMCustomField(c.Node.Ctx, c.Node.Govmomi.Client, vm.ExtensibleManagedObject.Self.Value, app.Config.FieldProject, pkeyFinal)
		if err != nil {
			return err
		}
		//		log.Println("Для виртуалки",vm.Summary.Config.Name, "установили URMSProjectKey -", pkeyFinal)
	}

	//// Вычитываем кастомные поля заново и проверяем, что все сохранилось
	vmChek, err := getVMByKey(c.Node.Ctx, c.Node.Govmomi.Client, vm.ExtensibleManagedObject.Self.Value)
	if err != nil {
		return err
	}
	contrDate := getCustomFieldByName(c.Node.Ctx, c.Node.Govmomi.Client, vmChek.Summary.CustomValue, app.Config.FieldExpire)
	contrPkey := getCustomFieldByName(c.Node.Ctx, c.Node.Govmomi.Client, vmChek.Summary.CustomValue, app.Config.FieldProject)

	if contrDate == dateFinal && contrPkey == pkeyFinal {
		// Записываем результирующее поле Annotation
		err = setVMAnnotation(c.Node.Ctx, c.Node.Govmomi.Client, vm.ExtensibleManagedObject.Self.Value, annotationModified)
		if err != nil {
			return err
		}
		//		log.Println("Для виртуалки",vm.Summary.Config.Name, "Обновили Annotation -", annotationModified)
		log.Println("Успешно мигрировали", vm.Summary.Config.Name)
	} else {
		log.Println("Миграция полей виртуалки", vm.Summary.Config.Name, "прошла не корректно:")
		log.Println("--- pkeyFinal:", pkeyFinal, "-- pkeyGeted:", contrPkey)
		log.Println("--- dateFinal:", dateFinal, "-- dateGeted:", contrDate)
		log.Println("Поле Annotation оставляем без изменений")
	}
	return nil
}

// Спец функции для использования в многопоточке (инициирующие сессию)
func getAllVMs(app app.App) ([]mo.VirtualMachine, error) {
	vms := []mo.VirtualMachine{}
	c, err := app.VmwareClient.GetClient(30 * time.Second)
	if err != nil {
		return vms, err
	}
	defer app.VmwareClient.PutClient(c)
	m := view.NewManager(c.Node.Govmomi.Client)
	v, err := m.CreateContainerView(c.Node.Ctx, c.Node.Govmomi.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return vms, err
	}
	defer v.Destroy(c.Node.Ctx)
	if err = v.Retrieve(c.Node.Ctx, []string{"VirtualMachine"}, []string{"summary"}, &vms); err != nil {
		return vms, err
	}
	return vms, nil
}

// Рядовые функции
func getVMByKey(ctx context.Context, c *vim25.Client, vmKey string) (mo.VirtualMachine, error) {
	virtualMachineView, err := view.NewManager(c).CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return mo.VirtualMachine{}, err
	}
	defer virtualMachineView.Destroy(ctx)
	var vms []mo.VirtualMachine
	if err = virtualMachineView.Retrieve(ctx, []string{"VirtualMachine"}, []string{"summary", "config", "datastore", "customValue"}, &vms); err != nil {
		return mo.VirtualMachine{}, err
	}
	for _, vm := range vms {
		if vm.ExtensibleManagedObject.Self.Value == vmKey {
			return vm, nil
		}
	}
	err = errors.New("Can't find VirtualMachine " + vmKey)
	return mo.VirtualMachine{}, err
}

func getCustomFieldByName(ctx context.Context, c *vim25.Client, customValues []types.BaseCustomFieldValue, customFieldName string) string {
	fieldKey, err := getCustomFieldKey(ctx, c, customFieldName)
	if err != nil {
		return "Error"
	}
	if customValues != nil {
		for _, customValue := range customValues {
			if customValue.GetCustomFieldValue().Key == fieldKey {
				return customValue.(*types.CustomFieldStringValue).Value
			}
		}
	}
	return "Нет данных"
}

func getCustomFieldKey(ctx context.Context, c *vim25.Client, customFieldName string) (int32, error) {
	customFieldsManager, err := object.GetCustomFieldsManager(c)
	if err != nil {
		return 0, err
	}
	fieldKey, err := customFieldsManager.FindKey(ctx, string(customFieldName))
	if err != nil {
		return 0, err
	}
	return fieldKey, nil
}

func setVMCustomField(ctx context.Context, c *vim25.Client, vmKey string, cField string, newCustomFieldValue string) error {
	vm, err := getVMByKey(ctx, c, vmKey)
	if err != nil {
		return err
	}
	cfKey, err := getCustomFieldKey(ctx, c, cField)
	if err != nil {
		return err
	}
	customFieldsManager, err := object.GetCustomFieldsManager(c)
	if err != nil {
		return err
	}
	err = customFieldsManager.Set(ctx, vm.ManagedEntity.Reference(), cfKey, newCustomFieldValue)
	if err != nil {
		return err
	}
	return nil
}

func setVMAnnotation(ctx context.Context, c *vim25.Client, vmKey string, newAnnotation string) error {
	vm, err := getVMByKey(ctx, c, vmKey)
	if err != nil {
		return err
	}
	vmManager := object.NewVirtualMachine(c, vm.Reference())
	task, err := vmManager.Reconfigure(ctx, types.VirtualMachineConfigSpec{Annotation: newAnnotation})
	if err != nil {
		return err
	}
	if err = task.Wait(ctx); err != nil {
		return err
	}
	return nil
}
