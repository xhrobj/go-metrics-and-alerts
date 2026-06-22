package agent

import "context"

func (a *Agent) report() {
	report, ok, err := a.service.PreparePeriodicReport(context.Background())
	if err != nil || !ok {
		return
	}

	select {
	case a.sendQueue <- report:
	default:
		a.service.Restore(report)
		a.log.Warn("sendQueue is full")
	}
}
