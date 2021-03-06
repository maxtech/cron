package cron

import (
    "log"
    "sync"
    "testing"
    "time"
)

func printFunc() {
    log.Println("test")
}

func panicFunc() {
    panic("test panic")
}

func TestNew(t *testing.T) {
    wg := sync.WaitGroup{}
    wg.Add(1)

    cron := New()
    cron.AddDayFunc(23, false, time.Second * 3, printFunc)
    //cron.AddDurationFunc(time.Second * 5, panicFunc)
    cron.Start()

    time.Sleep(time.Minute * 20)
    cron.Stop()
}
