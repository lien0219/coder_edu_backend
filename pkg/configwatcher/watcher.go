package configwatcher

import (
	"coder_edu_backend/internal/config"
	"coder_edu_backend/pkg/logger"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

type ConfigReloader func(cfg interface{})

func WatchConfig(configPath string, cfg interface{}, reloader ConfigReloader) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Failed to create config watcher:", err)
	}
	defer watcher.Close()

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		log.Fatal("Failed to get absolute path:", err)
	}

	if err := watcher.Add(absPath); err != nil {
		log.Fatal("Failed to watch config file:", err)
	}

	var mu sync.Mutex
	timer := time.NewTimer(0)
	<-timer.C

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				// 防抖处理
				mu.Lock()
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(1 * time.Second)
				mu.Unlock()
			}
		case <-timer.C:
			// 重新加载配置
			dirPath := filepath.Dir(configPath)
			newCfg, err := config.LoadConfig(dirPath)
			if err != nil {
				logger.Log.Error("Failed to reload config", zap.Error(err))
				continue
			}
			reloader(newCfg)
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Log.Error("Config watcher error", zap.Error(err))
		}
	}
}
