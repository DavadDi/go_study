package main

import (
    log "github.com/cihub/seelog"
    "fmt"
)

func main(){
    // init log
    defer log.Flush()

    logger, err := log.LoggerFromConfigAsFile("./seelog.xml")
    if err != nil {

        fmt.Println("Open seelog.xml failed ", err.Error())
        log.Critical("err parsing config log file", err)
        return
    }

    fmt.Println("Ready to call ReplaceLogger ")
    log.ReplaceLogger(logger)

    log.Debug("Hello debug")
    log.Error("Hello I am error")
    log.Info("Yes I am Info")
}
