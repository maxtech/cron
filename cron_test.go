package cron

import (
	"log"
	"sync"
	"testing"
	"time"
)

func printFunc()  {
	log.Println("test")
}

func panicFunc()  {
	panic("test panic")
}

func TestNew(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)

	cron := New()
	cron.AddFunc(time.Second * 3, printFunc)
	//cron.AddFunc(time.Second * 5, panicFunc)
	cron.Start()

	wg.Wait()
}
