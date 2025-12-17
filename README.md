# Distributed MapReduce

A robust, fault-tolerant **distributed MapReduce system** implemented in **Go**, built according to the **MIT 6.5840 (Distributed Systems) – Lab 1** specification.

This project follows a **Coordinator–Worker** architecture where a central coordinator manages the job lifecycle, assigns tasks to distributed workers, and transparently handles failures.

---

## 🚀 Key Features

- **Distributed Architecture**  
  Clear separation between a central `Coordinator` and multiple `Workers`, communicating via RPC over UNIX domain sockets.

- **Fault Tolerance**  
  The Coordinator detects worker crashes or stalls using a **10-second timeout**. Tasks that are not completed within the timeout are automatically reassigned to healthy workers.

- **Atomic Writes**  
  Workers write results to temporary files and atomically rename them using `os.Rename` only after successful task completion, preventing partial or corrupted output.

- **Concurrency Control**  
  Uses `sync.Mutex` to safely manage shared Coordinator state (`Idle`, `InProgress`, `Completed`) across concurrent RPC requests.

- **Complete MapReduce Pipeline**  
  Implements the full MapReduce workflow:

  1. **Map Phase** – Hashes keys into `nReduce` partitions  
  2. **Intermediate Files** – Produces `mr-X-Y` JSON files  
  3. **Reduce Phase** – Sorts keys and aggregates final results

---

## 📂 Project Structure

```
src/
├── mr/                 # Core MapReduce implementation
│   ├── coordinator.go  # Task scheduling, state tracking, timeout logic
│   ├── worker.go       # Task execution, JSON I/O, atomic file handling
│   └── rpc.go          # RPC definitions and shared types
│
├── mrapps/             # User-defined Map/Reduce applications
│   └── wc.go           # Example: Word Count
│
└── main/               # Program entry points
    ├── mrcoordinator.go
    └── mrworker.go
```

---

## 🛠️ Usage

This system dynamically loads Map/Reduce applications as **Go plugins** at runtime.

### 1️⃣ Build the MapReduce Plugin

```bash
cd src/main
go build -buildmode=plugin ../mrapps/wc.go
```

---

### 2️⃣ Start the Coordinator

```bash
rm -f mr-out*
go run mrcoordinator.go pg-*.txt
```

---

### 3️⃣ Start the Workers

```bash
cd src/main
go run mrworker.go wc.so
```

---

### 4️⃣ View Results

```bash
cat mr-out-* | sort | more
```

---

## ⚙️ Design Details

### The Coordinator

- **Idle**
- **InProgress**
- **Completed**

Tasks exceeding 10 seconds are reassigned.

---

### The Worker

- **Map Task**: map → partition → atomic write
- **Reduce Task**: read → sort → reduce → output

---

## 📚 References

- *MapReduce: Simplified Data Processing on Large Clusters* — Google Research
- **MIT 6.5840 – Distributed Systems, Lab 1 (MapReduce)**
