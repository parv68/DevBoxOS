package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

// runStartWithWatch starts services, then watches for file changes.
func runStartWithWatch(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if err := runStart(cmd, args); err != nil {
		return err
	}

	fmt.Println("\n  Watching for file changes... (Ctrl+C to stop)")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create file watcher: %w", err)
	}
	defer watcher.Close()

	debounce := make(chan struct{}, 1)
	debounceTimer := time.Now()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
					if time.Since(debounceTimer) > 2*time.Second {
						debounceTimer = time.Now()
						select {
						case debounce <- struct{}{}:
						default:
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Fprintf(os.Stderr, "  Watch error: %v\n", err)
			}
		}
	}()

	watchDirs := getWatchDirs(dir)
	for _, watchDir := range watchDirs {
		if err := watcher.Add(watchDir); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not watch %s: %v\n", watchDir, err)
		} else {
			fmt.Printf("  Watching: %s\n", watchDir)
		}
	}

	for range debounce {
		fmt.Printf("\n  [%s] Change detected, restarting services...\n", time.Now().Format("15:04:05"))
	}

	return nil
}

func getWatchDirs(dir string) []string {
	var dirs []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() != ".git" && info.Name() != ".devbox" && info.Name() != "node_modules" {
			dirs = append(dirs, path)
		}
		return nil
	})
	return dirs
}
