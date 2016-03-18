package main

import(
    "os"
    "io"
    "log"
    "time"
    "os/exec"
    "path/filepath"
    "encoding/json"

    "github.com/robfig/cron"
)

var c *cron.Cron

type ExecutableJob struct {
    Name        string `json:"name"`
    CronExp     string `json:"cronexp"`
    Cmd         string `json:"cmd"`
    Args        string `json:"args"`
    StdoutPath  string `json:"stdoutPath"`
    StderrPath  string `json:"stderrPath"`

    Stdout      io.Writer `json:"-"`
    Stderr      io.Writer `json:"-"`
}

func (ej *ExecutableJob) Run() {

    cmd := exec.Command(ej.Cmd, ej.Args)

    cmd.Stdout = ej.Stdout
    cmd.Stderr = ej.Stderr

    err := cmd.Run()

    if err != nil {
        log.Fatal("error running command: ", err)
    }

}

/*
 * In the future, we may want to open configs in separate goroutines, but
 * keep it serialized for now (version 2 or 3 feature?)
 */
func openJobConfig(path string) (*ExecutableJob, error) {

    file, err := os.Open(path)
    defer file.Close()

    if err != nil {
        log.Fatal("error opening job config file: ", err)
    }

    // parse the config
    var ej *ExecutableJob
    decoder := json.NewDecoder(file)

    err = decoder.Decode(&ej)
    if err != nil {
        return nil, err
    }

    openFileFlags := os.O_RDWR|os.O_APPEND|os.O_CREATE
    openFilePerms := os.FileMode(0644)

    // change the job to write out to custom stderr/stdout
    if outFile, err := os.OpenFile(ej.StdoutPath, openFileFlags, openFilePerms); err != nil {
        log.Println("unable to open stdout for writing: ", err)
    } else {
        ej.Stdout = outFile
    }

    if errFile, err := os.OpenFile(ej.StderrPath, openFileFlags, openFilePerms); err != nil {
        log.Println("unable to open stderr for writing: ", err)
    } else {
        ej.Stderr = errFile
    }

    return ej, nil

}

func main() {

    c = cron.New()
    defer c.Stop()

    configDirPath := "/home/nickantonelli/gotestit.d"
    // open the file
    dir, err := os.Open(configDirPath)
    if err != nil {
        log.Fatal("unable to open config directory for parsing: ", err)
    }

    filenames, err := dir.Readdirnames(0)
    if err != nil {
        log.Fatal("unable to get all file names: ", err)
    }

    for _, filename := range filenames {
        // add the job to the queue
        jobConfig := filepath.Join(configDirPath, filename)
        if job, err := openJobConfig(jobConfig); err !=  nil {
            log.Printf("error decoding job information for %s: %s", jobConfig, err)
        } else {
            c.AddJob(job.CronExp, job)
        }
    }

    c.Start()

    // let it ride
    for {
        time.Sleep(5000 * time.Millisecond)
    }

}

