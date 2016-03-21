package main

import(
    "os"
    "io"
    "log"
    "fmt"
    "time"
    "flag"
    "strings"
    "os/exec"
    "path/filepath"
    "encoding/json"

    "github.com/robfig/cron"
)

var c *cron.Cron
var eventStream chan *CronEvent

const DefaultConfigDirPath = "/etc/gotestit.d"

type CronEvent struct {
    EventJob        string
    EventTime       time.Time
    EventStatus     string
    EventMessage    string
}

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

    event := &CronEvent{
        EventJob: ej.Name,
        EventTime: time.Now(),
        EventStatus: "success",
        EventMessage: "",
    }

    if err != nil {
        event.EventStatus = "failure"
        event.EventMessage = fmt.Sprintf("error executing %s: %s", ej.Name, err)
    } else {
        event.EventMessage = fmt.Sprintf("successfully ran %s", ej.Name)
    }

    eventStream <-event

}


var cmdFlagConfigDirPath string
func init() {
    flag.StringVar(&cmdFlagConfigDirPath, "d", DefaultConfigDirPath, "sets the directory in which to read job configurations")
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

    flag.Parse()

    c = cron.New()
    defer c.Stop()
    eventStream = make(chan *CronEvent)

    // open the file
    dir, err := os.Open(cmdFlagConfigDir)
    if err != nil {
        log.Fatal("unable to open config directory for parsing: ", err)
    }

    if stat, err := dir.Stat(); err != nil {
        log.Fatal("unable to stat " + cmdFlagConfigDir + ", error: ", err)
    } else if !stat.IsDir() {
        log.Fatal(cmdFlagConfigDir + " is not a valid directory")
    }

    filenames, err := dir.Readdirnames(0)
    if err != nil {
        log.Fatal("unable to get all file names: ", err)
    }

    jobsAdded := false

    for _, filename := range filenames {
        // add the job to the queue
        jobConfig := filepath.Join(cmdFlagConfigDir, filename)

        if hasSuffix := strings.HasSuffix(jobConfig, "json"); !hasSuffix {
            continue
        }

        if job, err := openJobConfig(jobConfig); err !=  nil {
            log.Printf("error decoding job information for %s: %s", jobConfig, err)
        } else {
            c.AddJob(job.CronExp, job)
            jobsAdded = true
        }
    }

    if !jobsAdded {
        log.Fatal("no jobs found, exiting")
    }

    c.Start()

    // let it ride
    for {
        select {
        case event, ok := <-eventStream:
            if !ok {
                // not sure how we got here, maybe we caught a signal
                break
            }
            if event.EventStatus == "success" {
                log.Printf("ran job `%s` with message: %s\n", event.EventJob, event.EventMessage)
            } else {
                log.Printf("job `%s` failed: %s\n", event.EventJob, event.EventMessage)
            }
        default:
        }
    }
}

