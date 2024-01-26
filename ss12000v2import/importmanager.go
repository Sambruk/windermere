package ss12000v2import

// The ImportManager keeps track of the import runners (ImportRunner)
// It's basically just a container for ImportRunner which allows adding
// and deleting runners and controlled shutdown.
type ImportManager struct {
	quit   chan int
	add    chan RunnerConfig
	delete chan deleteCommand
}

type deleteCommand struct {
	tenant string
	reply  chan int
}

func NewImportManager() *ImportManager {
	var im ImportManager
	im.quit = make(chan int)
	im.add = make(chan RunnerConfig)
	im.delete = make(chan deleteCommand)
	go importManager(&im)
	return &im
}

// Stops the ImportManager (blocks until all runners have stopped)
func (im *ImportManager) Quit() {
	im.quit <- 0
	<-im.quit
}

func (im *ImportManager) AddRunner(conf RunnerConfig) {
	im.add <- conf
}

// DeleteRunner blocks until the runner is stopped and deleted.
// The reason it blocks is because we have to know for sure the
// runner isn't running an import when we remove the configuration
// from persistence (since the runner might want to record history
// while it's running).
func (im *ImportManager) DeleteRunner(tenant string) {
	command := deleteCommand{
		tenant: tenant,
		reply:  make(chan int),
	}
	im.delete <- command
	<-command.reply
}

func importManager(im *ImportManager) {
	runners := make(map[string]*ImportRunner)
	for {
		select {
		case conf := <-im.add:
			if oldRunner, ok := runners[conf.Tenant]; ok {
				oldRunner.Quit()
			}
			runner := NewImportRunner(conf)
			runners[conf.Tenant] = runner
		case delCmd := <-im.delete:
			if oldRunner, ok := runners[delCmd.tenant]; ok {
				oldRunner.Quit()
				delete(runners, delCmd.tenant)
			}
			delCmd.reply <- 0
		case <-im.quit:
			for _, runner := range runners {
				runner.Quit()
			}
			im.quit <- 0
			return
		}
	}
}
