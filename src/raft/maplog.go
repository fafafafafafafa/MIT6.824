package raft
import "sync"
import "log"
import "fmt"

type Entry struct{
	Index int	
	Term int	// term when command received by leader
	Command interface{}	// command from state machine
}

type MapLog struct{
	mu sync.Mutex
	mylog *Mylog
	logEntry []Entry

}

func GetMapLog(mylog *Mylog) *MapLog{
	
	var maplog *MapLog
	maplog = &MapLog{}
	maplog.mylog = mylog

	e := Entry{
		Index: 0,
		Term: 0,
		Command: -1,
	}
	maplog.logEntry = append(maplog.logEntry, e)

	return maplog
}

func (maplog *MapLog) GetFirstIndex() int {
	maplog.mu.Lock()
	defer maplog.mu.Unlock()
	
	return 0
}

func (maplog *MapLog) DeleteEntriesBeforeIndex(index int) {
	// include index
	maplog.mu.Lock()
	defer maplog.mu.Unlock()
	if index < 0 || index > len(maplog.logEntry)-1{
		// errMsg := error.New("index illegal! index(%v) \n", index)
		errMsg := fmt.Sprintf("DeleteEntriesBeforeIndex: index illegal! index = %v \n", index)
		maplog.mylog.DFprintf(errMsg)
		log.Fatal(errMsg)
	}else{
		maplog.logEntry = append(maplog.logEntry[:1], maplog.logEntry[index+1:]...)
	}
}


func (maplog *MapLog) DeleteEntriesAfterIndex(index int) {
	// include index
	maplog.mu.Lock()
	defer maplog.mu.Unlock()
	if index < 0 || index > len(maplog.logEntry)-1{
		
	}else{
		
		maplog.logEntry = maplog.logEntry[:index]

	}
}

func (maplog *MapLog) Append(entries Entry) {
	maplog.mu.Lock()
	defer maplog.mu.Unlock()
	l := len(maplog.logEntry)
	maplog.logEntry = append(maplog.logEntry, entries)

	maplog.mylog.DFprintf("Append: len of log from %v to %v", l, len(maplog.logEntry))
	 
}

func (maplog *MapLog) AppendEntries(entries []Entry) {
	maplog.mu.Lock()
	defer maplog.mu.Unlock()
	l := len(maplog.logEntry)
	maplog.logEntry = append(maplog.logEntry, entries...)

	maplog.mylog.DFprintf("AppendEntries: len of log from %v to %v", l, len(maplog.logEntry))
	 
}


func (maplog *MapLog) GetLen() int{
	maplog.mu.Lock()
	defer maplog.mu.Unlock()
	return len(maplog.logEntry)

}

func (maplog *MapLog) GetLastEntry() Entry{
	maplog.mu.Lock()
	defer maplog.mu.Unlock()
	l := len(maplog.logEntry)

	return maplog.logEntry[l-1]
}

func (maplog *MapLog) GetEntryFromIndex(index int) (Entry, bool) {
	maplog.mu.Lock()
	defer maplog.mu.Unlock()
	
	if index < 0 || index > len(maplog.logEntry)-1{
		// errMsg := error.New("index illegal! index(%v) \n", index)
		
		return Entry{}, false
	}else{
		e := maplog.logEntry[index]
		return e, true
	}
	
	
}

func (maplog *MapLog) GetEntriesAfterIndex(index int) []Entry{
	// include index
	maplog.mu.Lock()
	defer maplog.mu.Unlock()
	if index < 0 || index > len(maplog.logEntry)-1{
		return []Entry{}
	}else{
		
		return maplog.logEntry[index:]
		
	}
}