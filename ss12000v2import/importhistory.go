package ss12000v2import

import "time"

// The import needs to keep track of its history, both to know when to
// to the next full or incremental import, and for the incremental import
// it also needs to know timestamps of objects etc.
// This interface lets the import access and record this information without
// caring about how the history is stored.
type ImportHistory interface {
	GetTimeOfLastStartedFullImport() (time.Time, error)
	GetTimeOfLastCompletedFullImport() (time.Time, error)
	GetTimeOfLastStartedIncrementalImport() (time.Time, error)
	GetTimeOfLastCompletedIncrementalImport() (time.Time, error)

	SetTimeOfLastStartedFullImport(time.Time) error
	SetTimeOfLastCompletedFullImport(time.Time) error
	SetTimeOfLastStartedIncrementalImport(time.Time) error
	SetTimeOfLastCompletedIncrementalImport(time.Time) error

	RecordMostRecent(created []time.Time, modified []time.Time, queryType string) error
	GetMostRecentlyCreated(queryType string) (time.Time, error)
	GetMostRecentlyModified(queryType string) (time.Time, error)

	GetTimeOfLastDeletedEntitiesCall() (time.Time, error)
	SetTimeOfLastDeletedEntitiesCall(t time.Time) error
}

// The InMemoryImportHistory is a simple implementation of ImportHistory
// which only stores the information in memory.
type InMemoryImportHistory struct {
	timeOfLastStartedFullImport          time.Time
	timeOfLastCompletedFullImport        time.Time
	timeOfLastStartedIncrementalImport   time.Time
	timeOfLastCompletedIncrementalImport time.Time
	mostRecentlyCreatedByQueryType       map[string]time.Time
	mostRecentlyModifiedByQueryType      map[string]time.Time
	timeOfLastDeletedEntitiesCall        time.Time
}

func NewInMemoryImportHistory() *InMemoryImportHistory {
	var i InMemoryImportHistory
	i.mostRecentlyCreatedByQueryType = make(map[string]time.Time)
	i.mostRecentlyModifiedByQueryType = make(map[string]time.Time)
	return &i
}

func (i *InMemoryImportHistory) GetTimeOfLastStartedFullImport() (time.Time, error) {
	return i.timeOfLastStartedFullImport, nil
}

func (i *InMemoryImportHistory) GetTimeOfLastCompletedFullImport() (time.Time, error) {
	return i.timeOfLastCompletedFullImport, nil
}

func (i *InMemoryImportHistory) GetTimeOfLastStartedIncrementalImport() (time.Time, error) {
	return i.timeOfLastStartedIncrementalImport, nil
}

func (i *InMemoryImportHistory) GetTimeOfLastCompletedIncrementalImport() (time.Time, error) {
	return i.timeOfLastCompletedIncrementalImport, nil
}

func (i *InMemoryImportHistory) SetTimeOfLastStartedFullImport(t time.Time) error {
	i.timeOfLastStartedFullImport = t
	return nil
}

func (i *InMemoryImportHistory) SetTimeOfLastCompletedFullImport(t time.Time) error {
	i.timeOfLastCompletedFullImport = t
	return nil
}

func (i *InMemoryImportHistory) SetTimeOfLastStartedIncrementalImport(t time.Time) error {
	i.timeOfLastStartedIncrementalImport = t
	return nil
}

func (i *InMemoryImportHistory) SetTimeOfLastCompletedIncrementalImport(t time.Time) error {
	i.timeOfLastCompletedIncrementalImport = t
	return nil
}

func (i *InMemoryImportHistory) RecordMostRecent(created, modified []time.Time, queryType string) error {
	createdMostRecently, ok := i.mostRecentlyCreatedByQueryType[queryType]
	var zeroTime time.Time
	if !ok {
		createdMostRecently = zeroTime
	}
	modifiedMostRecently, ok := i.mostRecentlyModifiedByQueryType[queryType]
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

	i.mostRecentlyCreatedByQueryType[queryType] = createdMostRecently
	i.mostRecentlyModifiedByQueryType[queryType] = modifiedMostRecently
	return nil
}

func (i *InMemoryImportHistory) GetMostRecentlyCreated(queryType string) (time.Time, error) {
	val, ok := i.mostRecentlyCreatedByQueryType[queryType]
	var zeroTime time.Time
	if !ok {
		return zeroTime, nil
	}
	return val, nil
}

func (i *InMemoryImportHistory) GetMostRecentlyModified(queryType string) (time.Time, error) {
	val, ok := i.mostRecentlyModifiedByQueryType[queryType]
	var zeroTime time.Time
	if !ok {
		return zeroTime, nil
	}
	return val, nil
}

func (i *InMemoryImportHistory) GetTimeOfLastDeletedEntitiesCall() (time.Time, error) {
	return i.timeOfLastDeletedEntitiesCall, nil
}

func (i *InMemoryImportHistory) SetTimeOfLastDeletedEntitiesCall(t time.Time) error {
	i.timeOfLastDeletedEntitiesCall = t
	return nil
}
