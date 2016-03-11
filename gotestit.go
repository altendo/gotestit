package main

import(
    "os"
    "io"
    "log"
    "time"
    "os/exec"
    "encoding/json"

    "github.com/robfig/cron"
)

var c *cron.Cron

type ExecutableJob struct {
    Name        string `json:"name"`
    CronExp     string `json:"cronexp"`
    Script      string `json:"script"`
    Args        string `json:"args"`
    StdoutPath  string `json:"stdoutPath"`
    StderrPath  string `json:"stderrPath"`

    Stdout      io.WriteCloser `json:"-"`
    Stderr      io.WriteCloser `json:"-"`
}

func (ej *ExecutableJob) Run() {

    cmd := exec.Command(ej.Script, ej.Args)

    cmd.Stdout = ej.Stdout
    cmd.Stderr = ej.Stderr

    err := cmd.Run()

    if err != nil {
        log.Fatal("error running command: ", err)
    }

}

func main() {

    c = cron.New()

    // open the file
    configPath := "test.json"
    file, err := os.Open(configPath)

    if err != nil {
        log.Fatal("error opening config file,", err)
    }

    // parse the config
    var ej *ExecutableJob
    decoder := json.NewDecoder(file)

    err = decoder.Decode(&ej)
    if err != nil {
        log.Fatal("error decoding job information,", err)
    }

    // change the job to write out to custom stderr/stdout
    if outFile, err := os.OpenFile(ej.StdoutPath, os.O_APPEND|os.O_CREATE, 0644); err != nil {
        log.Fatal("unable to open stdout for writing: ", err)
    } else {
        ej.Stdout = outFile
    }

    if errFile, err := os.OpenFile(ej.StderrPath, os.O_APPEND|os.O_CREATE, 0644); err != nil {
        log.Fatal("unable to open stderr for writing: ", err)
    } else {
        ej.Stderr = errFile
    }

    // add the job to the queue
    c.AddJob(ej.CronExp, ej)
    c.Start()

    defer c.Stop()

    // let it ride
    for {
        time.Sleep(5000 * time.Millisecond)
    }

}

