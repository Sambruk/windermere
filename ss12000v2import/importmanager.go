package ss12000v2import

// The ImportManager keeps track of the import runners (ImportRunner)
// It's basically just a container for ImportRunner which allows adding
// and deleting runners and controlled shutdown.
type ImportManager struct {
	quit   chan int
	add    chan RunnerConfig
	delete chan string
}

func NewImportManager() *ImportManager {
	var im ImportManager
	im.quit = make(chan int)
	im.add = make(chan RunnerConfig)
	im.delete = make(chan string)
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

func (im *ImportManager) DeleteRunner(tenant string) {
	im.delete <- tenant
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
		case tenant := <-im.delete:
			if oldRunner, ok := runners[tenant]; ok {
				oldRunner.Quit()
				delete(runners, tenant)
			}
		case <-im.quit:
			for _, runner := range runners {
				runner.Quit()
			}
			im.quit <- 0
			return
		}
	}
}
