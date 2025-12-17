package mr
import "os"
import "strconv"


type TaskType int

const (
	MapTask TaskType = iota
	ReduceTask
	NoTask
	ExitTask
)

type Task struct {
	Type TaskType
	Id int
	FileName string
	NReduce int
	NMap int
}

type RequestTaskArgs struct {}

type RequestTaskReply struct {
	Task *Task
}

type ReportTaskArgs struct {
	TaskId int
	TaskType TaskType
}

type ReportTaskReply struct {}


// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
