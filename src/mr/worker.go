package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"sort"
	"time"
)

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

type ByKey []KeyValue

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}


//
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	for {
		task := requestTask()

		switch task.Type {
		case MapTask:
			doMap(task, mapf)
			reportTask(task.Id, MapTask)
		case ReduceTask:
			doReduce(task, reducef)
			reportTask(task.Id, ReduceTask)
		case NoTask:
			time.Sleep(500 * time.Millisecond)
		case ExitTask:
			return
		default:
			log.Fatalf("Unknown task type: %v", task.Type)
		}
	}
}

// requestTask requests a task from the coordinator.
func requestTask() *Task {
	args := RequestTaskArgs{}
	reply := RequestTaskReply{}
	ok := call("Coordinator.RequestTask", &args, &reply)
	if !ok {
		// Coordinator has likely exited, so workers can exit too.
		os.Exit(0)
	}
	return reply.Task
}

// reportTask reports a completed task to the coordinator.
func reportTask(taskId int, taskType TaskType) {
	args := ReportTaskArgs{TaskId: taskId, TaskType: taskType}
	reply := ReportTaskReply{}
	call("Coordinator.ReportTask", &args, &reply)
}

// doMap performs a map task.
func doMap(task *Task, mapf func(string, string) []KeyValue) {
	content, err := ioutil.ReadFile(task.FileName)
	if err != nil {
		log.Fatalf("cannot read %v: %v", task.FileName, err)
	}

	kva := mapf(task.FileName, string(content))

	intermediate := make([][]KeyValue, task.NReduce)
	for _, kv := range kva {
		reduceIndex := ihash(kv.Key) % task.NReduce
		intermediate[reduceIndex] = append(intermediate[reduceIndex], kv)
	}

	for i := 0; i < task.NReduce; i++ {
		oname := fmt.Sprintf("mr-%d-%d", task.Id, i)
		ofile, _ := ioutil.TempFile("", oname+"-*")
		enc := json.NewEncoder(ofile)
		for _, kv := range intermediate[i] {
			err := enc.Encode(&kv)
			if err != nil {
				log.Fatalf("cannot encode json: %v", err)
			}
		}
		ofile.Close()
		os.Rename(ofile.Name(), oname)
	}
}

// doReduce performs a reduce task.
func doReduce(task *Task, reducef func(string, []string) string) {
	intermediate := []KeyValue{}
	for i := 0; i < task.NMap; i++ {
		iname := fmt.Sprintf("mr-%d-%d", i, task.Id)
		file, err := os.Open(iname)
		if err != nil {
			log.Printf("cannot open %v: %v", iname, err)
			continue
		}
		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			intermediate = append(intermediate, kv)
		}
		file.Close()
	}

	sort.Sort(ByKey(intermediate))

	oname := fmt.Sprintf("mr-out-%d", task.Id)
	ofile, _ := ioutil.TempFile("", oname+"-*")

	i := 0
	for i < len(intermediate) {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := reducef(intermediate[i].Key, values)

		fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)

		i = j
	}

	ofile.Close()
	os.Rename(ofile.Name(), oname)
}


//
// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
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
