package handler_test

import (
	"testing"

	handlermocks "github.com/xhrobj/go-metrics-and-alerts/internal/server/transport/http/handler/mocks"
	"go.uber.org/mock/gomock"
)

func newMockService(t *testing.T) *handlermocks.MockService {
	t.Helper()

	return handlermocks.NewMockService(gomock.NewController(t))
}
