package mr

import "log"
import "net"
import "os"
import "net/rpc"
import "net/http"
import "sync"
import "time"
import "fmt"
//import "6.824/mr/runningmap"

type Coordinator struct {
	// Your definitions here.
	mu sync.Mutex
	outtime int64 // patince of waiting task to complete
	nFiles int // num of files 
	nReduce int // num of nReduce

	mapTaskWaitingQueue *LinkedList
	mapTaskRunningQueue *LinkedList

	reduceTaskWaitingQueue *LinkedList
	reduceTaskRunningQueue *LinkedList

	mapTaskMap *RunningMap
	reduceTaskMap *RunningMap
}
type TaskState struct {

	StartTime int64
	FileName string

	X int // map task num
	Y int // reduce task num 

	NFiles int // num of files 
	NReduce int // num of nReduce

}

func getNowTimeSecond() int64 {
	return time.Now().UnixNano() / int64(time.Second)
}

// Your code here -- RPC handlers for the worker to call.
func (c *Coordinator) AskTask(args *AskArgs, reply *AskReply) error {
	
	// fmt.Println("***********************************************")
	// defer fmt.Println("----------------------------------------------")
	// fmt.Println("in AskTask func , len of c.mapTaskWaitingQueue=", c.mapTaskWaitingQueue.GetLen())
	// fmt.Println("in AskTask func , len of c.mapTaskRunningQueue=", c.mapTaskRunningQueue.GetLen())
	// fmt.Println("in AskTask func , len of c.mapTaskMap=", c.mapTaskMap.GetLen())

	// fmt.Println("in AskTask func , len of c.reduceTaskWaitingQueue=", c.reduceTaskWaitingQueue.GetLen())
	// fmt.Println("in AskTask func , len of c.reduceTaskRunningQueue=", c.reduceTaskRunningQueue.GetLen())
	// fmt.Println("in AskTask func , len of c.reduceTaskMap=", c.reduceTaskMap.GetLen())
	c.mu.Lock()
	if c.mapTaskWaitingQueue.GetLen() > 0{
		
		maptask := c.mapTaskWaitingQueue.PopHead()
		c.mu.Unlock()
		curtime := getNowTimeSecond()
		maptask.StartTime = curtime
		c.mapTaskRunningQueue.AddTail(maptask)
		// fmt.Printf("in AskTask func, give mapTask , maptask=%+v\n", maptask)
		*reply = AskReply{
			StartTime: curtime,
			FileName: maptask.FileName,
			X: maptask.X,
			Y: maptask.Y,
			NFiles: maptask.NFiles,
			NReduce: maptask.NReduce,
			Method: "map",
		}
		return nil
	}else if c.mapTaskMap.GetLen() < c.nFiles{
		c.mu.Unlock()
		*reply = AskReply{
			Method: "wait",
		}
		return nil
	}
	if c.reduceTaskWaitingQueue.GetLen() > 0{
		reducetask := c.reduceTaskWaitingQueue.PopHead()
		c.mu.Unlock()
		curtime := getNowTimeSecond()
		reducetask.StartTime = curtime
		c.reduceTaskRunningQueue.AddTail(reducetask)
		// fmt.Printf("in AskTask func, give reducetask , reducetask=%+v\n", reducetask)
		*reply = AskReply{
			StartTime: curtime,
			FileName: reducetask.FileName,
			X: reducetask.X,
			Y: reducetask.Y,
			NFiles: reducetask.NFiles,
			NReduce: reducetask.NReduce,
			Method: "reduce",
		}
		return nil
	}else if c.reduceTaskMap.GetLen() < c.nReduce{
		c.mu.Unlock()
		*reply = AskReply{
			Method: "wait",
		}
		return nil
	}

	if c.mapTaskWaitingQueue.GetLen()==0 && c.reduceTaskWaitingQueue.GetLen()==0{
		c.mu.Unlock()
		*reply = AskReply{
			Method: "done",
		}
	}

	return nil
}
func (c *Coordinator) TaskDone(args *AskReply, reply *AskArgs) error{
	
	curtime := getNowTimeSecond()
	runtime := curtime-args.StartTime
	// fmt.Println("***********************************************")
	// defer fmt.Println("----------------------------------------------")
	// fmt.Println("in TaskDone func , len of c.mapTaskWaitingQueue=", c.mapTaskWaitingQueue.GetLen())
	// fmt.Println("in TaskDone func , len of c.mapTaskRunningQueue=", c.mapTaskRunningQueue.GetLen())
	// fmt.Println("in TaskDone func , len of c.mapTaskMap=", c.mapTaskMap.GetLen())

	// fmt.Println("in TaskDone func , len of c.reduceTaskWaitingQueue=", c.reduceTaskWaitingQueue.GetLen())
	// fmt.Println("in TaskDone func , len of c.reduceTaskRunningQueue=", c.reduceTaskRunningQueue.GetLen())
	// fmt.Println("in TaskDone func , len of c.reduceTaskMap=", c.reduceTaskMap.GetLen())

	// fmt.Printf("in TaskDone func, StartTime = %v, CurTime = %v\n", args.StartTime, curtime)
	// fmt.Printf("in TaskDone func, runtime = %v\n", runtime)
	
	switch args.Method{
		case "map":{
			
			maptask := TaskState{
				StartTime: args.StartTime,
				FileName: args.FileName,
				X: args.X,
				Y: args.Y,
				NFiles: args.NFiles,
				NReduce: args.NReduce,
				
			}
			//cur_time := getNowTimeSecond()
			if runtime > c.outtime{
				// assume this worker has been died 
				fmt.Println("task died!")
			}else{
				//c.mu.Lock()
				c.mapTaskMap.Insert(maptask)
				flag := c.mapTaskRunningQueue.RemoveTask(&maptask)
				//c.mu.Unlock()
				if flag == false{
					log.Fatal("remove maptask failed!")
				}
			}
			
		
		}
		case "reduce":{
			reducetask := TaskState{
				StartTime: args.StartTime,
				FileName: args.FileName,
				X: args.X,
				Y: args.Y,
				NFiles: args.NFiles,
				NReduce: args.NReduce,

			}
			if runtime > c.outtime{
				// assume this worker has been died 
				fmt.Println("task died!")
				
			}else{
				//c.mu.Lock()
				c.reduceTaskMap.Insert(reducetask)
				flag := c.reduceTaskRunningQueue.RemoveTask(&reducetask)
				//c.mu.Unlock()
				if flag == false{
					log.Fatal("remove reducetask failed!")
				}
			}
			
		
		}
	}
	return nil

}
//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

//
// start a thread that listens for RPCs from worker.go
//
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}
//
// check whether there is a worker died
//
func (c *Coordinator) CheckTaskDied(){
	
	// fmt.Println("***********************************************")
	// defer fmt.Println("----------------------------------------------")
	// fmt.Printf("^^^^^^ in CheckTaskDied func , time = %v ^^^^^^\n", getNowTimeSecond())
	// fmt.Println("in CheckTaskDied func , len of c.mapTaskWaitingQueue=", c.mapTaskWaitingQueue.GetLen())
	// fmt.Println("in CheckTaskDied func , len of c.mapTaskRunningQueue=", c.mapTaskRunningQueue.GetLen())
	// fmt.Println("in CheckTaskDied func , len of c.mapTaskMap=", c.mapTaskMap.GetLen())

	// fmt.Println("in CheckTaskDied func , len of c.reduceTaskWaitingQueue=", c.reduceTaskWaitingQueue.GetLen())
	// fmt.Println("in CheckTaskDied func , len of c.reduceTaskRunningQueue=", c.reduceTaskRunningQueue.GetLen())
	// fmt.Println("in CheckTaskDied func , len of c.reduceTaskMap=", c.reduceTaskMap.GetLen())
	//c.mu.Lock()
	if c.mapTaskMap.GetLen() < c.nFiles{
		// in map task process
		retqueue := c.mapTaskRunningQueue.CheckTimeout(c.outtime)
		for i := 0; i < len(retqueue); i++{
			// fmt.Println("put outtime task back into waiting queue")
			c.mapTaskWaitingQueue.AddTail(retqueue[i])
		} 
	}else if c.reduceTaskMap.GetLen() < c.nReduce{
		// in reduce task process
		retqueue := c.reduceTaskRunningQueue.CheckTimeout(c.outtime)
		for i := 0; i < len(retqueue); i++{
			// fmt.Println("put outtime task back into waiting queue")
			c.reduceTaskWaitingQueue.AddTail(retqueue[i])
		} 
	}
	//c.mu.Unlock()
}

//
// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
//
func (c *Coordinator) Done() bool {
	ret := false

	// Your code here.
	if c.mapTaskMap.GetLen() == c.nFiles && c.reduceTaskMap.GetLen() == c.nReduce{
		ret = true
	}
	
	return ret
}

//
// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	//fmt.Println("init start ...")
	c := Coordinator{}
	c.outtime = 10
	c.nFiles = len(files)
	c.nReduce = nReduce
	c.mapTaskWaitingQueue = GetNewQueue()
	c.mapTaskRunningQueue = GetNewQueue()

	c.reduceTaskWaitingQueue = GetNewQueue()
	c.reduceTaskRunningQueue = GetNewQueue()

	c.mapTaskMap = GetRunningMap()
	c.reduceTaskMap = GetRunningMap()
	//fmt.Printf("init done! c.nFiles == %v, c.nReduce == %v\n", c.nFiles, c.nReduce)
	
	// Your code here.
	// generate mapTask queue
	//fmt.Println("generate mapTask queue")
	for i, file := range files{
		mapTask := &TaskState{
			FileName: file,
			X: i,
			Y: -1,
			NFiles: c.nFiles,
			NReduce: nReduce,
		}
		//fmt.Printf("i = %v\n", i)
		c.mapTaskWaitingQueue.AddTail(mapTask)
	}
	// generate reduceTask queue
	//fmt.Println("generate reduceTask queue")
	for i := 0; i < nReduce; i++{
		reduceTask := &TaskState{
			FileName: "",
			X: -1,
			Y: i,
			NFiles: c.nFiles,
			NReduce: nReduce,
		}
		c.reduceTaskWaitingQueue.AddTail(reduceTask)
	}
	//fmt.Println(c.mapTaskWaitingQueue)
	//fmt.Printf("init done! len of c.mapTaskWaitingQueue == %v\n", len(c.mapTaskWaitingQueue))
	//fmt.Printf("c = +%v\n", c)

	//fmt.Println("link start! ...")
	c.server()
	//fmt.Println("wait for done ...")
	
	go func(){
		for{
			if (c.Done()){
			
				time.Sleep(5*time.Second) //wait for all workers close
				fmt.Println("coordinator is done!")
				return
			}
			time.Sleep(2*time.Second) //wait for all workers close

		}
		
	}()

	go func(){
		for{
			c.CheckTaskDied()
			time.Sleep(2*time.Second)
		}
		
	}()
	
	return &c
}
