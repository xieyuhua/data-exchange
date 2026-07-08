package services

import (
	"data-exchange/models"
	"log"
	"sync"

	"github.com/robfig/cron/v3"
)

var (
	CronScheduler *cron.Cron
	TaskEntryMap  = make(map[int64]cron.EntryID)
	mu            sync.Mutex
	initOnce      sync.Once
)

func InitScheduler() {
	initOnce.Do(func() {
		CronScheduler = cron.New(cron.WithSeconds())
		CronScheduler.Start()
		log.Println("[调度器] Cron调度器已启动")
		InitWorkerPool() // 初始化并发工作池
		LoadAllTasks()
	})
}

func LoadAllTasks() {
	mu.Lock()
	defer mu.Unlock()

	for _, entryID := range TaskEntryMap {
		CronScheduler.Remove(entryID)
	}
	TaskEntryMap = make(map[int64]cron.EntryID)

	var tasks []models.SQLTask
	models.DB.Preload("Vendor").
		Where("enabled = 1 AND cron_expression != ''").
		Find(&tasks)

	count := 0
	for _, t := range tasks {
		if t.Vendor == nil || t.Vendor.Enabled == 0 {
			continue
		}
		taskID := t.ID
		cronExpr := t.CronExpression
		entryID, err := CronScheduler.AddFunc(cronExpr, func() {
			log.Printf("[调度器] 触发定时任务 #%d，提交到工作池", taskID)
			// 通过 worker pool 执行，自动限流
			pool := GetGlobalPool()
			pool.Submit(taskID)
		})
		if err != nil {
			log.Printf("[调度器] 添加任务 #%d 失败 (cron: %s): %v", t.ID, cronExpr, err)
			continue
		}
		TaskEntryMap[t.ID] = entryID
		count++
	}
	log.Printf("[调度器] 已加载 %d 个定时任务", count)
}

func AddTaskToScheduler(taskID int64, cronExpr string) {
	mu.Lock()
	defer mu.Unlock()

	if entryID, ok := TaskEntryMap[taskID]; ok {
		CronScheduler.Remove(entryID)
		delete(TaskEntryMap, taskID)
	}

	tid := taskID
	entryID, err := CronScheduler.AddFunc(cronExpr, func() {
		log.Printf("[调度器] 触发定时任务 #%d，提交到工作池", tid)
		pool := GetGlobalPool()
		pool.Submit(tid)
	})
	if err != nil {
		log.Printf("[调度器] 添加任务 #%d 失败: %v", taskID, err)
		return
	}
	TaskEntryMap[taskID] = entryID
	log.Printf("[调度器] 任务 #%d 已添加 (cron: %s)", taskID, cronExpr)
}

func RemoveTaskFromScheduler(taskID int64) {
	mu.Lock()
	defer mu.Unlock()

	if entryID, ok := TaskEntryMap[taskID]; ok {
		CronScheduler.Remove(entryID)
		delete(TaskEntryMap, taskID)
		log.Printf("[调度器] 任务 #%d 已移除", taskID)
	}
}

func StopScheduler() {
	if CronScheduler != nil {
		CronScheduler.Stop()
		log.Println("[调度器] Cron调度器已停止")
	}
	StopWorkerPool()
	CloseAllCachedDBs()
}
