package raft

import "log"

// Debugging
const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
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
