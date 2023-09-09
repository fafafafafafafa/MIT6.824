package mr
import "sync"
// import "fmt"

type node struct{

	prevNode *node
	nextNode *node
	data *TaskState
}
type LinkedList struct{
	mu sync.Mutex
	Head *node
	Tail *node
	count int

}

func GetNewQueue() *LinkedList{
	
	l := LinkedList{}
	//fmt.Printf("in GetNewQueue func, l =%v\n", l)
	l.Head = &node{}
	l.Tail = &node{}

	l.Head.prevNode = l.Head
	l.Head.nextNode = l.Tail
	l.Tail.prevNode = l.Head
	l.Tail.nextNode = l.Tail

	l.count = 0
	return &l
}
func (l *LinkedList) GetLen() int{
	l.mu.Lock()
	l.mu.Unlock()
	return l.count
}
func (l *LinkedList) AddTail(t *TaskState){
	l.mu.Lock()
	defer l.mu.Unlock()
	n := &node{}
	n.data = t
	
	l.Tail.prevNode.nextNode = n
	n.prevNode = l.Tail.prevNode
	n.nextNode = l.Tail
	l.Tail.prevNode = n
	//fmt.Printf("in AddTail func, n = %+v\n", n)
	l.count ++

}
func (l *LinkedList) AddHead(t *TaskState){
	l.mu.Lock()
	defer l.mu.Unlock()
	n := &node{}
	n.data = t

	l.Head.nextNode.prevNode = n
	n.nextNode = l.Head.nextNode
	l.Head.nextNode = n
	n.prevNode = l.Head

	l.count ++
}
func (l *LinkedList) PopHead() *TaskState{
	l.mu.Lock()
	defer l.mu.Unlock()

	n := l.Head.nextNode
	n.nextNode.prevNode = l.Head
	l.Head.nextNode = n.nextNode
	l.count --

	return n.data
}
func (l *LinkedList) RemoveTask(t *TaskState) bool{
	l.mu.Lock()
	defer l.mu.Unlock()
	n := l.Head.nextNode
	// fmt.Printf("in RemoveTask func, t = %+v\n", t)

	for ; n != l.Tail; {
		// fmt.Printf("in RemoveTask func, n = %+v\n", n.data)
		if *n.data == *t{
			break
		}
		n = n.nextNode
	}
	if n == l.Tail{
		// fmt.Printf("in RemoveTask func, remove failed!\n")
		return false
	}else{
		n.nextNode.prevNode = n.prevNode
		n.prevNode.nextNode = n.nextNode
		l.count --
		// fmt.Printf("in RemoveTask func, remove successful!\n")
		// fmt.Printf("in RemoveTask func, %v task is running\n", l.count)

		return true
	}
}

func  (l *LinkedList) CheckTimeout(outtime int64) []*TaskState{
	l.mu.Lock()
	defer l.mu.Unlock()
	curtime := getNowTimeSecond()

	n := l.Head.nextNode
	retqueue := make([]*TaskState, 0)
	for ; n != l.Tail; {
		if (curtime-n.data.StartTime)>outtime{
			task := n.data
			retqueue = append(retqueue, task)

			temp := n.prevNode
			n.prevNode.nextNode = n.nextNode
			n.nextNode.prevNode = n.prevNode
			l.count--
			n = temp
		}
		n = n.nextNode
	}
	return retqueue

}
