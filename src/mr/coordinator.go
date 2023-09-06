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
	nFiles int // num of files 
	nReduce int // num of nReduce

	mapTaskWaitingQueue []TaskState
	mapTaskRunningQueue []TaskState

	reduceTaskWaitingQueue []TaskState
	reduceTaskRunningQueue []TaskState

	mapTaskMap *RunningMap
	reduceTaskMap *RunningMap
}
type TaskState struct {

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
	c.mu.Lock()
	defer c.mu.Unlock()
	
	//fmt.Println("in AskTask func , len of c.mapTaskWaitingQueue=", len(c.mapTaskWaitingQueue))
	//fmt.Println("in AskTask func , len of c.mapTaskMap=", c.mapTaskMap.GetLen())
	//fmt.Println("in AskTask func , len of c.reduceTaskWaitingQueue=", len(c.reduceTaskWaitingQueue))
	//fmt.Println("in AskTask func , len of c.reduceTaskMap=", c.reduceTaskMap.GetLen())
	if len(c.mapTaskWaitingQueue) > 0{
		l := len(c.mapTaskWaitingQueue)
		maptask := c.mapTaskWaitingQueue[l-1]
		c.mapTaskWaitingQueue = c.mapTaskWaitingQueue[:l-1]
		c.mapTaskRunningQueue = append(c.mapTaskRunningQueue, maptask)
		//fmt.Printf("in AskTask func, give mapTask , maptask=+%v\n", maptask)
		*reply = AskReply{
			StartTime: getNowTimeSecond(),
			FileName: maptask.FileName,
			X: maptask.X,
			Y: maptask.Y,
			NFiles: maptask.NFiles,
			NReduce: maptask.NReduce,
			Method: "map",
		}
		return nil
	}else if c.mapTaskMap.GetLen() < c.nFiles{
		*reply = AskReply{
			Method: "wait",
		}
		return nil
	}
	if len(c.reduceTaskWaitingQueue) > 0{
		l := len(c.reduceTaskWaitingQueue)
		reducetask := c.reduceTaskWaitingQueue[l-1]
		c.reduceTaskWaitingQueue = c.reduceTaskWaitingQueue[:l-1]
		c.reduceTaskRunningQueue = append(c.reduceTaskRunningQueue, reducetask)
		*reply = AskReply{
			StartTime: getNowTimeSecond(),
			FileName: reducetask.FileName,
			X: reducetask.X,
			Y: reducetask.Y,
			NFiles: reducetask.NFiles,
			NReduce: reducetask.NReduce,
			Method: "reduce",
		}
		return nil
	}else if c.reduceTaskMap.GetLen() < c.nReduce{
		*reply = AskReply{
			Method: "wait",
		}
		return nil
	}

	if len(c.mapTaskWaitingQueue)==0 && len(c.reduceTaskWaitingQueue)==0{
		*reply = AskReply{
			Method: "done",
		}
	}

	return nil
}
func (c *Coordinator) TaskDone(args *AskReply, reply *AskArgs) error{
	c.mu.Lock()
	defer c.mu.Unlock()
	switch args.Method{
		case "map":{
			maptask := TaskState{
				FileName: args.FileName,
				X: args.X,
				
				NFiles: args.NFiles,
				NReduce: args.NReduce,

			}
			c.mapTaskMap.Insert(maptask)
			
		}
		case "reduce":{
			reducetask := TaskState{
				Y: args.Y,
				NFiles: args.NFiles,
				NReduce: args.NReduce,

			}
			c.reduceTaskMap.Insert(reducetask)
			
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
	c.nFiles = len(files)
	c.nReduce = nReduce
	c.mapTaskWaitingQueue = make([]TaskState, 0)
	c.mapTaskRunningQueue = make([]TaskState, 0)

	c.reduceTaskWaitingQueue = make([]TaskState, 0)
	c.reduceTaskRunningQueue = make([]TaskState, 0)

	c.mapTaskMap = GetRunningMap()
	c.reduceTaskMap = GetRunningMap()
	//fmt.Printf("init done! c.nFiles == %v, c.nReduce == %v\n", c.nFiles, c.nReduce)
	
	// Your code here.
	// generate mapTask queue
	for i, file := range files{
		mapTask := TaskState{
			FileName: file,
			X: i,
			NFiles: c.nFiles,
			NReduce: nReduce,
		}
		c.mapTaskWaitingQueue = append(c.mapTaskWaitingQueue, mapTask)
	}
	// generate reduceTask queue
	for i := 0; i < nReduce; i++{
		reduceTask := TaskState{
			FileName: "",
			X: -1,
			Y: i,
			NFiles: c.nFiles,
			NReduce: nReduce,
		}
		c.reduceTaskWaitingQueue = append(c.reduceTaskWaitingQueue, reduceTask)
	}
	//fmt.Println(c.mapTaskWaitingQueue)
	//fmt.Printf("init done! len of c.mapTaskWaitingQueue == %v\n", len(c.mapTaskWaitingQueue))
	//fmt.Printf("c = +%v\n", c)

	//fmt.Println("link start! ...")
	c.server()
	//fmt.Println("wait for done ...")
	go func(){
		if (c.Done()){
			
			time.Sleep(5*time.Second) //wait for all workers close
			fmt.Println("coordinator is done!")
			return
		}
	}()
	
	
	return &c
}
