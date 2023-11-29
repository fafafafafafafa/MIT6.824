package raft
import "sync"
// import "log"
// import "fmt"

type Entry struct{
	Index int	
	Term int	// term when command received by leader
	Command interface{}	// command from state machine
}

type ListLog struct{
	mu sync.Mutex
	mylog *Mylog
	logEntry []Entry

}

func GetListLog(mylog *Mylog) *ListLog{
	
	var listLog *ListLog
	listLog = &ListLog{}
	listLog.mylog = mylog

	e := Entry{
		Index: 0,
		Term: 0,
		Command: -1,
	}
	listLog.logEntry = append(listLog.logEntry, e)

	return listLog
}


func (listLog *ListLog) DeleteEntriesBeforeIndex(index int) {
	// not include index
	listLog.mu.Lock()
	defer listLog.mu.Unlock()
	if index < 0 || index > len(listLog.logEntry)-1{
	
		listLog.mylog.DFprintf("DeleteEntriesBeforeIndex: index is illegal, no change for lsitlog\n")
		
	}else{
		listLog.logEntry = listLog.logEntry[index:]
	}
}


func (listLog *ListLog) DeleteEntriesAfterIndex(index int) {
	// not include index
	listLog.mu.Lock()
	defer listLog.mu.Unlock()
	if index < 0 || index > len(listLog.logEntry)-1{
		
	}else{
		
		listLog.logEntry = listLog.logEntry[:index]

	}
}

func (listLog *ListLog) Append(entries Entry) {
	listLog.mu.Lock()
	defer listLog.mu.Unlock()
	l := len(listLog.logEntry)
	listLog.logEntry = append(listLog.logEntry, entries)

	listLog.mylog.DFprintf("listLog.Append: len of log from %v to %v\n", l, len(listLog.logEntry))
	 
}

func (listLog *ListLog) AppendEntries(entries []Entry) {
	listLog.mu.Lock()
	defer listLog.mu.Unlock()
	l := len(listLog.logEntry)
	listLog.logEntry = append(listLog.logEntry, entries...)

	listLog.mylog.DFprintf("listLog.AppendEntries: len of log from %v to %v\n", l, len(listLog.logEntry))
	 
}


func (listLog *ListLog) GetLen() int{
	listLog.mu.Lock()
	defer listLog.mu.Unlock()
	return len(listLog.logEntry)

}

func (listLog *ListLog) GetLastEntry() Entry{
	listLog.mu.Lock()
	defer listLog.mu.Unlock()
	l := len(listLog.logEntry)

	return listLog.logEntry[l-1]
}

func (listLog *ListLog) GetEntryFromIndex(index int) (Entry, bool) {
	listLog.mu.Lock()
	defer listLog.mu.Unlock()
	
	if index < 0 || index > len(listLog.logEntry)-1{
		// errMsg := error.New("index illegal! index(%v) \n", index)
		
		return Entry{}, false
	}else{
		e := listLog.logEntry[index]
		return e, true
	}
	
	
}

func (listLog *ListLog) GetEntriesAfterIndex(index int) []Entry{
	// include index
	listLog.mu.Lock()
	defer listLog.mu.Unlock()
	if index < 0 || index > len(listLog.logEntry)-1{
		return []Entry{}
	}else{
		slice := listLog.logEntry[index:]
		
		reEntries := make([]Entry, len(slice), len(slice))
		copy(reEntries, slice)
		return reEntries
	}
}

