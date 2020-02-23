package cron

import (
    "errors"
    "fmt"
    uuid "github.com/satori/go.uuid"
    "io"
    "os"
    "os/signal"
    "runtime"
    "runtime/debug"
    "sync"
    "syscall"
    "time"
)

type entry struct {
    id         string
    checkDay   int
    oneDayLoop bool
    checkPoint time.Time
    duration   time.Duration
    entryFunc  func()
    lock       sync.Mutex
}

func (e *entry) ableRun() bool {
    nowTime := time.Now()
    if e.checkDay > 0 && nowTime.Day() == e.checkDay {
        if e.oneDayLoop {
            return nowTime.After(e.checkPoint.Add(e.duration))
        } else {
            targetTime := time.Date(nowTime.Year(), nowTime.Month(), nowTime.Day(), 0, 0, 0, 0, nowTime.Location())
            targetTime = targetTime.Add(e.duration)
            return  nowTime.After(targetTime) && e.checkPoint.Year() != targetTime.Year() && e.checkPoint.Month() != targetTime.Month()
        }

    }
    return nowTime.After(e.checkPoint.Add(e.duration))
}

type Cron struct {
    baseDir string
    entries []*entry
    lock    sync.RWMutex
    isRun   bool
    ch      chan *entry
    writer  io.Writer
}

func New() *Cron {
    return &Cron{
        baseDir: "",
        isRun:   false,
        ch:      make(chan *entry),
        writer:  os.Stderr,
    }
}

func NewWithWriter(writer io.Writer) *Cron {
    return &Cron{
        baseDir: "",
        isRun:   false,
        ch:      make(chan *entry),
        writer:  writer,
    }
}

func (c *Cron) AddDurationFunc(duration time.Duration, cmd func()) {
    entry := &entry{
        id:         uuid.NewV4().String(),
        checkPoint: time.Now(),
        duration:   duration,
        entryFunc:  cmd,
    }
    c.entries = append(c.entries, entry)
}

func (c *Cron) AddDayFunc(monthDay int, oneDayLoop bool, duration time.Duration, cmd func()) error {
    if monthDay > 31 && monthDay < 0 {
        return errors.New("month day error: only 0 ~ 31")
    }

    date := time.Unix(0, 0).In(time.UTC)
    date = date.AddDate(0, 0, monthDay-1)

    entry := &entry{
        id:         uuid.NewV4().String(),
        checkPoint: time.Unix(0, 0),
        oneDayLoop: oneDayLoop,
        checkDay:   date.Day(),
        duration:   duration,
        entryFunc:  cmd,
    }
    c.entries = append(c.entries, entry)
    return nil
}

func (c *Cron) Start() {
    go c.watcher()
    go c.worker()
    c.lock.RLock()
    defer c.lock.RUnlock()
    c.isRun = true
    go c.workChecker()
}

func (c *Cron) workChecker() {
    for {
        for _, job := range c.entries {
            if !c.isRun {
                return
            }
            if c.isRun && job.ableRun() {
                job.lock.Lock()
                job.checkPoint = time.Now()
                job.lock.Unlock()
                c.ch <- job
            }
            runtime.Gosched()
        }
    }
}

func (c *Cron) worker() {
    defer func() {
        if err := recover(); err != nil {
            _, _ = fmt.Fprintln(c.writer, time.Now(), "[ERROR] cron worker error:", err)
            _, _ = fmt.Fprintln(c.writer, string(debug.Stack()))
            go c.worker()
        }
    }()

    for {
        select {
        case entry := <-c.ch:
            if !c.isRun {
                return
            }
            entry.entryFunc()
        }
    }
}

func (c *Cron) Stop() {
    c.lock.RLock()
    defer c.lock.RUnlock()
    c.isRun = false
    close(c.ch)
}

func (c *Cron) watcher() {
    signalCh := make(chan os.Signal, 1)
    signal.Notify(signalCh, os.Interrupt, os.Kill, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
    _ = <-signalCh
    c.Stop()
}
