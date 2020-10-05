package client

import (
	"context"
	"fmt"
	"main/models"
	"net/url"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/vim25/soap"
)

type Client struct {
	Govmomi *govmomi.Client
	Ctx     context.Context
	Cancel  context.CancelFunc
}

func getClient(conf models.Conf) (Client, error) {
	var c Client
	var err error
	c.Ctx, c.Cancel = context.WithCancel(context.Background())
	c.Govmomi, err = newClient(c.Ctx, conf)
	if err != nil {
		return Client{}, fmt.Errorf("ошибка создания клиента: %w", err)
	}
	// SessionIsActive выполняет запрос к VMware, а не проверяет состояние объектов govmomi.
	sessionIsActive, errS := c.Govmomi.SessionManager.SessionIsActive(c.Ctx)
	// проверим аутентификацию.
	userSession, errU := c.Govmomi.SessionManager.UserSession(c.Ctx)
	if !sessionIsActive || userSession == nil || errS != nil || errU != nil {
		// если инстанст есть, но с ним что-то не так, убиваем и пересоздаём.
		_ = c.Govmomi.Logout(c.Ctx)
		c.Cancel()
		c.Govmomi, err = newClient(c.Ctx, conf)
	}
	if c.Govmomi == nil {
		return Client{}, fmt.Errorf("ошибка создания клиента: %w", err)
	}
	return c, err
}

// NewClient creates a govmomi.Client for use in the examples.
func newClient(ctx context.Context, conf models.Conf) (*govmomi.Client, error) {
	// Parse URL from string
	u, err := soap.ParseURL(conf.Server)
	if err != nil {
		return nil, err
	}

	// Override username and/or password as required
	processOverride(conf, u)

	// Connect and log in to ESX or vCenter
	user := u.User
	u.User = nil
	c, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		return nil, err
	}
	c.RoundTripper = session.KeepAlive(c.RoundTripper, time.Minute*10)
	err = c.Login(ctx, user)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func processOverride(conf models.Conf, u *url.URL) {
	// Override username if provided
	var password string
	var ok bool
	var username string
	if u.User != nil {
		password, ok = u.User.Password()
	}
	if ok {
		u.User = url.UserPassword(conf.Domain+"\\"+conf.Login, password)
	} else {
		u.User = url.User(conf.Domain + "\\" + conf.Login)
	}
	// Override password if provided
	if u.User != nil {
		username = u.User.Username()
	}
	u.User = url.UserPassword(username, conf.Pass)
}
