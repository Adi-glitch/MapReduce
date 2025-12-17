package mr

import "log"
import "net"
import "os"
import "net/rpc"
import "net/http"
import "time"
import "sync"


type TaskStatus int

const (
	Idle TaskStatus = iota
	InProgress
	Completed
)

type TaskInfo struct{
	Status    TaskStatus
	StartTime time.Time
	Task      *Task
}

type Coordinator struct {
	mapTasks				[]TaskInfo
	reduceTasks 				[]TaskInfo
	nMap 					int
	nReduce 				int
	mapTasksCompleted 		int
	reduceTasksCompleted 	int
	mu 						sync.Mutex
}


// RequestTask handles RPC requests from workers for a task.
func (c *Coordinator) RequestTask(args *RequestTaskArgs, reply *RequestTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.mapTasksCompleted < c.nMap {
		for i := range c.mapTasks {
			if c.mapTasks[i].Status == Idle {
				c.mapTasks[i].Status = InProgress
				c.mapTasks[i].StartTime = time.Now()
				reply.Task = c.mapTasks[i].Task
				return nil
			}
		}
		// No idle map tasks available yet, tell worker to wait
		reply.Task = &Task{Type: NoTask}
		return nil
	}

	if c.reduceTasksCompleted < c.nReduce {
		for i := range c.reduceTasks {
			if c.reduceTasks[i].Status == Idle {
				c.reduceTasks[i].Status = InProgress
				c.reduceTasks[i].StartTime = time.Now()
				reply.Task = c.reduceTasks[i].Task
				return nil
			}
		}

		reply.Task = &Task{Type: NoTask}
		return nil
	}

	// All tasks are completed
	reply.Task = &Task{Type: ExitTask}
	return nil
}

// ReportTask handles RPCs from workers reporting task completion.
func (c *Coordinator) ReportTask(args *ReportTaskArgs, reply *ReportTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if args.TaskType == MapTask {
		if c.mapTasks[args.TaskId].Status == InProgress {
			c.mapTasks[args.TaskId].Status = Completed
			c.mapTasksCompleted++
		}
	} else if args.TaskType == ReduceTask {
		if c.reduceTasks[args.TaskId].Status == InProgress {
			c.reduceTasks[args.TaskId].Status = Completed
			c.reduceTasksCompleted++
		}
	}
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

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.reduceTasksCompleted ==  c.nReduce {
		ret = true
	}

	return ret
}

func (c *Coordinator) checkTimeouts(){

	for {
		time.Sleep(1 * time.Second)
		c.mu.Lock()
		for i := range c.mapTasks {
			if c.mapTasks[i].Status == InProgress && time.Since(c.mapTasks[i].StartTime) > 10 * time.Second {
				c.mapTasks[i].Status = Idle
			}
		}

		for i := range c.reduceTasks {
			if c.reduceTasks[i].Status == InProgress && time.Since(c.reduceTasks[i].StartTime) > 10 * time.Second {
				c.reduceTasks[i].Status = Idle
			}
		}

		c.mu.Unlock()
	}



}

//
// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		mapTasks :    make([]TaskInfo, len(files)),
		reduceTasks : make([]TaskInfo, nReduce),
		nMap : len(files),
		nReduce : nReduce,
		mapTasksCompleted : 0,
		reduceTasksCompleted : 0,
	}

	for i, filename := range files{
		c.mapTasks[i] = TaskInfo{
			Status: Idle,
			Task: &Task{
				Type: MapTask,
				Id: i,
				FileName: filename,
				NReduce: nReduce,
			},
		}
	} 

	for i := 0; i < nReduce; i++ {
		c.reduceTasks[i] = TaskInfo{
			Status: Idle,
			Task: &Task{
				Type: ReduceTask,
				Id: i,
				NMap: len(files),
			},
		}
	} 

	c.server()
	go c.checkTimeouts()
	return &c
}
