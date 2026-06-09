package repository

import "errors"

// ErrMetricNotFound возвращается, когда метрика не найдена в хранилище.
var ErrMetricNotFound = errors.New("metric not found")
