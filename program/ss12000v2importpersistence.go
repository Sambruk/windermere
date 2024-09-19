package program

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Sambruk/windermere/ss12000v2import"
	"github.com/jmoiron/sqlx"
)

// Database driver used for the import persistence
const DriverName = "sqlite"

// The import persistence layer is responsible for storing import
// configurations and import history to persistent storage.
type ss12000v2ImportPersistence struct {
	db *sqlx.DB
}

// Creates a new persistence layer, path is the path to the file where we
// will store the information.
func NewSS12000v2ImportPersistence(path string) (*ss12000v2ImportPersistence, error) {
	database, err := sqlx.Open(DriverName, path)

	if err != nil {
		return nil, err
	}

	persistence := &ss12000v2ImportPersistence{
		db: database,
	}

	err = persistence.initSchema()

	if err != nil {
		return nil, err
	}

	return persistence, nil
}

func (p *ss12000v2ImportPersistence) initSchema() error {
	tx, err := p.db.Beginx()

	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS configs (
		  tenant NVARCHAR(255) NOT NULL,
		  config BLOB NOT NULL,
		  history BLOB NOT NULL,
		  PRIMARY KEY (tenant));
		 `, nil)

	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

// Creates or updates an import.
// If the import already existed, the history is untouched.
func (p *ss12000v2ImportPersistence) AddImport(config ImportConfig) error {
	configBuff, err := json.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %s", err.Error())
	}

	emptyHistory := NewTenantImportHistory()
	historyBuff, err := json.Marshal(&emptyHistory)
	if err != nil {
		return fmt.Errorf("failed to marshal empty history: %s", err.Error())
	}
	_, err = p.db.NamedExec(`INSERT INTO configs (tenant, config, history) VALUES (:tenant, :config, :history) ON CONFLICT (tenant) DO UPDATE SET config=:config`,
		map[string]interface{}{
			"tenant":  config.Tenant,
			"config":  configBuff,
			"history": historyBuff,
		})

	if err != nil {
		return fmt.Errorf("failed to upsert config: %s", err.Error())
	}
	return nil
}

func (p *ss12000v2ImportPersistence) DeleteImport(tenant string) error {
	_, err := p.db.NamedExec(`DELETE FROM configs WHERE tenant=:tenant`,
		map[string]interface{}{
			"tenant": tenant,
		})
	return err
}

func (p *ss12000v2ImportPersistence) GetHistory(tenant string) ss12000v2import.ImportHistory {
	ih := PersistenceImportHistory{
		db:     p.db,
		tenant: tenant,
	}
	return &ih
}

func (p *ss12000v2ImportPersistence) GetAllImports() ([]string, error) {
	tenants := []string{}
	err := p.db.Select(&tenants, `SELECT tenant FROM configs`)
	if err != nil {
		return nil, err
	}
	return tenants, nil
}

func getDatabaseRow(db *sqlx.DB, tenant string) ([]byte, []byte, error) {
	type row struct {
		Tenant  string `db:"tenant"`
		Config  []byte `db:"config"`
		History []byte `db:"history"`
	}

	namedQuery, err := db.PrepareNamed(`SELECT * FROM configs WHERE tenant = :tenant`)
	if err != nil {
		return nil, nil, err
	}
	defer namedQuery.Close()

	rows := []row{}
	err = namedQuery.Select(&rows,
		map[string]interface{}{
			"tenant": tenant,
		})
	if err != nil {
		return nil, nil, err
	} else if len(rows) < 1 {
		return nil, nil, nil
	}
	return rows[0].Config, rows[0].History, nil
}

// Finds the configuration for a tenant. Error means failure to read
// or parse from the storage. If the tenant didn't have a configured
// import a nil ImportConfig is returned (and no error).
func (p *ss12000v2ImportPersistence) GetImportConfig(tenant string) (*ImportConfig, error) {
	configBlob, _, err := getDatabaseRow(p.db, tenant)
	if err != nil {
		return nil, err
	} else if configBlob == nil {
		return nil, nil
	}
	var config ImportConfig
	err = json.Unmarshal(configBlob, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal import config: %s", err.Error())
	}
	return &config, nil
}

// This type implements ss12000v2import.ImportHistory for this persistence layer
type PersistenceImportHistory struct {
	db     *sqlx.DB
	tenant string
}

// This is what we store in the database to keep track of history for a single tenant
type TenantImportHistory struct {
	TimeOfLastStartedFullImport          time.Time
	TimeOfLastCompletedFullImport        time.Time
	TimeOfLastStartedIncrementalImport   time.Time
	TimeOfLastCompletedIncrementalImport time.Time
	MostRecentlyCreatedByQueryType       map[string]time.Time
	MostRecentlyModifiedByQueryType      map[string]time.Time
	TimeOfLastDeletedEntitiesCall        time.Time
}

func NewTenantImportHistory() TenantImportHistory {
	var result TenantImportHistory
	result.MostRecentlyCreatedByQueryType = make(map[string]time.Time)
	result.MostRecentlyModifiedByQueryType = make(map[string]time.Time)
	return result
}

func (ih *PersistenceImportHistory) getHistory() (TenantImportHistory, error) {
	_, historyBlob, err := getDatabaseRow(ih.db, ih.tenant)
	if err != nil {
		return TenantImportHistory{}, err
	} else if historyBlob == nil {
		return TenantImportHistory{}, fmt.Errorf("failed to get history for non-existant import: %s", ih.tenant)
	}
	var history TenantImportHistory
	err = json.Unmarshal(historyBlob, &history)
	if err != nil {
		return TenantImportHistory{}, fmt.Errorf("failed to unmarshal import history: %s", err.Error())
	}
	return history, nil
}

func (ih *PersistenceImportHistory) setHistory(history TenantImportHistory) error {
	blob, err := json.Marshal(history)
	if err != nil {
		return nil
	}
	_, err = ih.db.NamedExec(`UPDATE configs SET history = :history WHERE tenant = :tenant`,
		map[string]interface{}{
			"tenant":  ih.tenant,
			"history": blob,
		})

	if err != nil {
		return fmt.Errorf("failed to update import history: %s", err.Error())
	}
	return nil
}

func (ih *PersistenceImportHistory) GetTimeOfLastStartedFullImport() (time.Time, error) {
	history, err := ih.getHistory()
	if err != nil {
		return time.Time{}, err
	}
	return history.TimeOfLastStartedFullImport, nil
}

func (ih *PersistenceImportHistory) GetTimeOfLastCompletedFullImport() (time.Time, error) {
	history, err := ih.getHistory()
	if err != nil {
		return time.Time{}, err
	}
	return history.TimeOfLastCompletedFullImport, nil
}

func (ih *PersistenceImportHistory) GetTimeOfLastStartedIncrementalImport() (time.Time, error) {
	history, err := ih.getHistory()
	if err != nil {
		return time.Time{}, err
	}
	return history.TimeOfLastStartedIncrementalImport, nil
}

func (ih *PersistenceImportHistory) GetTimeOfLastCompletedIncrementalImport() (time.Time, error) {
	history, err := ih.getHistory()
	if err != nil {
		return time.Time{}, err
	}
	return history.TimeOfLastCompletedIncrementalImport, nil
}

func (ih *PersistenceImportHistory) SetTimeOfLastStartedFullImport(t time.Time) error {
	history, err := ih.getHistory()
	if err != nil {
		return err
	}
	history.TimeOfLastStartedFullImport = t
	return ih.setHistory(history)
}

func (ih *PersistenceImportHistory) SetTimeOfLastCompletedFullImport(t time.Time) error {
	history, err := ih.getHistory()
	if err != nil {
		return err
	}
	history.TimeOfLastCompletedFullImport = t
	return ih.setHistory(history)
}

func (ih *PersistenceImportHistory) SetTimeOfLastStartedIncrementalImport(t time.Time) error {
	history, err := ih.getHistory()
	if err != nil {
		return err
	}
	history.TimeOfLastStartedIncrementalImport = t
	return ih.setHistory(history)
}

func (ih *PersistenceImportHistory) SetTimeOfLastCompletedIncrementalImport(t time.Time) error {
	history, err := ih.getHistory()
	if err != nil {
		return err
	}
	history.TimeOfLastCompletedIncrementalImport = t
	return ih.setHistory(history)
}

func (ih *PersistenceImportHistory) RecordMostRecent(created []time.Time, modified []time.Time, queryType string) error {
	history, err := ih.getHistory()
	if err != nil {
		return err
	}

	createdMostRecently, ok := history.MostRecentlyCreatedByQueryType[queryType]
	var zeroTime time.Time
	if !ok {
		createdMostRecently = zeroTime
	}
	modifiedMostRecently, ok := history.MostRecentlyModifiedByQueryType[queryType]
	if !ok {
		modifiedMostRecently = zeroTime
	}

	for i := range created {
		if created[i].After(createdMostRecently) {
			createdMostRecently = created[i]
		}
	}

	for i := range modified {
		if modified[i].After(modifiedMostRecently) {
			modifiedMostRecently = modified[i]
		}
	}

	history.MostRecentlyCreatedByQueryType[queryType] = createdMostRecently
	history.MostRecentlyModifiedByQueryType[queryType] = modifiedMostRecently
	return ih.setHistory(history)
}

func (ih *PersistenceImportHistory) GetMostRecentlyCreated(queryType string) (time.Time, error) {
	history, err := ih.getHistory()
	if err != nil {
		return time.Time{}, err
	}
	val, ok := history.MostRecentlyCreatedByQueryType[queryType]
	var zeroTime time.Time
	if !ok {
		return zeroTime, nil
	}
	return val, nil
}

func (ih *PersistenceImportHistory) GetMostRecentlyModified(queryType string) (time.Time, error) {
	history, err := ih.getHistory()
	if err != nil {
		return time.Time{}, err
	}
	val, ok := history.MostRecentlyModifiedByQueryType[queryType]
	var zeroTime time.Time
	if !ok {
		return zeroTime, nil
	}
	return val, nil
}

func (ih *PersistenceImportHistory) GetTimeOfLastDeletedEntitiesCall() (time.Time, error) {
	history, err := ih.getHistory()
	if err != nil {
		return time.Time{}, err
	}
	return history.TimeOfLastDeletedEntitiesCall, nil
}

func (ih *PersistenceImportHistory) SetTimeOfLastDeletedEntitiesCall(t time.Time) error {
	history, err := ih.getHistory()
	if err != nil {
		return err
	}
	history.TimeOfLastDeletedEntitiesCall = t
	return ih.setHistory(history)
}
