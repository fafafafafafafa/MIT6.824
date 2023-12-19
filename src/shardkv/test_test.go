package shardkv

import "6.824/porcupine"
import "6.824/models"
import "testing"
import "strconv"
import "time"
import "fmt"
import "sync/atomic"
import "sync"
import "math/rand"
import "io/ioutil"
import "flag"
import "os"
import "6.824/raft"
import "log"

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if true {
		log.Printf(format, a...)
	}
	return
}

const linearizabilityCheckTimeout = 1 * time.Second

func check(t *testing.T, ck *Clerk, key string, value string, mylog *raft.Mylog) {
	v := ck.Get(key)
	if v != value {
		mylog.DFprintf("fail: Get(%v): expected:\n%v\nreceived:\n%v\n", key, value, v)
		t.Fatalf("Get(%v): expected:\n%v\nreceived:\n%v", key, value, v)
	}
}

//
// test static 2-way sharding, without shard movement.
//

// func TestStaticShards(t *testing.T) {
// 	argList := flag.Args()
	
// 	arg := "1"
// 	if len(argList) == 1{
// 		arg = argList[0]
// 	}
// 	nums, err := strconv.Atoi(arg)
// 	if err != nil{
// 		DPrintf("%v \n", err)
// 	}
// 	for i := 0; i < nums; i++{
// 		dir := "./logs4B/TestStaticShards"
// 		if _, err := os.Stat(dir); os.IsNotExist(err) {
// 			// 必须分成两步：先创建文件夹、再修改权限
// 			os.Mkdir(dir, 0777) //0777也可以os.ModePerm
// 			os.Chmod(dir, 0777)
// 		}
// 		filename := fmt.Sprintf("%v/%v.txt", dir, i)
// 		w, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
		
// 		if err != nil{
// 			DPrintf("%v \n", err)
// 			return 
// 		}
// 		mylog := raft.Mylog{
// 			W: w,
// 			Debug: true,
// 		}
// 		err = w.Truncate(0)
// 		if err != nil {
// 			panic(err)
// 		}
// 		StaticShards(t, &mylog)
// 		w.Close()
// 	}

// }

// func StaticShards(t *testing.T, mylog *raft.Mylog) {
// 	fmt.Printf("Test: static shards ...\n")
// 	mylog.DFprintf("Test: static shards ...\n")
// 	mylog.GoroutineStack()

// 	cfg := make_config(t, 3, false, -1, mylog)
// 	defer cfg.cleanup()

// 	ck := cfg.makeClient() // make two shardctrler.Clerk : ck.sm, cfg.mck  .  why?
// 	mylog.DFprintf("*-------------join 0, 1----------------\n")
// 	cfg.join(0)
// 	cfg.join(1)

// 	n := 10
// 	ka := make([]string, n)
// 	va := make([]string, n)
// 	for i := 0; i < n; i++ {
// 		ka[i] = strconv.Itoa(i) // ensure multiple shards
// 		va[i] = randstring(20)
// 		ck.Put(ka[i], va[i])
// 	}
// 	for i := 0; i < n; i++ {
// 		check(t, ck, ka[i], va[i], mylog)
// 	}

// 	// make sure that the data really is sharded by
// 	// shutting down one shard and checking that some
// 	// Get()s don't succeed.
// 	mylog.DFprintf("*-------------ShutdownGroup 1----------------\n")

// 	cfg.ShutdownGroup(1)
// 	cfg.checklogs() // forbid snapshots

// 	ch := make(chan string)
// 	// group1 shard 0, 1, 2, 3, 4, 5 - ascii "2, 3, 4, 5, 6"
// 	xis := []int{7, 8, 9, 0, 1} // to avoid routines leak
// 	// for xi := 0; xi < n; xi++ {
// 	for _, xi := range xis{
// 		ck1 := cfg.makeClient() // only one call allowed per client
// 		go func(i int) {
// 			v := ck1.Get(ka[i])
// 			// if v == "timeout"{
// 			// 	return 
// 			// }
// 			if v != va[i] {
// 				ch <- fmt.Sprintf("Get(%v): expected:\n%v\nreceived:\n%v", ka[i], va[i], v)
// 			} else {
// 				ch <- ""
// 			}
// 		}(xi)
// 	}

// 	// wait a bit, only about half the Gets should succeed.
// 	ndone := 0
// 	done := false
// 	for done == false {
// 		select {
// 		case err := <-ch:
// 			if err != "" {
// 				mylog.DFprintf("fail: %v\n", err)
// 				t.Fatal(err)
// 			}
// 			ndone += 1
// 		case <-time.After(time.Second * 2):
// 			done = true
// 			break
// 		}
// 	}

// 	if ndone != 5 {
// 		mylog.DFprintf("fail: expected 5 completions with one shard dead; got %v\n", ndone)
// 		t.Fatalf("expected 5 completions with one shard dead; got %v\n", ndone)
// 	}

// 	// bring the crashed shard/group back to life.
// 	cfg.StartGroup(1)
// 	for i := 0; i < n; i++ {
// 		check(t, ck, ka[i], va[i], mylog)
// 	}

// 	fmt.Printf("  ... Passed\n")
// 	mylog.DFprintf("  ... Passed\n")
// }
func TestStaticShards(t *testing.T) {
	dir := "./logs4B/TestStaticShards"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// 必须分成两步：先创建文件夹、再修改权限
		os.Mkdir(dir, 0777) //0777也可以os.ModePerm
		os.Chmod(dir, 0777)
	}
	filename := fmt.Sprintf("%v/%v.txt", dir, 0)
	w, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
	
	if err != nil{
		DPrintf("%v \n", err)
		return 
	}
	mylog := &raft.Mylog{
		W: w,
		Debug: true,
	}
	err = w.Truncate(0)
	if err != nil {
		panic(err)
	}
	defer w.Close()

	fmt.Printf("Test: static shards ...\n")
	mylog.DFprintf("Test: static shards ...\n")
	mylog.GoroutineStack()

	cfg := make_config(t, 3, false, -1, mylog)
	defer cfg.cleanup()

	ck := cfg.makeClient() // make two shardctrler.Clerk : ck.sm, cfg.mck  .  why?
	mylog.DFprintf("*-------------join 0, 1----------------\n")
	cfg.join(0)
	cfg.join(1)

	n := 10
	ka := make([]string, n)
	va := make([]string, n)
	for i := 0; i < n; i++ {
		ka[i] = strconv.Itoa(i) // ensure multiple shards
		va[i] = randstring(20)
		ck.Put(ka[i], va[i])
	}
	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}

	// make sure that the data really is sharded by
	// shutting down one shard and checking that some
	// Get()s don't succeed.
	mylog.DFprintf("*-------------ShutdownGroup 1----------------\n")

	cfg.ShutdownGroup(1)
	cfg.checklogs() // forbid snapshots

	ch := make(chan string)
	// group1 shard 0, 1, 2, 3, 4, 5 - ascii "2, 3, 4, 5, 6"
	xis := []int{7, 8, 9, 0, 1} // to avoid routines leak
	// for xi := 0; xi < n; xi++ {
	for _, xi := range xis{
		ck1 := cfg.makeClient() // only one call allowed per client
		go func(i int) {
			v := ck1.Get(ka[i])
			// if v == "timeout"{
			// 	return 
			// }
			if v != va[i] {
				ch <- fmt.Sprintf("Get(%v): expected:\n%v\nreceived:\n%v", ka[i], va[i], v)
			} else {
				ch <- ""
			}
		}(xi)
	}

	// wait a bit, only about half the Gets should succeed.
	ndone := 0
	done := false
	for done == false {
		select {
		case err := <-ch:
			if err != "" {
				mylog.DFprintf("fail: %v\n", err)
				t.Fatal(err)
			}
			ndone += 1
		case <-time.After(time.Second * 2):
			done = true
			break
		}
	}

	if ndone != 5 {
		mylog.DFprintf("fail: expected 5 completions with one shard dead; got %v\n", ndone)
		t.Fatalf("expected 5 completions with one shard dead; got %v\n", ndone)
	}

	// bring the crashed shard/group back to life.
	cfg.StartGroup(1)
	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}

	fmt.Printf("  ... Passed\n")
	mylog.DFprintf("  ... Passed\n")
}

func TestJoinLeave(t *testing.T) {
	argList := flag.Args()
	
	arg := "1"
	if len(argList) == 1{
		arg = argList[0]
	}
	nums, err := strconv.Atoi(arg)
	if err != nil{
		DPrintf("%v \n", err)
	}
	for i := 0; i < nums; i++{
		dir := "./logs4B/TestJoinLeave"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// 必须分成两步：先创建文件夹、再修改权限
			os.Mkdir(dir, 0777) //0777也可以os.ModePerm
			os.Chmod(dir, 0777)
		}
		filename := fmt.Sprintf("%v/%v.txt", dir, i)
		w, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
		
		if err != nil{
			DPrintf("%v \n", err)
			return 
		}
		mylog := raft.Mylog{
			W: w,
			Debug: true,
		}
		err = w.Truncate(0)
		if err != nil {
			panic(err)
		}
		JoinLeave(t, &mylog)
		w.Close()
	}

}

func JoinLeave(t *testing.T, mylog *raft.Mylog) {
	fmt.Printf("Test: join then leave ...\n")
	mylog.DFprintf("Test: join then leave ...\n")
	mylog.GoroutineStack()

	cfg := make_config(t, 3, false, -1, mylog)
	defer cfg.cleanup()
	ck := cfg.makeClient()

	cfg.mylog.DFprintf("*-----------join 0------------\n")
	cfg.join(0)

	n := 10
	ka := make([]string, n)
	va := make([]string, n)
	for i := 0; i < n; i++ {
		ka[i] = strconv.Itoa(i) // ensure multiple shards
		va[i] = randstring(5)
		ck.Put(ka[i], va[i])
	}
	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}

	cfg.mylog.DFprintf("*-----------join 1------------\n")
	cfg.join(1)

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
		x := randstring(5)
		ck.Append(ka[i], x)
		va[i] += x
	}

	cfg.mylog.DFprintf("*-----------leave 0------------\n")
	cfg.leave(0)

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
		x := randstring(5)
		ck.Append(ka[i], x)
		va[i] += x
	}

	// allow time for shards to transfer.
	time.Sleep(1 * time.Second)

	cfg.checklogs()

	cfg.mylog.DFprintf("*-----------ShutdownGroup 0------------\n")
	cfg.ShutdownGroup(0)

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}

	fmt.Printf("  ... Passed\n")
	mylog.DFprintf("  ... Passed\n")
}
func TestSnapshot(t *testing.T) {
	argList := flag.Args()
	
	arg := "1"
	if len(argList) == 1{
		arg = argList[0]
	}
	nums, err := strconv.Atoi(arg)
	if err != nil{
		DPrintf("%v \n", err)
	}
	for i := 0; i < nums; i++{
		dir := "./logs4B/TestSnapshot"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// 必须分成两步：先创建文件夹、再修改权限
			os.Mkdir(dir, 0777) //0777也可以os.ModePerm
			os.Chmod(dir, 0777)
		}
		filename := fmt.Sprintf("%v/%v.txt", dir, i)
		w, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
		
		if err != nil{
			DPrintf("%v \n", err)
			return 
		}
		mylog := raft.Mylog{
			W: w,
			Debug: true,
		}
		err = w.Truncate(0)
		if err != nil {
			panic(err)
		}
		Snapshot(t, &mylog)
		w.Close()
	}

}

func Snapshot(t *testing.T, mylog *raft.Mylog) {
	fmt.Printf("Test: snapshots, join, and leave ...\n")
	mylog.DFprintf("Test: snapshots, join, and leave ...\n")
	mylog.GoroutineStack()

	cfg := make_config(t, 3, false, 1000, mylog)
	defer cfg.cleanup()

	ck := cfg.makeClient()
	cfg.mylog.DFprintf("*-----------join 0------------\n")
	cfg.join(0)

	n := 30
	ka := make([]string, n)
	va := make([]string, n)
	for i := 0; i < n; i++ {
		ka[i] = strconv.Itoa(i) // ensure multiple shards
		va[i] = randstring(20)
		ck.Put(ka[i], va[i])
	}
	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}
	cfg.mylog.DFprintf("*-----------join 1 2, leave 0------------\n")

	cfg.join(1)
	cfg.join(2)
	cfg.leave(0)

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
		x := randstring(20)
		ck.Append(ka[i], x)
		va[i] += x
	}

	cfg.mylog.DFprintf("*-----------join 0, leave 1------------\n")
	cfg.leave(1)
	cfg.join(0)

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
		x := randstring(20)
		ck.Append(ka[i], x)
		va[i] += x
	}

	time.Sleep(1 * time.Second)

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}

	time.Sleep(1 * time.Second)

	cfg.checklogs()
	cfg.mylog.DFprintf("*-----------shutdown and restart all------------\n")

	cfg.ShutdownGroup(0)
	cfg.ShutdownGroup(1)
	cfg.ShutdownGroup(2)

	cfg.StartGroup(0)
	cfg.StartGroup(1)
	cfg.StartGroup(2)

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}

	fmt.Printf("  ... Passed\n")
	mylog.DFprintf("  ... Passed\n")

}
func TestMissChange(t *testing.T) {
	argList := flag.Args()
	
	arg := "1"
	if len(argList) == 1{
		arg = argList[0]
	}
	nums, err := strconv.Atoi(arg)
	if err != nil{
		DPrintf("%v \n", err)
	}
	for i := 0; i < nums; i++{
		dir := "./logs4B/TestMissChange"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// 必须分成两步：先创建文件夹、再修改权限
			os.Mkdir(dir, 0777) //0777也可以os.ModePerm
			os.Chmod(dir, 0777)
		}
		filename := fmt.Sprintf("%v/%v.txt", dir, i)
		w, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
		
		if err != nil{
			DPrintf("%v \n", err)
			return 
		}
		mylog := raft.Mylog{
			W: w,
			Debug: true,
		}
		err = w.Truncate(0)
		if err != nil {
			panic(err)
		}
		MissChange(t, &mylog)
		w.Close()
	}

}
func MissChange(t *testing.T, mylog *raft.Mylog) {
	fmt.Printf("Test: servers miss configuration changes...\n")
	mylog.DFprintf("Test: servers miss configuration changes...\n")
	mylog.GoroutineStack()

	cfg := make_config(t, 3, false, 1000, mylog)
	defer cfg.cleanup()

	ck := cfg.makeClient()

	cfg.join(0)

	n := 10
	ka := make([]string, n)
	va := make([]string, n)
	for i := 0; i < n; i++ {
		ka[i] = strconv.Itoa(i) // ensure multiple shards
		va[i] = randstring(20)
		ck.Put(ka[i], va[i])
	}
	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}

	cfg.join(1)

	cfg.ShutdownServer(0, 0)
	cfg.ShutdownServer(1, 0)
	cfg.ShutdownServer(2, 0)

	cfg.join(2)
	cfg.leave(1)
	cfg.leave(0)

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
		x := randstring(20)
		ck.Append(ka[i], x)
		va[i] += x
	}

	cfg.join(1)

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
		x := randstring(20)
		ck.Append(ka[i], x)
		va[i] += x
	}

	cfg.StartServer(0, 0)
	cfg.StartServer(1, 0)
	cfg.StartServer(2, 0)

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
		x := randstring(20)
		ck.Append(ka[i], x)
		va[i] += x
	}

	time.Sleep(2 * time.Second)

	cfg.ShutdownServer(0, 1)
	cfg.ShutdownServer(1, 1)
	cfg.ShutdownServer(2, 1)

	cfg.join(0)
	cfg.leave(2)

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
		x := randstring(20)
		ck.Append(ka[i], x)
		va[i] += x
	}

	cfg.StartServer(0, 1)
	cfg.StartServer(1, 1)
	cfg.StartServer(2, 1)

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}

	fmt.Printf("  ... Passed\n")
	mylog.DFprintf("  ... Passed\n")

}
func TestConcurrent1(t *testing.T) {
	argList := flag.Args()
	
	arg := "1"
	if len(argList) == 1{
		arg = argList[0]
	}
	nums, err := strconv.Atoi(arg)
	if err != nil{
		DPrintf("%v \n", err)
	}
	for i := 0; i < nums; i++{
		dir := "./logs4B/TestConcurrent1"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// 必须分成两步：先创建文件夹、再修改权限
			os.Mkdir(dir, 0777) //0777也可以os.ModePerm
			os.Chmod(dir, 0777)
		}
		filename := fmt.Sprintf("%v/%v.txt", dir, i)
		w, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
		
		if err != nil{
			DPrintf("%v \n", err)
			return 
		}
		mylog := raft.Mylog{
			W: w,
			Debug: true,
		}
		err = w.Truncate(0)
		if err != nil {
			panic(err)
		}
		Concurrent1(t, &mylog)
		w.Close()
	}

}
func Concurrent1(t *testing.T, mylog *raft.Mylog) {
	fmt.Printf("Test: concurrent puts and configuration changes...\n")
	mylog.DFprintf("Test: concurrent puts and configuration changes...\n")
	mylog.GoroutineStack()

	cfg := make_config(t, 3, false, 100, mylog)
	defer cfg.cleanup()

	ck := cfg.makeClient()

	cfg.join(0)

	n := 10
	ka := make([]string, n)
	va := make([]string, n)
	for i := 0; i < n; i++ {
		ka[i] = strconv.Itoa(i) // ensure multiple shards
		va[i] = randstring(5)
		ck.Put(ka[i], va[i])
	}

	var done int32
	ch := make(chan bool)

	ff := func(i int) {
		defer func() { ch <- true }()
		ck1 := cfg.makeClient()
		for atomic.LoadInt32(&done) == 0 {
			x := randstring(5)
			ck1.Append(ka[i], x)
			va[i] += x
			time.Sleep(10 * time.Millisecond)
		}
	}

	for i := 0; i < n; i++ {
		go ff(i)
	}

	time.Sleep(150 * time.Millisecond)
	cfg.join(1)
	time.Sleep(500 * time.Millisecond)
	cfg.join(2)
	time.Sleep(500 * time.Millisecond)
	cfg.leave(0)

	cfg.ShutdownGroup(0)
	time.Sleep(100 * time.Millisecond)
	cfg.ShutdownGroup(1)
	time.Sleep(100 * time.Millisecond)
	cfg.ShutdownGroup(2)

	cfg.leave(2)

	time.Sleep(100 * time.Millisecond)
	cfg.StartGroup(0)
	cfg.StartGroup(1)
	cfg.StartGroup(2)

	time.Sleep(100 * time.Millisecond)
	cfg.join(0)
	cfg.leave(1)
	time.Sleep(500 * time.Millisecond)
	cfg.join(1)

	time.Sleep(1 * time.Second)

	atomic.StoreInt32(&done, 1)
	for i := 0; i < n; i++ {
		<-ch
	}

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}

	fmt.Printf("  ... Passed\n")
	mylog.DFprintf("  ... Passed\n")

}

//
// this tests the various sources from which a re-starting
// group might need to fetch shard contents.
//
func TestConcurrent2(t *testing.T) {
	argList := flag.Args()
	
	arg := "1"
	if len(argList) == 1{
		arg = argList[0]
	}
	nums, err := strconv.Atoi(arg)
	if err != nil{
		DPrintf("%v \n", err)
	}
	for i := 0; i < nums; i++{
		dir := "./logs4B/TestConcurrent2"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// 必须分成两步：先创建文件夹、再修改权限
			os.Mkdir(dir, 0777) //0777也可以os.ModePerm
			os.Chmod(dir, 0777)
		}
		filename := fmt.Sprintf("%v/%v.txt", dir, i)
		w, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
		
		if err != nil{
			DPrintf("%v \n", err)
			return 
		}
		mylog := raft.Mylog{
			W: w,
			Debug: true,
		}
		err = w.Truncate(0)
		if err != nil {
			panic(err)
		}
		Concurrent2(t, &mylog)
		w.Close()
	}

}
func Concurrent2(t *testing.T, mylog *raft.Mylog) {
	fmt.Printf("Test: more concurrent puts and configuration changes...\n")
	mylog.DFprintf("Test: more concurrent puts and configuration changes...\n")
	mylog.GoroutineStack()

	cfg := make_config(t, 3, false, -1, mylog)
	defer cfg.cleanup()

	ck := cfg.makeClient()

	cfg.join(1)
	cfg.join(0)
	cfg.join(2)

	n := 10
	ka := make([]string, n)
	va := make([]string, n)
	for i := 0; i < n; i++ {
		ka[i] = strconv.Itoa(i) // ensure multiple shards
		va[i] = randstring(1)
		ck.Put(ka[i], va[i])
	}

	var done int32
	ch := make(chan bool)

	ff := func(i int, ck1 *Clerk) {
		defer func() { ch <- true }()
		for atomic.LoadInt32(&done) == 0 {
			x := randstring(1)
			ck1.Append(ka[i], x)
			va[i] += x
			time.Sleep(50 * time.Millisecond)
		}
	}

	for i := 0; i < n; i++ {
		ck1 := cfg.makeClient()
		go ff(i, ck1)
	}

	cfg.leave(0)
	cfg.leave(2)
	time.Sleep(3000 * time.Millisecond)
	cfg.join(0)
	cfg.join(2)
	cfg.leave(1)
	time.Sleep(3000 * time.Millisecond)
	cfg.join(1)
	cfg.leave(0)
	cfg.leave(2)
	time.Sleep(3000 * time.Millisecond)

	cfg.ShutdownGroup(1)
	cfg.ShutdownGroup(2)
	time.Sleep(1000 * time.Millisecond)
	cfg.StartGroup(1)
	cfg.StartGroup(2)

	time.Sleep(2 * time.Second)

	atomic.StoreInt32(&done, 1)
	for i := 0; i < n; i++ {
		<-ch
	}

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}

	fmt.Printf("  ... Passed\n")
	mylog.DFprintf("  ... Passed\n")

}
func TestConcurrent3(t *testing.T) {
	argList := flag.Args()
	
	arg := "1"
	if len(argList) == 1{
		arg = argList[0]
	}
	nums, err := strconv.Atoi(arg)
	if err != nil{
		DPrintf("%v \n", err)
	}
	for i := 0; i < nums; i++{
		dir := "./logs4B/TestConcurrent3"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// 必须分成两步：先创建文件夹、再修改权限
			os.Mkdir(dir, 0777) //0777也可以os.ModePerm
			os.Chmod(dir, 0777)
		}
		filename := fmt.Sprintf("%v/%v.txt", dir, i)
		w, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
		
		if err != nil{
			DPrintf("%v \n", err)
			return 
		}
		mylog := raft.Mylog{
			W: w,
			Debug: true,
		}
		err = w.Truncate(0)
		if err != nil {
			panic(err)
		}
		Concurrent3(t, &mylog)
		w.Close()
	}

}
func Concurrent3(t *testing.T, mylog *raft.Mylog) {
	fmt.Printf("Test: concurrent configuration change and restart...\n")
	mylog.DFprintf("Test: concurrent configuration change and restart...\n")
	mylog.GoroutineStack()

	cfg := make_config(t, 3, false, 300, mylog)
	defer cfg.cleanup()

	ck := cfg.makeClient()

	cfg.join(0)

	n := 10
	ka := make([]string, n)
	va := make([]string, n)
	for i := 0; i < n; i++ {
		ka[i] = strconv.Itoa(i)
		va[i] = randstring(1)
		ck.Put(ka[i], va[i])
	}

	var done int32
	ch := make(chan bool)

	ff := func(i int, ck1 *Clerk) {
		defer func() { ch <- true }()
		for atomic.LoadInt32(&done) == 0 {
			x := randstring(1)
			ck1.Append(ka[i], x)
			va[i] += x
		}
	}

	for i := 0; i < n; i++ {
		ck1 := cfg.makeClient()
		go ff(i, ck1)
	}

	t0 := time.Now()
	for time.Since(t0) < 12*time.Second {
		cfg.join(2)
		cfg.join(1)
		time.Sleep(time.Duration(rand.Int()%900) * time.Millisecond)
		cfg.ShutdownGroup(0)
		cfg.ShutdownGroup(1)
		cfg.ShutdownGroup(2)
		cfg.StartGroup(0)
		cfg.StartGroup(1)
		cfg.StartGroup(2)

		time.Sleep(time.Duration(rand.Int()%900) * time.Millisecond)
		cfg.leave(1)
		cfg.leave(2)
		time.Sleep(time.Duration(rand.Int()%900) * time.Millisecond)
	}

	time.Sleep(2 * time.Second)

	atomic.StoreInt32(&done, 1)
	for i := 0; i < n; i++ {
		<-ch
	}

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}

	fmt.Printf("  ... Passed\n")
	mylog.DFprintf("  ... Passed\n")

}
func TestUnreliable1(t *testing.T) {
	argList := flag.Args()
	
	arg := "1"
	if len(argList) == 1{
		arg = argList[0]
	}
	nums, err := strconv.Atoi(arg)
	if err != nil{
		DPrintf("%v \n", err)
	}
	for i := 0; i < nums; i++{
		dir := "./logs4B/TestUnreliable1"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// 必须分成两步：先创建文件夹、再修改权限
			os.Mkdir(dir, 0777) //0777也可以os.ModePerm
			os.Chmod(dir, 0777)
		}
		filename := fmt.Sprintf("%v/%v.txt", dir, i)
		w, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
		
		if err != nil{
			DPrintf("%v \n", err)
			return 
		}
		mylog := raft.Mylog{
			W: w,
			Debug: true,
		}
		err = w.Truncate(0)
		if err != nil {
			panic(err)
		}
		Unreliable1(t, &mylog)
		w.Close()
	}

}
func Unreliable1(t *testing.T, mylog *raft.Mylog) {
	fmt.Printf("Test: unreliable 1...\n")
	mylog.DFprintf("Test: unreliable 1...\n")
	mylog.GoroutineStack()

	cfg := make_config(t, 3, true, 100, mylog)
	defer cfg.cleanup()

	ck := cfg.makeClient()

	cfg.join(0)

	n := 10
	ka := make([]string, n)
	va := make([]string, n)
	for i := 0; i < n; i++ {
		ka[i] = strconv.Itoa(i) // ensure multiple shards
		va[i] = randstring(5)
		ck.Put(ka[i], va[i])
	}

	cfg.join(1)
	cfg.join(2)
	cfg.leave(0)

	for ii := 0; ii < n*2; ii++ {
		i := ii % n
		check(t, ck, ka[i], va[i], mylog)
		x := randstring(5)
		ck.Append(ka[i], x)
		va[i] += x
	}

	cfg.join(0)
	cfg.leave(1)

	for ii := 0; ii < n*2; ii++ {
		i := ii % n
		check(t, ck, ka[i], va[i], mylog)
	}

	fmt.Printf("  ... Passed\n")
	mylog.DFprintf("  ... Passed\n")

}
func TestUnreliable2(t *testing.T) {
	argList := flag.Args()
	
	arg := "1"
	if len(argList) == 1{
		arg = argList[0]
	}
	nums, err := strconv.Atoi(arg)
	if err != nil{
		DPrintf("%v \n", err)
	}
	for i := 0; i < nums; i++{
		dir := "./logs4B/TestUnreliable2"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// 必须分成两步：先创建文件夹、再修改权限
			os.Mkdir(dir, 0777) //0777也可以os.ModePerm
			os.Chmod(dir, 0777)
		}
		filename := fmt.Sprintf("%v/%v.txt", dir, i)
		w, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
		
		if err != nil{
			DPrintf("%v \n", err)
			return 
		}
		mylog := raft.Mylog{
			W: w,
			Debug: true,
		}
		err = w.Truncate(0)
		if err != nil {
			panic(err)
		}
		Unreliable2(t, &mylog)
		w.Close()
	}

}
func Unreliable2(t *testing.T, mylog *raft.Mylog) {
	fmt.Printf("Test: unreliable 2...\n")
	mylog.DFprintf("Test: unreliable 2...\n")
	mylog.GoroutineStack()

	cfg := make_config(t, 3, true, 100, mylog)
	defer cfg.cleanup()

	ck := cfg.makeClient()

	cfg.join(0)

	n := 10
	ka := make([]string, n)
	va := make([]string, n)
	for i := 0; i < n; i++ {
		ka[i] = strconv.Itoa(i) // ensure multiple shards
		va[i] = randstring(5)
		ck.Put(ka[i], va[i])
	}

	var done int32
	ch := make(chan bool)

	ff := func(i int) {
		defer func() { ch <- true }()
		ck1 := cfg.makeClient()
		for atomic.LoadInt32(&done) == 0 {
			x := randstring(5)
			ck1.Append(ka[i], x)
			va[i] += x
		}
	}

	for i := 0; i < n; i++ {
		go ff(i)
	}

	time.Sleep(150 * time.Millisecond)
	cfg.join(1)
	time.Sleep(500 * time.Millisecond)
	cfg.join(2)
	time.Sleep(500 * time.Millisecond)
	cfg.leave(0)
	time.Sleep(500 * time.Millisecond)
	cfg.leave(1)
	time.Sleep(500 * time.Millisecond)
	cfg.join(1)
	cfg.join(0)

	time.Sleep(2 * time.Second)

	atomic.StoreInt32(&done, 1)
	cfg.net.Reliable(true)
	for i := 0; i < n; i++ {
		<-ch
	}

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}

	fmt.Printf("  ... Passed\n")
	mylog.DFprintf("  ... Passed\n")

}
func TestUnreliable3(t *testing.T) {
	argList := flag.Args()
	
	arg := "1"
	if len(argList) == 1{
		arg = argList[0]
	}
	nums, err := strconv.Atoi(arg)
	if err != nil{
		DPrintf("%v \n", err)
	}
	for i := 0; i < nums; i++{
		dir := "./logs4B/TestUnreliable3"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// 必须分成两步：先创建文件夹、再修改权限
			os.Mkdir(dir, 0777) //0777也可以os.ModePerm
			os.Chmod(dir, 0777)
		}
		filename := fmt.Sprintf("%v/%v.txt", dir, i)
		w, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
		
		if err != nil{
			DPrintf("%v \n", err)
			return 
		}
		mylog := raft.Mylog{
			W: w,
			Debug: true,
		}
		err = w.Truncate(0)
		if err != nil {
			panic(err)
		}
		Unreliable3(t, &mylog)
		w.Close()
	}

}
func Unreliable3(t *testing.T, mylog *raft.Mylog) {
	fmt.Printf("Test: unreliable 3...\n")
	mylog.DFprintf("Test: unreliable 3...\n")
	mylog.GoroutineStack()

	cfg := make_config(t, 3, true, 100, mylog)
	defer cfg.cleanup()

	begin := time.Now()
	var operations []porcupine.Operation
	var opMu sync.Mutex

	ck := cfg.makeClient()

	cfg.join(0)

	n := 10
	ka := make([]string, n)
	va := make([]string, n)
	for i := 0; i < n; i++ {
		ka[i] = strconv.Itoa(i) // ensure multiple shards
		va[i] = randstring(5)
		start := int64(time.Since(begin))
		ck.Put(ka[i], va[i])
		end := int64(time.Since(begin))
		inp := models.KvInput{Op: 1, Key: ka[i], Value: va[i]}
		var out models.KvOutput
		op := porcupine.Operation{Input: inp, Call: start, Output: out, Return: end, ClientId: 0}
		operations = append(operations, op)
	}

	var done int32
	ch := make(chan bool)

	ff := func(i int) {
		defer func() { ch <- true }()
		ck1 := cfg.makeClient()
		for atomic.LoadInt32(&done) == 0 {
			ki := rand.Int() % n
			nv := randstring(5)
			var inp models.KvInput
			var out models.KvOutput
			start := int64(time.Since(begin))
			if (rand.Int() % 1000) < 500 {
				ck1.Append(ka[ki], nv)
				inp = models.KvInput{Op: 2, Key: ka[ki], Value: nv}
			} else if (rand.Int() % 1000) < 100 {
				ck1.Put(ka[ki], nv)
				inp = models.KvInput{Op: 1, Key: ka[ki], Value: nv}
			} else {
				v := ck1.Get(ka[ki])
				inp = models.KvInput{Op: 0, Key: ka[ki]}
				out = models.KvOutput{Value: v}
			}
			end := int64(time.Since(begin))
			op := porcupine.Operation{Input: inp, Call: start, Output: out, Return: end, ClientId: i}
			opMu.Lock()
			operations = append(operations, op)
			opMu.Unlock()
		}
	}

	for i := 0; i < n; i++ {
		go ff(i)
	}

	time.Sleep(150 * time.Millisecond)
	cfg.join(1)
	time.Sleep(500 * time.Millisecond)
	cfg.join(2)
	time.Sleep(500 * time.Millisecond)
	cfg.leave(0)
	time.Sleep(500 * time.Millisecond)
	cfg.leave(1)
	time.Sleep(500 * time.Millisecond)
	cfg.join(1)
	cfg.join(0)

	time.Sleep(2 * time.Second)

	atomic.StoreInt32(&done, 1)
	cfg.net.Reliable(true)
	for i := 0; i < n; i++ {
		<-ch
	}

	res, info := porcupine.CheckOperationsVerbose(models.KvModel, operations, linearizabilityCheckTimeout)
	if res == porcupine.Illegal {
		file, err := ioutil.TempFile("", "*.html")
		if err != nil {
			fmt.Printf("info: failed to create temp file for visualization\n")
			mylog.DFprintf("info: failed to create temp file for visualization\n")

		} else {
			err = porcupine.Visualize(models.KvModel, info, file)
			if err != nil {
				fmt.Printf("info: failed to write history visualization to %s\n", file.Name())
				mylog.DFprintf("info: failed to write history visualization to %s\n", file.Name())

			} else {
				fmt.Printf("info: wrote history visualization to %s\n", file.Name())
				mylog.DFprintf("info: wrote history visualization to %s\n", file.Name())

			}
		}
		mylog.DFprintf("history is not linearizable\n")
		t.Fatal("history is not linearizable")
	} else if res == porcupine.Unknown {
		fmt.Println("info: linearizability check timed out, assuming history is ok")
	}

	fmt.Printf("  ... Passed\n")
	mylog.DFprintf("  ... Passed\n")

}

//
// optional test to see whether servers are deleting
// shards for which they are no longer responsible.
//
func TestChallenge1Delete(t *testing.T) {
	argList := flag.Args()
	
	arg := "1"
	if len(argList) == 1{
		arg = argList[0]
	}
	nums, err := strconv.Atoi(arg)
	if err != nil{
		DPrintf("%v \n", err)
	}
	for i := 0; i < nums; i++{
		dir := "./logs4B/TestChallenge1Delete"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// 必须分成两步：先创建文件夹、再修改权限
			os.Mkdir(dir, 0777) //0777也可以os.ModePerm
			os.Chmod(dir, 0777)
		}
		filename := fmt.Sprintf("%v/%v.txt", dir, i)
		w, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
		
		if err != nil{
			DPrintf("%v \n", err)
			return 
		}
		mylog := raft.Mylog{
			W: w,
			Debug: true,
		}
		err = w.Truncate(0)
		if err != nil {
			panic(err)
		}
		Challenge1Delete(t, &mylog)
		w.Close()
	}

}
func Challenge1Delete(t *testing.T, mylog *raft.Mylog) {
	fmt.Printf("Test: shard deletion (challenge 1) ...\n")
	mylog.DFprintf("Test: shard deletion (challenge 1) ...\n")
	mylog.GoroutineStack()

	// "1" means force snapshot after every log entry.
	cfg := make_config(t, 3, false, 1, mylog)
	defer cfg.cleanup()

	ck := cfg.makeClient()

	cfg.join(0)

	// 30,000 bytes of total values.
	n := 30
	ka := make([]string, n)
	va := make([]string, n)
	for i := 0; i < n; i++ {
		ka[i] = strconv.Itoa(i)
		va[i] = randstring(1000)
		ck.Put(ka[i], va[i])
	}
	for i := 0; i < 3; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}

	for iters := 0; iters < 2; iters++ {
		cfg.join(1)
		cfg.leave(0)
		cfg.join(2)
		time.Sleep(3 * time.Second)
		for i := 0; i < 3; i++ {
			check(t, ck, ka[i], va[i], mylog)
		}
		cfg.leave(1)
		cfg.join(0)
		cfg.leave(2)
		time.Sleep(3 * time.Second)
		for i := 0; i < 3; i++ {
			check(t, ck, ka[i], va[i], mylog)
		}
	}

	cfg.join(1)
	cfg.join(2)
	time.Sleep(1 * time.Second)
	for i := 0; i < 3; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}
	time.Sleep(1 * time.Second)
	for i := 0; i < 3; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}
	time.Sleep(1 * time.Second)
	for i := 0; i < 3; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}

	total := 0
	for gi := 0; gi < cfg.ngroups; gi++ {
		for i := 0; i < cfg.n; i++ {
			raft := cfg.groups[gi].saved[i].RaftStateSize()
			snap := len(cfg.groups[gi].saved[i].ReadSnapshot())
			total += raft + snap
		}
	}

	// 27 keys should be stored once.
	// 3 keys should also be stored in client dup tables.
	// everything on 3 replicas.
	// plus slop.
	expected := 3 * (((n - 3) * 1000) + 2*3*1000 + 6000)
	if total > expected {
		mylog.DFprintf("fail: snapshot + persisted Raft state are too big: %v > %v\n", total, expected)
		t.Fatalf("snapshot + persisted Raft state are too big: %v > %v\n", total, expected)
	}

	for i := 0; i < n; i++ {
		check(t, ck, ka[i], va[i], mylog)
	}

	fmt.Printf("  ... Passed\n")
	mylog.DFprintf("  ... Passed\n")

}

//
// optional test to see whether servers can handle
// shards that are not affected by a config change
// while the config change is underway
//
func TestChallenge2Unaffected(t *testing.T) {
	argList := flag.Args()
	
	arg := "1"
	if len(argList) == 1{
		arg = argList[0]
	}
	nums, err := strconv.Atoi(arg)
	if err != nil{
		DPrintf("%v \n", err)
	}
	for i := 0; i < nums; i++{
		dir := "./logs4B/TestChallenge2Unaffected"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// 必须分成两步：先创建文件夹、再修改权限
			os.Mkdir(dir, 0777) //0777也可以os.ModePerm
			os.Chmod(dir, 0777)
		}
		filename := fmt.Sprintf("%v/%v.txt", dir, i)
		w, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
		
		if err != nil{
			DPrintf("%v \n", err)
			return 
		}
		mylog := raft.Mylog{
			W: w,
			Debug: true,
		}
		err = w.Truncate(0)
		if err != nil {
			panic(err)
		}
		Challenge2Unaffected(t, &mylog)
		w.Close()
	}

}
func Challenge2Unaffected(t *testing.T, mylog *raft.Mylog) {
	fmt.Printf("Test: unaffected shard access (challenge 2) ...\n")
	mylog.DFprintf("Test: unaffected shard access (challenge 2) ...\n")
	mylog.GoroutineStack()

	cfg := make_config(t, 3, true, 100, mylog)
	defer cfg.cleanup()

	ck := cfg.makeClient()

	// JOIN 100
	cfg.join(0)

	// Do a bunch of puts to keys in all shards
	n := 10
	ka := make([]string, n)
	va := make([]string, n)
	for i := 0; i < n; i++ {
		ka[i] = strconv.Itoa(i) // ensure multiple shards
		va[i] = "100"
		ck.Put(ka[i], va[i])
	}

	// JOIN 101
	cfg.join(1)

	// QUERY to find shards now owned by 101
	c := cfg.mck.Query(-1)
	owned := make(map[int]bool, n)
	for s, gid := range c.Shards {
		owned[s] = gid == cfg.groups[1].gid
	}

	// Wait for migration to new config to complete, and for clients to
	// start using this updated config. Gets to any key k such that
	// owned[shard(k)] == true should now be served by group 101.
	<-time.After(1 * time.Second)
	for i := 0; i < n; i++ {
		if owned[i] {
			va[i] = "101"
			ck.Put(ka[i], va[i])
		}
	}

	// KILL 100
	cfg.ShutdownGroup(0)

	// LEAVE 100
	// 101 doesn't get a chance to migrate things previously owned by 100
	cfg.leave(0)

	// Wait to make sure clients see new config
	<-time.After(1 * time.Second)

	// And finally: check that gets/puts for 101-owned keys still complete
	for i := 0; i < n; i++ {
		shard := int(ka[i][0]) % 10
		if owned[shard] {
			check(t, ck, ka[i], va[i], mylog)
			ck.Put(ka[i], va[i]+"-1")
			check(t, ck, ka[i], va[i]+"-1", mylog)
		}
	}

	fmt.Printf("  ... Passed\n")
	mylog.DFprintf("  ... Passed\n")

}

//
// optional test to see whether servers can handle operations on shards that
// have been received as a part of a config migration when the entire migration
// has not yet completed.
//
func TestChallenge2Partial(t *testing.T) {
	argList := flag.Args()
	
	arg := "1"
	if len(argList) == 1{
		arg = argList[0]
	}
	nums, err := strconv.Atoi(arg)
	if err != nil{
		DPrintf("%v \n", err)
	}
	for i := 0; i < nums; i++{
		dir := "./logs4B/TestChallenge2Partial"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// 必须分成两步：先创建文件夹、再修改权限
			os.Mkdir(dir, 0777) //0777也可以os.ModePerm
			os.Chmod(dir, 0777)
		}
		filename := fmt.Sprintf("%v/%v.txt", dir, i)
		w, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
		
		if err != nil{
			DPrintf("%v \n", err)
			return 
		}
		mylog := raft.Mylog{
			W: w,
			Debug: true,
		}
		err = w.Truncate(0)
		if err != nil {
			panic(err)
		}
		Challenge2Partial(t, &mylog)
		w.Close()
	}

}
func Challenge2Partial(t *testing.T, mylog *raft.Mylog) {
	fmt.Printf("Test: partial migration shard access (challenge 2) ...\n")
	mylog.DFprintf("Test: partial migration shard access (challenge 2) ...\n")
	mylog.GoroutineStack()

	cfg := make_config(t, 3, true, 100, mylog)
	defer cfg.cleanup()

	ck := cfg.makeClient()

	// JOIN 100 + 101 + 102
	cfg.joinm([]int{0, 1, 2})

	// Give the implementation some time to reconfigure
	<-time.After(1 * time.Second)

	// Do a bunch of puts to keys in all shards
	n := 10
	ka := make([]string, n)
	va := make([]string, n)
	for i := 0; i < n; i++ {
		ka[i] = strconv.Itoa(i) // ensure multiple shards
		va[i] = "100"
		ck.Put(ka[i], va[i])
	}

	// QUERY to find shards owned by 102
	c := cfg.mck.Query(-1)
	owned := make(map[int]bool, n)
	for s, gid := range c.Shards {
		owned[s] = gid == cfg.groups[2].gid
	}

	// KILL 100
	cfg.ShutdownGroup(0)

	// LEAVE 100 + 102
	// 101 can get old shards from 102, but not from 100. 101 should start
	// serving shards that used to belong to 102 as soon as possible
	cfg.leavem([]int{0, 2})

	// Give the implementation some time to start reconfiguration
	// And to migrate 102 -> 101
	<-time.After(1 * time.Second)

	// And finally: check that gets/puts for 101-owned keys now complete
	for i := 0; i < n; i++ {
		shard := key2shard(ka[i])
		if owned[shard] {
			check(t, ck, ka[i], va[i], mylog)
			ck.Put(ka[i], va[i]+"-2")
			check(t, ck, ka[i], va[i]+"-2", mylog)
		}
	}

	fmt.Printf("  ... Passed\n")
	mylog.DFprintf("  ... Passed\n")

}
