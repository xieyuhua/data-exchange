package services

import (
	"log"
	"sync"

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
		s.CronScheduler = cron.New(cron.WithSeconds())
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
