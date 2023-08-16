package thd

import (
	"log"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

const (
	filePath = "C:/Users/Public/GCS/Logging/"
	findStr  = "read: "
)

type LogFileInfo struct {
	Pattern string
	Path    string
	Name    string
}

func getLatestLogFile(filePattern string) string {
	files, err := filepath.Glob(filepath.Join(filePath, filePattern))
	if err != nil || len(files) == 0 {
		return ""
	}

	latestLogFile := files[0]
	for _, file := range files {
		if file > latestLogFile {
			latestLogFile = file
		}
	}

	return latestLogFile
}

func watchLogFile(filePattern string, fileInfoCh chan<- LogFileInfo) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("watcher error:", err)
	}
	defer watcher.Close()

	done := make(chan bool)

	// Start a goroutine to handle events from the watcher
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Check if the log file was modified
				if event.Op&fsnotify.Write == fsnotify.Write {
					//log.Println("Log file modified:", event.Name)

					// Retrieve the latest log file information
					latestLogFile := getLatestLogFile(filePattern)
					fileInfo := LogFileInfo{
						Pattern: filePattern,
						Path:    filePath,
						Name:    latestLogFile[len(latestLogFile)-14:],
					}
					fileInfoCh <- fileInfo
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					//log.Println("Log file created:", event.Name)

					// Retrieve the latest log file information
					latestLogFile := getLatestLogFile(filePattern)
					fileInfo := LogFileInfo{
						Pattern: filePattern,
						Path:    filePath,
						Name:    latestLogFile[len(latestLogFile)-14:],
					}
					fileInfoCh <- fileInfo
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Error:", err)
			}
		}
	}()

	// Add the log file or directory to the watcher
	err = watcher.Add(filePath)
	if err != nil {
		log.Println("watcher error:", err)
	}

	<-done
}

func StartLogWatcher(filePattern string, fileInfoCh chan<- LogFileInfo) {
	latestLogFile := getLatestLogFile(filePattern)

	if latestLogFile == "" {
		log.Println("No log files found matching the pattern:", filePattern)

		// Get the file pattern for the latest log file
		latestFilePattern := "*.log"
		latestLogFile = getLatestLogFile(latestFilePattern)
		if latestLogFile == "" {
			log.Println("No log files found.")
		} else {
			log.Println("Latest log file found:", latestLogFile)
		}
	} else {
		log.Println("Today's log file found:", latestLogFile)
	}

	// Send the log file information to the channel
	fileInfo := LogFileInfo{
		Pattern: filePattern,
		Path:    filePath,
		Name:    latestLogFile[len(latestLogFile)-14:],
	}
	fileInfoCh <- fileInfo

	// Start watching the log file for changes
	go watchLogFile(filePattern, fileInfoCh)

}
