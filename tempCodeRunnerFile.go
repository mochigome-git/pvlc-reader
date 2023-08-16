func fsnotifyStart(counterCallback func(string)) {
	log.Println("監視開始", dirname)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("error creating watcher:", err)
		return
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Println("event:", event)
				switch {
				case event.Op&fsnotify.Write == fsnotify.Write:
					contents, _ := getContents()

					sig, err := thd.Insert(host, port, user, password, dbname, pcName, contents)
					if err != nil {
						log.Println("error inserting data:", err)
					} else {
						counterCallback(sig) // Call the completion callback
					}

				case event.Op&fsnotify.Create == fsnotify.Create:
					contents, _ := getContents()

					sig, err := thd.Insert(host, port, user, password, dbname, pcName, contents)
					if err != nil {
						log.Println("error inserting data:", err)
					} else {
						counterCallback(sig) // Call the completion callback
					}
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