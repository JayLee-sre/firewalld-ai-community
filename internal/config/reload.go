package config

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

func Watch(path string, onChange func(*Config)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write != 0 {
					log.Println("config file changed, reloading...")
					cfg, err := Load(path)
					if err != nil {
						log.Printf("reload config failed: %v", err)
						continue
					}
					onChange(cfg)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("config watcher error: %v", err)
			}
		}
	}()

	return watcher.Add(path)
}
