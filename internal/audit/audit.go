package audit

import (
	"context"
	"errors"
)

type Event struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

type Observer interface {
	// NOTE: добавим context для сетевого запроса, чтобы отключать по таймауту
	Notify(ctx context.Context, event Event) error
}

type Auditor struct {
	observers []Observer
}

func NewAuditor() *Auditor {
	return &Auditor{}
}

func (a *Auditor) Subscribe(observer Observer) {
	if observer == nil {
		return
	}

	a.observers = append(a.observers, observer)
}

func (a *Auditor) Notify(ctx context.Context, event Event) error {
	var errs []error

	for _, observer := range a.observers {
		if err := observer.Notify(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
