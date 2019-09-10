package trigger

import (
	"time"

	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
)

type CronScheduler interface {
	Submit(task func())
	Start()
}

type cronScheduler struct {
	passport chan struct{}
	cronStr  string
	cron     *cron.Cron
}

func NewCronScheduler(cronStr string) CronScheduler {
	passport := make(chan struct{}, 1)
	passport <- struct{}{}
	return &cronScheduler{
		passport: passport,
		cronStr:  cronStr,
		cron:     cron.New(),
	}
}

func (s *cronScheduler) Submit(task func()) {
	s.cron.AddFunc(s.cronStr, func() {
		logrus.Info("Attempt to perform cleanup task")
		var granted bool
		select {
		case <-s.passport:
			granted = true
		case <-time.After(time.Second):
			granted = false
		}

		if !granted {
			logrus.Info("Another cleanup task is still running, skip")
			return
		}

		defer func() {
			s.passport <- struct{}{}
		}()

		logrus.Info("Start to run cleanup task")
		task()
	})
}

func (s *cronScheduler) Start() {
	s.cron.Start()
}
