package main

// Pool represents a goroutine pool
type Pool struct {
    work chan func()
    sem  chan struct{}
}

// New creates a new goroutine pool
func New(size int) *Pool {
    return &Pool{
        work: make(chan func()),
        sem:  make(chan struct{}, size),
    }
}

// Schedule schedules work to the pool
func (p *Pool) Schedule(task func()) {
    select {
    case p.work <- task:
    case p.sem <- struct{}{}:
        go p.worker(task)
    }
}

func (p *Pool) worker(task func()) {
    defer func() { <-p.sem }()
    for {
        task()
        task = <-p.work
    }
}

import (
	"bufio"
	"database/sql"
	"fmt"
	"sync"
	"github.com/akamensky/argparse"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	//"sync"
	"syscall"
	"time"
)

type MySql struct {
	Db	*sql.DB
}


func (sqObj *MySql)Crt_tb() {
	// create table if not exists
	sql_job_table := `
	CREATE TABLE IF NOT EXISTS job(
		Id INTEGER NOT NULL PRIMARY KEY,
		subJob_num INTEGER UNIQUE NOT NULL,
		shellPath	TEXT,
		status	TEXT,
		exitCode	integer,
		retry	integer, 
		starttime	datetime,
		endtime	datetime 
	);
	`
	_, err := sqObj.Db.Exec(sql_job_table)
	if err != nil {
		panic(err)
	}
}


type jobStatusType string

// These are project or module type.
const (
	J_pending    jobStatusType = "Pending"
	J_failed    jobStatusType = "Failed"
	J_running  jobStatusType = "Running"
	J_finished  jobStatusType = "Finished"
)


func CheckCount(rows *sql.Rows) (count int) {
	count = 0
	for rows.Next() {
		count ++
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	return count
}

func GenerateShell(shellPath, content  string) {
	fi, err := os.Create(shellPath)
	if err != nil {
		panic(err)
	}
	defer fi.Close()

	content = strings.TrimRight(content, "\n")
	content = fmt.Sprintf("#!/bin/bash\necho ========== start at : `date +%%Y/%%m/%%d` ==========\n%s",content)
	content = fmt.Sprintf("%s && \\\necho ========== end at : `date +%%Y/%%m/%%d` ========== && \\\n",content)
	content = fmt.Sprintf("%secho LLAP 1>&2 && \\\necho LLAP > %s.sign\n", content, shellPath)

	_, err = fi.Write([]byte(content))
}

func Creat_tb(shell_path string, line_unit int)(dbObj *MySql) {
	shellAbsName, _ := filepath.Abs(shell_path)
	dbpath := shellAbsName + ".db"
	subShellPath := shellAbsName + ".shell"

	err := os.MkdirAll(subShellPath, 0777)
	CheckErr(err)

	conn, err := sql.Open("sqlite3", dbpath)
	CheckErr(err)
	dbObj = &MySql{Db: conn}
	dbObj.Crt_tb()

	tx, _ := dbObj.Db.Begin()
	defer tx.Rollback()
	insert_job, err := tx.Prepare("INSERT INTO job(subJob_num, shellPath, status, retry) values(?,?,?,?)")
	CheckErr(err)

	f, err := os.Open(shellAbsName)
	if err != nil {
		panic(err)
	}
	buf := bufio.NewReader(f)

	ii := 0
	var cmd_l string = ""
	N := 0
	for {
		line, err := buf.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}

		if ii == 0{
			cmd_l = line
			ii++
		}else if ii < line_unit{
			cmd_l = cmd_l + line
			ii++
		}else{
			N++
			Nrows, err := tx.Query("select Id from job where subJob_num = ?", N)
			defer Nrows.Close()
			CheckErr(err)
			if CheckCount(Nrows)==0 {
				cmd_l = strings.TrimRight(cmd_l, "\n")
				subShell := subShellPath + "/work_" + strings.Repeat("0", 6-len(strconv.Itoa(N))) + strconv.Itoa(N) + ".sh"
				GenerateShell(subShell, cmd_l)
				_, _ = insert_job.Exec(N, subShell, J_pending, 0)
			}

			ii = 1
			cmd_l = line
		}
	}

	if ii > 0{
		N++
		Nrows, err := tx.Query("select Id from job where subJob_num = ?", N)
		defer Nrows.Close()
		CheckErr(err)
		if CheckCount(Nrows)==0 {
			cmd_l = strings.TrimRight(cmd_l, "\n")
			subShell := subShellPath + "/work_" + strings.Repeat("0", 6-len(strconv.Itoa(N))) + strconv.Itoa(N) + ".sh"
			GenerateShell(subShell, cmd_l)
			_, _ = insert_job.Exec(N, subShell, J_pending, 0)
		}
	}

	err = tx.Commit()
	CheckErr(err)
	return
}

func GetNeed2Run(dbObj *MySql)([]int){
	//need2run := make(map[int]int)
	tx, _ := dbObj.Db.Begin()
	defer tx.Rollback()

	rows, err := tx.Query("select subJob_num from job where Status!=?", J_finished)
	CheckErr(err)
	defer rows.Close()
	var subJob_num int

	need2run_N := CheckCount(rows)
	need2run := make([]int, need2run_N)

	ii := 0
	rows2, err := tx.Query("select subJob_num from job where Status!=?", J_finished)
	CheckErr(err)
	defer rows2.Close()
	for rows2.Next() {
		err = rows2.Scan(&subJob_num)
		CheckErr(err)
		need2run[ii] = subJob_num
		ii++
	}
	return need2run
}

func IlterCommand(dbObj *MySql, thred int, need2run []int){
	pool := New(thred)
	var writeWg sync.WaitGroup

	for _, N := range need2run{
		N := N // create local copy for closure
		pool.Schedule(func() {
			RunCommand(N, dbObj, &writeWg)
		})
	}
}


func RunCommand(N int, dbObj *MySql, writeWg *sync.WaitGroup){
	tx, _ := dbObj.Db.Begin()
	defer tx.Rollback()

	var subShellPath string
	err := dbObj.Db.QueryRow("select shellPath from job where subJob_num = ?", N).Scan(&subShellPath)
	CheckErr(err)

	now := time.Now().Format("2006-01-02 15:04:05")
	writeWg.Add(1)
	_, err = dbObj.Db.Exec("UPDATE job set status=?, starttime=? where subJob_num=?", J_running, now, N)
	CheckErr(err)
	writeWg.Done()

	defaultFailedCode := 1
	cmd := exec.Command("sh", subShellPath)
	// 其他程序stdout stderr改到当前目录pwd
	sho, err := os.OpenFile(fmt.Sprintf("%s.o", subShellPath), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	CheckErr(err)
	defer sho.Close()
	she, err := os.OpenFile(fmt.Sprintf("%s.e", subShellPath), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	CheckErr(err)
	defer she.Close()
	Owriter := io.MultiWriter(sho)
	Ewriter := io.MultiWriter(she)
	cmd.Stdout = Owriter
	cmd.Stderr = Ewriter
	err = cmd.Run() //blocks until sub process is complete
	//CheckErr(err)

	var exitCode int

	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			exitCode = defaultFailedCode
		}
	} else {
		// success, exitCode should be 0 if go is ok
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}

	//var lock sync.Mutex //互斥锁
	//lock.Lock()
	writeWg.Add(1)
	now = time.Now().Format("2006-01-02 15:04:05")
	if exitCode == 0{
		_, err = dbObj.Db.Exec("UPDATE job set status=?, endtime=?, exitCode=? where subJob_num=?", J_finished, now, exitCode, N)
	}else{
		_, err = dbObj.Db.Exec("UPDATE job set status=?, endtime=?, exitCode=? where subJob_num=?", J_failed, now, exitCode, N)
	}

	writeWg.Done()
	CheckErr(err)
}

func CheckExitCode(dbObj *MySql){
	tx, _ := dbObj.Db.Begin()
	defer tx.Rollback()

	rows1, err := tx.Query("select subJob_num, shellPath from job where exitCode!=0")
	CheckErr(err)
	defer rows1.Close()
	rows12, err := tx.Query("select subJob_num, shellPath from job where exitCode!=0")
	CheckErr(err)
	defer rows12.Close()

	rows0, err := tx.Query("select exitCode from job where exitCode==0")
	CheckErr(err)
	defer rows0.Close()

	SuccessCount := CheckCount(rows0)
	ErrorCount := CheckCount(rows1)

	exitCode := 0
	os.Stderr.WriteString(fmt.Sprintf("All works: %v\nSuccessed: %v\nError: %v\n", SuccessCount+ErrorCount, SuccessCount, ErrorCount))
	if ErrorCount >0 {
		exitCode = 1
		os.Stderr.WriteString("Err Shells:\n")
	}

	var subJob_num int
	var shellPath string
	for rows12.Next() {
		ErrorCount++
		err := rows12.Scan(&subJob_num, &shellPath)
		CheckErr(err)
		os.Stderr.WriteString(fmt.Sprintf("%v\t%s\n", subJob_num, shellPath))
	}

	os.Exit(exitCode)
}

var documents string = `辅助并发程序
                    Created by Yuan Zan(seqyuan@gmail.com)
                    Version 0.0.4 (20220223)
                    输入格式同qsub_sge的输入文件格式
1) 生成infile.shell目录`

func CheckErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}


func main() {
	parser := argparse.NewParser("multiProcess", documents)
	opt_i := parser.String("i", "infile", &argparse.Options{Required: true, Help: "Work.sh, same as qsub_sge's input format"})
	opt_l := parser.Int("l", "line", &argparse.Options{Default: 1, Help: "Number of lines as a unit"})
	opt_p := parser.Int("p", "thred", &argparse.Options{Default: 1, Help: "Thread process at same time"})
	//opt_r := parser.Int("r", "retry", &argparse.Options{Default: 1, Help: "Max retry times"})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		return
	}

	dbObj := Creat_tb(*opt_i, *opt_l)
	need2run := GetNeed2Run(dbObj)
	fmt.Println(need2run)

	IlterCommand(dbObj, *opt_p, need2run)
	CheckExitCode(dbObj)
}
