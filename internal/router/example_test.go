package router_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"github.com/xhrobj/go-metrics-and-alerts/internal/router"
	"github.com/xhrobj/go-metrics-and-alerts/internal/service"
	"go.uber.org/zap"
)

func newExampleRouter() http.Handler {
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo)
	h := handler.New(svc, nil)
	log := zap.NewNop()

	return router.New(h, log, router.Options{})
}

func ExampleNew_legacyEndpoints() {
	r := newExampleRouter()

	updateRq := httptest.NewRequest(http.MethodPost, "/update/gauge/temperature/5.11", nil)
	updateRs := httptest.NewRecorder()
	r.ServeHTTP(updateRs, updateRq)

	valueRq := httptest.NewRequest(http.MethodGet, "/value/gauge/temperature", nil)
	valueRs := httptest.NewRecorder()
	r.ServeHTTP(valueRs, valueRq)

	fmt.Println(updateRs.Code)
	fmt.Println(valueRs.Code)
	fmt.Println(valueRs.Body.String())

	// Output:
	// 200
	// 200
	// 5.11
}

func ExampleNew_jsonEndpoints() {
	r := newExampleRouter()

	updateBody := strings.NewReader(`{"id":"requests","type":"counter","delta":42}`)
	updateRq := httptest.NewRequest(http.MethodPost, "/update", updateBody)
	updateRq.Header.Set("Content-Type", "application/json")
	updateRs := httptest.NewRecorder()
	r.ServeHTTP(updateRs, updateRq)

	valueBody := strings.NewReader(`{"id":"requests","type":"counter"}`)
	valueRq := httptest.NewRequest(http.MethodPost, "/value", valueBody)
	valueRq.Header.Set("Content-Type", "application/json")
	valueRs := httptest.NewRecorder()
	r.ServeHTTP(valueRs, valueRq)

	fmt.Println(updateRs.Code)
	fmt.Print(updateRs.Body.String())
	fmt.Println(valueRs.Code)
	fmt.Print(valueRs.Body.String())

	// Output:
	// 200
	// {"id":"requests","type":"counter","delta":42}
	// 200
	// {"id":"requests","type":"counter","delta":42}
}

func ExampleNew_batchUpdateEndpoint() {
	r := newExampleRouter()

	updateBody := strings.NewReader(`[
		{"id":"alloc","type":"gauge","value":5.11},
		{"id":"poll_count","type":"counter","delta":42}
	]`)
	updateRq := httptest.NewRequest(http.MethodPost, "/updates", updateBody)
	updateRq.Header.Set("Content-Type", "application/json")
	updateRs := httptest.NewRecorder()
	r.ServeHTTP(updateRs, updateRq)

	gaugeRq := httptest.NewRequest(http.MethodGet, "/value/gauge/alloc", nil)
	gaugeRs := httptest.NewRecorder()
	r.ServeHTTP(gaugeRs, gaugeRq)

	counterRq := httptest.NewRequest(http.MethodGet, "/value/counter/poll_count", nil)
	counterRs := httptest.NewRecorder()
	r.ServeHTTP(counterRs, counterRq)

	fmt.Println(updateRs.Code)
	fmt.Println(gaugeRs.Body.String())
	fmt.Println(counterRs.Body.String())

	// Output:
	// 200
	// 5.11
	// 42
}
