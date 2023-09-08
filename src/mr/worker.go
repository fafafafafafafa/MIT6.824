package mr

import "fmt"
import "log"
import "net/rpc"
import "hash/fnv"
import "time"
import "os"
import "io/ioutil"
import "encoding/json"
import "sort"
//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

func splitKvaIntoFiles(kvas []KeyValue, fileid int,nReduce int){
	//fmt.Println("in splitKvaIntoFiles func...")
	//fmt.Printf("fileid = %v, nReduce = %v\n", fileid, nReduce)
	fileskva := make([][]KeyValue, nReduce)
	for _, kva := range kvas{
		idx := ihash(kva.Key)%nReduce
		fileskva[idx] = append(fileskva[idx], kva)
	}
	//fmt.Println("split done!")
	//fmt.Printf("kvas writes into %v files\n", nReduce)
	for i := 0; i < nReduce; i++{
		filename := fmt.Sprintf("temp-%v-%v", fileid, i)
		f, err := os.Create(filename)
		if err != nil{
			log.Fatalf("create temp file: %v failed!\n", filename)
			return
		}
		enc := json.NewEncoder(f)
		for _, kva := range fileskva[i]{
			err := enc.Encode(kva)
			if err != nil{
				log.Fatal("encode kva failed!")
				return 
			}
		}
	}

}
func getKvaFromFiles(fileid int,nReduceId int) []KeyValue{
	//fmt.Println("in getKvaFromFiles func...")
	kvas := make([]KeyValue, 0)
	filename := fmt.Sprintf("temp-%v-%v", fileid, nReduceId)
	//fmt.Printf("in getKvaFromFiles func, write to file : %v\n", filename)
	f, err := os.Open(filename)
	if err != nil{
		log.Fatalf("cannot open %v", filename)
	}
	dec := json.NewDecoder(f)
	for {
	  var kva KeyValue
	  if err := dec.Decode(&kva); err != nil {
		break
	  }
	  kvas = append(kvas, kva)
	}
	f.Close()
	return kvas
}
//
// main/mrworker.go calls this function.
//
// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.
	for{
		//fmt.Println("ask for task...")
		askreply := CallAskTask()
		//fmt.Println("task start...")
		//fmt.Printf("askreply = %+v\n", askreply)
		switch askreply.Method{
			case "map":{
				//fmt.Println("map method start ...")
				filename := askreply.FileName
				file, err := os.Open(filename)
				if err != nil {
					log.Fatalf("cannot open %v", filename)
				}
				content, err := ioutil.ReadAll(file)
				if err != nil {
					log.Fatalf("cannot read %v", filename)
				}
				file.Close()
				kvas := mapf(filename, string(content))
				splitKvaIntoFiles(kvas, askreply.X, askreply.NReduce)
				args := AskReply{
					StartTime: askreply.StartTime,
					FileName: askreply.FileName,
					X: askreply.X,
					Y: askreply.Y,
					NFiles: askreply.NFiles,
					NReduce: askreply.NReduce,
					Method: askreply.Method,
				}
				CallTaskDone(&args)
			}
			case "reduce":{
				nReduceId := askreply.Y
				kvas := make([]KeyValue, 0)
				for i := 0; i < askreply.NFiles; i++{
					kvas = append(kvas, getKvaFromFiles(i, nReduceId)...)	//why use ...
				}
				sort.Sort(ByKey(kvas))
	
				oname := fmt.Sprintf("mr-out-%v", nReduceId)
				ofile, _ := os.Create(oname)
	
				//
				// call Reduce on each distinct key in intermediate[],
				// and print the result to mr-out-0.
				//
				i := 0
				for i < len(kvas) {
					j := i + 1
					for j < len(kvas) && kvas[j].Key == kvas[i].Key {
						j++
					}
					values := []string{}
					for k := i; k < j; k++ {
						values = append(values, kvas[k].Value)
					}
					output := reducef(kvas[i].Key, values)
	
					// this is the correct format for each line of Reduce output.
					fmt.Fprintf(ofile, "%v %v\n", kvas[i].Key, output)
	
					i = j
				}
	
				ofile.Close()
				args := AskReply{
					StartTime: askreply.StartTime,
					FileName: askreply.FileName,
					X: askreply.X,
					Y: askreply.Y,
					NFiles: askreply.NFiles,
					NReduce: askreply.NReduce,
					Method: askreply.Method,
				}
				
				CallTaskDone(&args)
			}	
			case "wait":{
				time.Sleep(2*time.Second)
			}
			case "done":{
				fmt.Println("this worker is done!")
				return
			}
		}
	}

	// uncomment to send the Example RPC to the coordinator.
	// CallExample()

}

//
// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
//
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	call("Coordinator.Example", &args, &reply)

	// reply.Y should be 100.
	fmt.Printf("reply.Y %v\n", reply.Y)
}
func CallAskTask() AskReply{

	// declare an argument structure.
	args := AskArgs{}

	// declare a reply structure.
	reply := AskReply{}

	// send the RPC request, wait for the reply.
	call("Coordinator.AskTask", &args, &reply)
	//fmt.Printf("receive from CallAskTask func, reply.Method= %v\n", reply.Method)
	return reply
		 
}
func CallTaskDone(args *AskReply){
	
	// declare a reply structure.
	reply := AskArgs{}

	// send the RPC request, wait for the reply.
	call("Coordinator.TaskDone", args, &reply)

}
//
// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	//c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
