package ss12000v2import

import "time"

// The import needs to keep track of its history, both to know when to
// to the next full or incremental import, and for the incremental import
// it also needs to know timestamps of objects etc.
// This interface lets the import access and record this information without
// caring about how the history is stored.
type ImportHistory interface {
	GetTimeOfLastStartedFullImport() time.Time
	GetTimeOfLastCompletedFullImport() time.Time
	GetTimeOfLastStartedIncrementalImport() time.Time
	GetTimeOfLastCompletedIncrementalImport() time.Time

	SetTimeOfLastStartedFullImport(time.Time)
	SetTimeOfLastCompletedFullImport(time.Time)
	SetTimeOfLastStartedIncrementalImport(time.Time)
	SetTimeOfLastCompletedIncrementalImport(time.Time)

	RecordMostRecent(created []time.Time, modified []time.Time, queryType string)
	GetMostRecentlyCreated(queryType string) time.Time
	GetMostRecentlyModified(queryType string) time.Time

	GetTimeOfLastDeletedEntitiesCall() time.Time
	SetTimeOfLastDeletedEntitiesCall(t time.Time)
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

func (i *InMemoryImportHistory) GetTimeOfLastStartedFullImport() time.Time {
	return i.timeOfLastStartedFullImport
}

func (i *InMemoryImportHistory) GetTimeOfLastCompletedFullImport() time.Time {
	return i.timeOfLastCompletedFullImport
}

func (i *InMemoryImportHistory) GetTimeOfLastStartedIncrementalImport() time.Time {
	return i.timeOfLastStartedIncrementalImport
}

func (i *InMemoryImportHistory) GetTimeOfLastCompletedIncrementalImport() time.Time {
	return i.timeOfLastCompletedIncrementalImport
}

func (i *InMemoryImportHistory) SetTimeOfLastStartedFullImport(t time.Time) {
	i.timeOfLastStartedFullImport = t
}

func (i *InMemoryImportHistory) SetTimeOfLastCompletedFullImport(t time.Time) {
	i.timeOfLastCompletedFullImport = t
}

func (i *InMemoryImportHistory) SetTimeOfLastStartedIncrementalImport(t time.Time) {
	i.timeOfLastStartedIncrementalImport = t
}

func (i *InMemoryImportHistory) SetTimeOfLastCompletedIncrementalImport(t time.Time) {
	i.timeOfLastCompletedIncrementalImport = t
}

func (i *InMemoryImportHistory) RecordMostRecent(created, modified []time.Time, queryType string) {
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
}

func (i *InMemoryImportHistory) GetMostRecentlyCreated(queryType string) time.Time {
	val, ok := i.mostRecentlyCreatedByQueryType[queryType]
	var zeroTime time.Time
	if !ok {
		return zeroTime
	}
	return val
}

func (i *InMemoryImportHistory) GetMostRecentlyModified(queryType string) time.Time {
	val, ok := i.mostRecentlyModifiedByQueryType[queryType]
	var zeroTime time.Time
	if !ok {
		return zeroTime
	}
	return val
}

func (i *InMemoryImportHistory) GetTimeOfLastDeletedEntitiesCall() time.Time {
	return i.timeOfLastDeletedEntitiesCall
}

func (i *InMemoryImportHistory) SetTimeOfLastDeletedEntitiesCall(t time.Time) {
	i.timeOfLastDeletedEntitiesCall = t
}
