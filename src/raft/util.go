package raft

import "log"
import "io"
import "fmt"
import "runtime/pprof"
// Debugging
const Debug = true

type Mylog struct{
	W io.Writer
	Debug bool
} 

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

func (mylog *Mylog)DFprintf(format string, a ...interface{}) (n int, err error) {
	mylog.Debug = Debug
	if mylog.Debug {
		// log.Printf(format, a...)
		fmt.Fprintf(mylog.W, format, a...)
		log.Printf(format, a...)
	}
	return
}
func (mylog *Mylog)GoroutineStack(){
	_ = pprof.Lookup("goroutine").WriteTo(mylog.W, 1)
}

func DPrintAllRafts(rafts []*Raft, connected []bool){
	if Debug{
	log.Println("^^^^^^^^print all rafts info^^^^^^^^")

	
		for i, raft := range rafts{
			term, isleader := raft.GetState()
			log.Printf("raft %v: raft.term %v, raft.isleader %v, connected %v\n", i, term, isleader, connected[i])
		}
	
	log.Println("^^^^^^^^print all rafts info^^^^^^^^")
	}
}


