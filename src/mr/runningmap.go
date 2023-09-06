package mr
import "sync"

type RunningMap struct{
	m sync.Mutex
	isrunnig map[TaskState]bool
	count int
}


func GetRunningMap() *RunningMap{
	rmap := RunningMap{}
	rmap.isrunnig = make(map[TaskState]bool)
	rmap.count = 0
	return &rmap

}

func (rmap *RunningMap)Insert(t TaskState){
	rmap.m.Lock()
	rmap.isrunnig[t] = true
	rmap.count += 1
	rmap.m.Unlock()
}

func (rmap *RunningMap)Remove(t TaskState){
	rmap.m.Lock()
	rmap.isrunnig[t] = false
	rmap.count += 1
	rmap.m.Unlock()

}

func (rmap *RunningMap)IsEmpty() bool {
	rmap.m.Lock()
	defer rmap.m.Unlock()
	if rmap.count == 0{
		return true
	}else{
		return false
	}

}

func (rmap *RunningMap)GetLen() int {
	rmap.m.Lock()
	defer rmap.m.Unlock()
	return rmap.count

}