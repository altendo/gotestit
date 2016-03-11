package main

import(
    "os"
    "log"
    "bytes"
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
}

func (ej *ExecutableJob) Run() {
    cmd := exec.Command(ej.Script, ej.Args)
    out, err := cmd.CombinedOutput()

    if err != nil {
        log.Fatal(err)
    }

    buffer := bytes.NewBuffer(out)
    str, err := buffer.ReadString(byte('\n'))
    if err != nil {
        log.Fatal(err)
    }

    log.Print(str)
}

func main() {

    c = cron.New()

    configPath := "test.json"

    file, err := os.Open(configPath)

    if err != nil {
        log.Fatal("error opening config file,", err)
    }

    var ej *ExecutableJob
    decoder := json.NewDecoder(file)

    err = decoder.Decode(&ej)
    if err != nil {
        log.Fatal("error decoding job information,", err)
    }

    c.AddJob(ej.CronExp, ej)
    c.Start()

    defer c.Stop()

    for {
        time.Sleep(5000 * time.Millisecond)
    }

}

