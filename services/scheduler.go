package services

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"data-exchange/repository"

	"github.com/robfig/cron/v3"
)

// Scheduler 定时调度器，承载 cron 实例与任务入口映射
type Scheduler struct {
	CronScheduler *cron.Cron
	TaskEntryMap  map[int64]cron.EntryID
	taskRepo      *repository.SQLTaskRepo
	mu            sync.Mutex
	initOnce      sync.Once
}

// NewScheduler 构建调度器
func NewScheduler(taskRepo *repository.SQLTaskRepo) *Scheduler {
	return &Scheduler{
		TaskEntryMap: make(map[int64]cron.EntryID),
		taskRepo:     taskRepo,
	}
}

// Init 启动 cron 并加载全部启用任务（复用 App 的并发工作池）
func (s *Scheduler) Init(app *App) {
	s.initOnce.Do(func() {
		// 使用标准 5 段 cron（分 时 日 月 周），与任务表单默认表达式一致
		s.CronScheduler = cron.New()
		s.CronScheduler.Start()
		log.Println("[调度器] Cron调度器已启动")
		SetTaskExecutor(app.Executor)
		s.LoadAllTasks()
	})
}

// LoadAllTasks 重新加载全部启用且配置了 cron 的任务
func (s *Scheduler) LoadAllTasks() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, entryID := range s.TaskEntryMap {
		s.CronScheduler.Remove(entryID)
	}
	s.TaskEntryMap = make(map[int64]cron.EntryID)

	tasks, err := s.taskRepo.LoadAllEnabled()
	if err != nil {
		log.Printf("[调度器] 加载任务失败: %v", err)
		return
	}

	count := 0
	for _, t := range tasks {
		if t.Vendor == nil || t.Vendor.Enabled == 0 {
			continue
		}
		taskID := t.ID
		cronExpr := t.CronExpression
		entryID, err := s.CronScheduler.AddFunc(cronExpr, func() {
			log.Printf("[调度器] 触发定时任务 #%d，提交到工作池", taskID)
			pool := GetGlobalPool()
			pool.Submit(taskID)
		})
		if err != nil {
			log.Printf("[调度器] 添加任务 #%d 失败 (cron: %s): %v", t.ID, cronExpr, err)
			continue
		}
		s.TaskEntryMap[t.ID] = entryID
		count++
	}
	log.Printf("[调度器] 已加载 %d 个定时任务", count)
}

// AddTask 注册单个任务的 cron 触发
func (s *Scheduler) AddTask(taskID int64, cronExpr string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, ok := s.TaskEntryMap[taskID]; ok {
		s.CronScheduler.Remove(entryID)
		delete(s.TaskEntryMap, taskID)
	}

	tid := taskID
	entryID, err := s.CronScheduler.AddFunc(cronExpr, func() {
		log.Printf("[调度器] 触发定时任务 #%d，提交到工作池", tid)
		pool := GetGlobalPool()
		pool.Submit(tid)
	})
	if err != nil {
		log.Printf("[调度器] 添加任务 #%d 失败: %v", taskID, err)
		return
	}
	s.TaskEntryMap[taskID] = entryID
	log.Printf("[调度器] 任务 #%d 已添加 (cron: %s)", taskID, cronExpr)
}

// RemoveTask 移除单个任务的 cron 触发
func (s *Scheduler) RemoveTask(taskID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, ok := s.TaskEntryMap[taskID]; ok {
		s.CronScheduler.Remove(entryID)
		delete(s.TaskEntryMap, taskID)
		log.Printf("[调度器] 任务 #%d 已移除", taskID)
	}
}

// Stop 停止调度器与底层资源
func (s *Scheduler) Stop() {
	if s.CronScheduler != nil {
		s.CronScheduler.Stop()
		log.Println("[调度器] Cron调度器已停止")
	}
	StopWorkerPool()
	CloseAllCachedDBs()
}

// ==================== App 调度封装（结构体方法，替代原包级函数） ====================

// InitScheduler 初始化并启动调度器
func (a *App) InitScheduler() {
	a.Scheduler.Init(a)
}

// StopScheduler 停止调度器
func (a *App) StopScheduler() {
	a.Scheduler.Stop()
}

// AddTaskToScheduler 注册任务到调度器
func (a *App) AddTaskToScheduler(taskID int64, cronExpr string) {
	a.Scheduler.AddTask(taskID, cronExpr)
}

// RemoveTaskFromScheduler 从调度器移除任务
func (a *App) RemoveTaskFromScheduler(taskID int64) {
	a.Scheduler.RemoveTask(taskID)
}

// NextRunTimes 计算 cron 表达式未来 n 次执行时间（兼容标准 5 段与含秒 6 段格式）
// 返回格式：2006-01-02 15:04:05
func NextRunTimes(expr string, n int) ([]string, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("cron 表达式为空")
	}
	var sched cron.Schedule
	var err error
	// 优先按标准 5 段（分 时 日 月 周）解析
	if sched, err = cron.ParseStandard(expr); err != nil {
		// 回退到 6 段（含秒）格式
		p := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		sched, err = p.Parse(expr)
		if err != nil {
			return nil, err
		}
	}
	times := make([]string, 0, n)
	t := time.Now()
	for i := 0; i < n; i++ {
		t = sched.Next(t)
		times = append(times, t.Format("2006-01-02 15:04:05"))
	}
	return times, nil
}

// computeNextRun 返回 cron 表达式的「下一次执行时间」字符串，无效或空表达式返回空串
func computeNextRun(expr string) string {
	times, err := NextRunTimes(expr, 1)
	if err != nil || len(times) == 0 {
		return ""
	}
	return times[0]
}
