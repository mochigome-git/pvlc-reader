package main

import (
	"embed"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/joho/godotenv"
	"github.com/zserge/lorca"

	thd "testcode/pkg"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	PCName     string
	DirName    string
}

//go:embed www
var fs embed.FS

// Go types that are bound to the UI must be thread-safe because each binding
// is executed in its own goroutine. In this simple case, we may use atomic
// operations, but for more complex cases one should use proper synchronization.
type counter struct {
	sync.Mutex
	count int
}

var (
	config          *Config
	fsnotifyStarted bool
	contentMutex    sync.Mutex
	contentReady    bool
	content         string
	contentCond     *sync.Cond
	logPathMutex    sync.Mutex
	logPath         string
	initialStartup  bool
)

type ContentManager struct {
	sync.Mutex
	contents      string
	contentsReady bool
}

var (
	contentCh = make(chan string)
	logPathCh = make(chan string)
)

func main() {
	// Load the configuration from the environment file
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("error loading configuration from .env file:", err)
	}

	config = &Config{
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		PCName:     os.Getenv("PC_NAME"),
		DirName:    os.Getenv("DIR_NAME"),
	}

	go logWatcherLoop()

	args := []string{}
	if runtime.GOOS == "linux" {
		args = append(args, "--class=Lorca")
	}
	ui, err := lorca.New("", "", 780, 620, "--remote-allow-origins=*")
	if err != nil {
		log.Println("runtime error:", err)
	}
	defer ui.Close()

	ui.Bind("start", func() {
		log.Println("UI is ready")
	})

	c := &counter{}
	var jobFull string

	ui.Bind("saveJobGo", func(jobOrder string, jobMonth string) {
		jobFull = jobMonth + jobOrder
	})
	ui.Bind("counterAdd", c.Add)
	ui.Bind("counterValue", c.Count)
	ui.Bind("saveConfigGo", thd.SetConfig)
	ui.Bind("loadConfigGo", thd.LoadConfigGo)
	ui.Bind("resetCounter", c.Reset)
	ui.Bind("NotifyStart", func() {
		if jobFull != "" {
			fsnotifyStart(jobFull, func(sig string) {
				ui.Eval(`window.NotifyStartComplete("` + sig + `")`)
			})
			fsnotifyStarted = true
		}
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Println("listen tcp error:", err)
	}
	defer ln.Close()
	go http.Serve(ln, http.FileServer(http.FS(fs)))
	ui.Load(fmt.Sprintf("http://%s/www", ln.Addr()))

	sigc := make(chan os.Signal)
	signal.Notify(sigc, os.Interrupt)
	go func() {
		<-sigc
		log.Println("exiting...")
		os.Exit(0)
	}()

	select {
	case <-sigc:
	case <-ui.Done():
		log.Println("exiting...")
		os.Exit(0)
	}
	log.Println("exiting...")
	os.Exit(0)
}

func setContents(value string) {
	contentCh <- value
}

func waitForContentsReady() (string, bool) {
	contents := <-contentCh
	return contents, true
}

func getContents() (string, bool) {
	contentMutex.Lock()
	defer contentMutex.Unlock()
	return content, contentReady
}

func setLogPath(value string) {
	logPathMutex.Lock()
	defer logPathMutex.Unlock()
	logPath = value
}

func getLogPath() (string, bool) {
	logPathMutex.Lock()
	defer logPathMutex.Unlock()
	return logPath, logPath != ""
}

func deleteLogFileContents(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString("")
	if err != nil {
		return err
	}

	return nil
}

func pickLog(filePattern string, fileInfo thd.LogFileInfo) (string, error) {
	filePath := fileInfo.Path
	fileName := fileInfo.Name
	findStr := "read: "

	// Create a channel to receive the log contents
	logContentCh := make(chan string)
	errCh := make(chan error)

	// Start a goroutine to read the log contents
	go func() {
		time.Sleep(500 * time.Millisecond)
		contents, err := thd.GetLog(fileName, findStr, filePath)
		if err != nil {
			errCh <- fmt.Errorf("error getting log: %w", err)
			return
		}
		logContentCh <- contents

	}()

	// Wait for the log contents or an error
	select {
	case contents := <-logContentCh:
		//fmt.Println(contents)
		return contents, nil
	case err := <-errCh:
		return "", err
	}
}

func logWatcherLoop() {

	fileInfoCh := make(chan thd.LogFileInfo)
	filePattern := time.Now().Format("2006_01_02") + "*.log"

	go thd.StartLogWatcher(filePattern, fileInfoCh)

	// Loop continuously to process log files
	go func() {
		for {
			fileInfo := <-fileInfoCh
			logContent, err := pickLog(filePattern, fileInfo)
			if err != nil {
				log.Println("error getting log:", err)
				continue
			}
			logPathMutex.Lock()
			logPath = fileInfo.Path + fileInfo.Name
			logPathMutex.Unlock()
			setContents(logContent)
		}
	}()
}

func fsnotifyStart(jobFull string, counterCallback func(string)) {
	dirname := config.DirName
	joborder := jobFull
	log.Println("監視開始", dirname)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("error creating watcher:", err)
		return
	}
	defer watcher.Close()

	done := make(chan bool)
	fmt.Println(joborder)
	// Start a separate goroutine to handle the thd.Insert operation
	go func() {
		if !initialStartup {
			for {
				contents := <-contentCh
				if initialStartup && !strings.Contains(contents, "FAILED") && !strings.Contains(contents, "Init Programming") && strings.Contains(contents, "verified") {
					sig, err := thd.Insert(config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName, config.PCName, joborder, contents)
					if err != nil {
						log.Println("error inserting data:", err)
					}
					counterCallback(sig) // Call the completion callback
					setContents("")
					joborder = ""
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Println("event:", event)
				switch {
				case event.Op&fsnotify.Write == fsnotify.Write:
					initialStartup = true
					// Get the latest contents
					contents, contentsReady := getContents()
					if !contentsReady {
						// Contents not ready, wait until it becomes available
						contents, contentsReady = waitForContentsReady()
					}

					fmt.Println(contents)

					// Update the contents using the setter function
					setContents(contents)

				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
				done <- true
			}
		}
	}()

	err = watcher.Add(dirname)
	if err != nil {
		log.Println("error adding watcher:", err)
		return
	}
	<-done
}

func (c *counter) Add(n int) {
	c.Lock()
	defer c.Unlock()
	// If the insert is successful, increment the count
	c.count += n
}

func (c *counter) Count() int {
	c.Lock()
	defer c.Unlock()
	return c.count
}

func (c *counter) Reset() {
	c.Lock()
	defer c.Unlock()
	c.count = 0
}
