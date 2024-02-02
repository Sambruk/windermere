package ss12000v2import

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"
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
	stopped          bool

	// This mutex protects the contextCanceller and stopped
	// (the other members should be thread safe or only accessed by the ImportRunner which is single threaded)
	lock sync.Mutex

	config RunnerConfig
}

// The RunnerConfig describes how the import should be done for a tenant.
// It's all the information the ImportRunner needs.
type RunnerConfig struct {
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
func NewImportRunner(conf RunnerConfig) *ImportRunner {
	ir := &ImportRunner{
		quit:   make(chan int),
		config: conf,
	}
	go importRunner(ir)
	return ir
}

// Stops the ImportRunner (blocks until the runner has stopped)
func (ir *ImportRunner) Quit() {
	if ir.isStopped() {
		return
	}
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

func (ir *ImportRunner) setStopped() {
	ir.lock.Lock()
	defer ir.lock.Unlock()
	ir.stopped = true
}

func (ir *ImportRunner) isStopped() bool {
	ir.lock.Lock()
	defer ir.lock.Unlock()
	return ir.stopped
}

func timeForFullImport(config RunnerConfig) (bool, error) {
	lastStartedFull, err := config.History.GetTimeOfLastStartedFullImport()
	if err != nil {
		return false, err
	}
	lastCompletedFull, err := config.History.GetTimeOfLastCompletedFullImport()
	if err != nil {
		return false, err
	}
	if lastCompletedFull.Before(lastStartedFull) && time.Now().Sub(lastStartedFull) < config.FullImportRetryWait {
		return false, nil
	} else {
		return time.Now().Sub(lastCompletedFull) > config.FullImportFrequency, nil
	}
}

func timeForIncrementalImport(config RunnerConfig) (bool, error) {
	lastStartedIncremental, err := config.History.GetTimeOfLastStartedIncrementalImport()
	if err != nil {
		return false, err
	}
	lastCompletedIncremental, err := config.History.GetTimeOfLastCompletedIncrementalImport()
	if err != nil {
		return false, err
	}
	fullImport, err := timeForFullImport(config)
	if err != nil {
		return false, err
	}
	if fullImport {
		return false, nil
	} else if lastCompletedIncremental.Before(lastStartedIncremental) && time.Now().Sub(lastStartedIncremental) < config.IncrementalImportRetryWait {
		return false, nil
	} else {
		return time.Now().Sub(lastCompletedIncremental) > config.IncrementalImportFrequency, nil
	}
}

// The import tick carries out the work of the import and is called regularly.
// It will usually do nothing, but sometimes a full import and sometimes an
// incremental import.
// The error returned from this function is ONLY used to signal that a panic has
// happened during the import, and that the import runner should stop.
func (ir *ImportRunner) importTick(ctx context.Context, logger *log.Logger) (panicerr error) {
	// Make sure we recover from panic so as to not crash the whole service,
	// only this import runner should terminate.
	defer func() {
		if r := recover(); r != nil {
			logger.Printf("Unexpected panic in import runner")
			logger.Printf("Recovered reason: %v", r)
			logger.Println("Stacktrace from panic: \n" + string(debug.Stack()))
			panicerr = fmt.Errorf("recovered from panic")
		}
	}()
	timeForFull, err := timeForFullImport(ir.config)
	if err != nil {
		logger.Printf("Failed to determine whether it's time to do a full import for %s: %s", ir.config.Tenant, err.Error())
		return
	}
	if timeForFull {
		err = ir.config.History.SetTimeOfLastStartedFullImport(time.Now())
		if err != nil {
			logger.Printf("Failed to set time of last started full import for %s: %s", ir.config.Tenant, err.Error())
			return
		}
		err = FullImport(ctx, logger, ir.config.Tenant, ir.config.Client, ir.config.Backend, ir.config.History)
		if err == nil {
			err = ir.config.History.SetTimeOfLastCompletedFullImport(time.Now())
			if err != nil {
				logger.Printf("Failed to set time of last completed full import for %s: %s", ir.config.Tenant, err.Error())
			}
		} else {
			logger.Printf("Failed to do full import for %s: %s", ir.config.Tenant, err.Error())
		}
		return
	}

	timeForIncremental, err := timeForIncrementalImport(ir.config)
	if err != nil {
		logger.Printf("Failed to determine whether it's time to do an incremental import for %s: %s", ir.config.Tenant, err.Error())
		return
	}
	if timeForIncremental {
		err = ir.config.History.SetTimeOfLastStartedIncrementalImport(time.Now())
		if err != nil {
			logger.Printf("Failed to set time of last started incremental import for %s: %s", ir.config.Tenant, err.Error())
			return
		}
		err = IncrementalImport(ctx, logger, ir.config.Tenant, ir.config.Client, ir.config.Backend, ir.config.History)
		if err == nil {
			err = ir.config.History.SetTimeOfLastCompletedIncrementalImport(time.Now())
			if err != nil {
				logger.Printf("Failed to set time of last completed incremental import for %s: %s", ir.config.Tenant, err.Error())
			}
		} else {
			logger.Printf("Failed to do incremental import for %s: %s", ir.config.Tenant, err.Error())
		}
		return
	}
	return
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
			err := ir.importTick(ctx, logger)
			ir.setContextCanceller(nil)
			runningCancel()
			if err != nil {
				logger.Printf("Stopping import runner")
				ir.setStopped()
				// In the unlikely event that someone has already asked us to quit
				// we don't want them to block indefinitely waiting on the quit channel.
				select {
				case <-ir.quit:
					ir.quit <- 0
					return
				default:
				}
				return
			}
		}
	}
}
