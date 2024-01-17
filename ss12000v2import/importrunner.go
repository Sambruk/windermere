package ss12000v2import

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Sambruk/windermere/ss12000v2"
)

// The ImportRunner handles the import for a single tenant.
// It takes care of running full and incremental imports when it's
// time to do so. The runner executes in its own single threaded goroutine.
// The import's configuration is set when the runner is created and started,
// when an import needs to be reconfigured the runner is stopped and removed
// and a new ImportRunner for that tenant is created with the new configuration.
type ImportRunner struct {
	quit             chan int
	contextCanceller context.CancelFunc

	// This mutex protects the contextCanceller
	// (the other members should be thread safe or only accessed by the ImportRunner which is single threaded)
	lock sync.Mutex

	config ImportConfig
}

// The ImportConfig describes how the import should be done for a tenant.
type ImportConfig struct {
	Tenant                     string
	Backend                    SS12000v1Backend
	Client                     ss12000v2.ClientInterface
	History                    ImportHistory
	FullImportFrequency        time.Duration
	FullImportRetryWait        time.Duration
	IncrementalImportFrequency time.Duration
	IncrementalImportRetryWait time.Duration
}

// Creates and starts a new ImportRunner
func NewImportRunner(conf ImportConfig) *ImportRunner {
	ir := &ImportRunner{
		quit:   make(chan int),
		config: conf,
	}
	go importRunner(ir)
	return ir
}

// Stops the ImportRunner (blocks until the runner has stopped)
func (ir *ImportRunner) Quit() {
	canceller := ir.getContextCanceller()
	if canceller != nil {
		canceller()
	}

	ir.quit <- 0
	<-ir.quit
}

func (ir *ImportRunner) setContextCanceller(cf context.CancelFunc) {
	ir.lock.Lock()
	defer ir.lock.Unlock()
	ir.contextCanceller = cf
}

func (ir *ImportRunner) getContextCanceller() context.CancelFunc {
	ir.lock.Lock()
	defer ir.lock.Unlock()
	return ir.contextCanceller
}

func timeForFullImport(config ImportConfig) bool {
	if time.Now().Sub(config.History.GetTimeOfLastStartedFullImport()) < config.FullImportRetryWait {
		return false
	} else {
		return time.Now().Sub(config.History.GetTimeOfLastCompletedFullImport()) > config.FullImportFrequency
	}
}

func timeForIncrementalImport(config ImportConfig) bool {
	if timeForFullImport(config) {
		return false
	} else if time.Now().Sub(config.History.GetTimeOfLastStartedIncrementalImport()) < config.IncrementalImportRetryWait {
		return false
	} else {
		return time.Now().Sub(config.History.GetTimeOfLastCompletedIncrementalImport()) > config.IncrementalImportFrequency
	}
}

func (ir *ImportRunner) importTick(ctx context.Context, logger *log.Logger) {
	var err error
	if timeForFullImport(ir.config) {
		ir.config.History.SetTimeOfLastStartedFullImport(time.Now())
		err = FullImport(ctx, logger, ir.config.Tenant, ir.config.Client, ir.config.Backend, ir.config.History)
		if err == nil {
			ir.config.History.SetTimeOfLastCompletedFullImport(time.Now())
		}
	} else if timeForIncrementalImport(ir.config) {
		ir.config.History.SetTimeOfLastStartedIncrementalImport(time.Now())
		err = IncrementalImport(ctx, logger, ir.config.Tenant, ir.config.Client, ir.config.Backend, ir.config.History)
		if err == nil {
			ir.config.History.SetTimeOfLastCompletedIncrementalImport(time.Now())
		}
	}
	if err != nil {
		logger.Printf("Failed to import for %s: %s", ir.config.Tenant, err.Error())
	}
}

func importRunner(ir *ImportRunner) {
	logger := log.New(os.Stderr, fmt.Sprintf("SS12000 Import:%s: ", ir.config.Tenant), log.LstdFlags|log.Lmsgprefix)

	retry := time.NewTicker(5 * time.Second)
	defer retry.Stop()

	for {
		// The next select below this one will choose randomly but we
		// want to prioritize quit if there's something on that channel,
		// that's why we have an extra non-blocking select here.
		select {
		case <-ir.quit:
			ir.quit <- 0
			return
		default:
		}

		select {
		case <-ir.quit:
			ir.quit <- 0
			return
		case <-retry.C:
			ctx, runningCancel := context.WithCancel(context.Background())
			ir.setContextCanceller(runningCancel)
			ir.importTick(ctx, logger)
			ir.setContextCanceller(nil)
			runningCancel()
		}
	}
}
