// Package audit содержит механизм аудита успешной обработки метрик.
//
// Пакет реализует паттерн "Наблюдатель": Auditor рассылает Event всем
// подписанным Observer'ам.
package audit

import (
	"context"
	"errors"
)

// Event описывает событие аудита успешной обработки метрик Сервером.
type Event struct {
	// TS содержит Unix timestamp события.
	TS int64 `json:"ts"`

	// Metrics содержит имена успешно обработанных метрик.
	Metrics []string `json:"metrics"`

	// IPAddress содержит IP-адрес клиента, отправившего запрос.
	IPAddress string `json:"ip_address"`
}

// Observer описывает получателя событий аудита.
type Observer interface {
	// Notify обрабатывает событие аудита.
	//
	// Контекст позволяет прерывать долгие операции наблюдателя,
	// например HTTP-запрос к удалённому приёмнику аудита.
	Notify(ctx context.Context, event Event) error
}

// Auditor рассылает события аудита всем подписанным Observer-ам.
type Auditor struct {
	observers []Observer
}

// NewAuditor создаёт новый диспетчер событий аудита.
func NewAuditor() *Auditor {
	return &Auditor{}
}

// Subscribe добавляет Observer-а в список получателей событий аудита.
//
// Nil Observer игнорируется.
func (a *Auditor) Subscribe(observer Observer) {
	if observer == nil {
		return
	}

	a.observers = append(a.observers, observer)
}

// Notify отправляет событие аудита всем подписанным Observer-ам.
//
// Если один или несколько Observer-ов вернули ошибку, Notify всё равно
// продолжает рассылку остальным Observer-ам и возвращает объединённую ошибку.
func (a *Auditor) Notify(ctx context.Context, event Event) error {
	var errs []error

	for _, observer := range a.observers {
		if err := observer.Notify(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
