package vmware

import (
	"context"
	"errors"
	"main/vmware/client"
	"time"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

const empty = "Нет данных"

// Спец функции для использования в многопоточке (инициирующие сессию).
func getAllVMs(pool client.Pool) ([]mo.VirtualMachine, error) {
	vms := []mo.VirtualMachine{}
	c, err := pool.GetClient(30 * time.Second)
	if err != nil {
		return vms, err
	}
	defer pool.PutClient(c)
	m := view.NewManager(c.Node.Govmomi.Client)
	v, err := m.CreateContainerView(c.Node.Ctx, c.Node.Govmomi.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return vms, err
	}
	defer v.Destroy(c.Node.Ctx)
	if err = v.Retrieve(c.Node.Ctx, []string{"VirtualMachine"}, []string{"summary", "config"}, &vms); err != nil {
		return vms, err
	}
	return vms, nil
}

// Рядовые функции.
func getVMByKey(ctx context.Context, c *vim25.Client, vmKey string) (mo.VirtualMachine, error) {
	virtualMachineView, err := view.NewManager(c).CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return mo.VirtualMachine{}, err
	}
	defer virtualMachineView.Destroy(ctx)
	var vms []mo.VirtualMachine

	err = virtualMachineView.RetrieveWithFilter(ctx, []string{"VirtualMachine"}, []string{"summary", "config", "customValue"}, &vms, property.Filter{"config.uuid": vmKey})
	if err != nil {
		return mo.VirtualMachine{}, err
	}
	if len(vms) < 1 {
		return mo.VirtualMachine{}, errors.New("Can't find VirtualMachine with UUID" + vmKey)
	}

	return vms[0], err
}

func getCustomFieldByName(ctx context.Context, c *vim25.Client, customValues []types.BaseCustomFieldValue, customFieldName string) string {
	fieldKey, err := getCustomFieldKey(ctx, c, customFieldName)
	if err != nil {
		return "Error"
	}
	if len(customValues) > 0 {
		for _, customValue := range customValues {
			if customValue.GetCustomFieldValue().Key == fieldKey {
				return customValue.(*types.CustomFieldStringValue).Value
			}
		}
	}
	return empty
}

func getCustomFieldKey(ctx context.Context, c *vim25.Client, customFieldName string) (int32, error) {
	customFieldsManager, err := object.GetCustomFieldsManager(c)
	if err != nil {
		return 0, err
	}
	fieldKey, err := customFieldsManager.FindKey(ctx, customFieldName)
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
