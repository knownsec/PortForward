/**
* Filename: log.go
* Description: the log information format
* Author: knownsec404
* Time: 2020.08.17
*/

package main

import (
    "fmt"
    "time"
)


const (
    LOG_LEVEL_NONE  uint32 = 0
    LOG_LEVEL_FATAL uint32 = 1
    LOG_LEVEL_ERROR uint32 = 2
    LOG_LEVEL_WARN  uint32 = 3
    LOG_LEVEL_INFO  uint32 = 4
    LOG_LEVEL_DEBUG uint32 = 5
)

var LOG_LEVEL uint32 = LOG_LEVEL_DEBUG

/**********************************************************************
* @Function: LogFatal(format string, a ...interface{})
* @Description: log infomations with fatal level
* @Parameter: format string, the format string template
* @Parameter: a ...interface{}, the value
* @Return: nil
**********************************************************************/
func LogFatal(format string, a ...interface{}) {
    if LOG_LEVEL < LOG_LEVEL_FATAL {
        return
    }
    msg := fmt.Sprintf(format, a...)
    msg  = fmt.Sprintf("[%s] [FATAL] %s", getCurrentTime(), msg)
    fmt.Println(msg)
}


/**********************************************************************
* @Function: LogError(format string, a ...interface{})
* @Description: log infomations with error level
* @Parameter: format string, the format string template
* @Parameter: a ...interface{}, the value
* @Return: nil
**********************************************************************/
func LogError(format string, a ...interface{}) {
    if LOG_LEVEL < LOG_LEVEL_WARN {
        return
    }
    msg := fmt.Sprintf(format, a...)
    msg  = fmt.Sprintf("[%s] [ERROR] %s", getCurrentTime(), msg)
    fmt.Println(msg)
}


/**********************************************************************
* @Function: LogWarn(format string, a ...interface{})
* @Description: log infomations with warn level
* @Parameter: format string, the format string template
* @Parameter: a ...interface{}, the value
* @Return: nil
**********************************************************************/
func LogWarn(format string, a ...interface{}) {
    if LOG_LEVEL < LOG_LEVEL_WARN {
        return
    }
    msg := fmt.Sprintf(format, a...)
    msg  = fmt.Sprintf("[%s] [WARN] %s", getCurrentTime(), msg)
    fmt.Println(msg)
}


/**********************************************************************
* @Function: LogInfo(format string, a ...interface{})
* @Description: log infomations with info level
* @Parameter: format string, the format string template
* @Parameter: a ...interface{}, the value
* @Return: nil
**********************************************************************/
func LogInfo(format string, a ...interface{}) {
    if LOG_LEVEL < LOG_LEVEL_INFO {
        return
    }
    msg := fmt.Sprintf(format, a...)
    msg  = fmt.Sprintf("[%s] [INFO] %s", getCurrentTime(), msg)
    fmt.Println(msg)
}


/**********************************************************************
* @Function: LogDebug(format string, a ...interface{})
* @Description: log infomations with debug level
* @Parameter: format string, the format string template
* @Parameter: a ...interface{}, the value
* @Return: nil
**********************************************************************/
func LogDebug(format string, a ...interface{}) {
    if LOG_LEVEL < LOG_LEVEL_DEBUG {
        return
    }
    msg := fmt.Sprintf(format, a...)
    msg  = fmt.Sprintf("[%s] [DEBUG] %s", getCurrentTime(), msg)
    fmt.Println(msg)
}


/**********************************************************************
* @Function: getCurrentTime() (string)
* @Description: get current time as log format string
* @Parameter: nil
* @Return: string, the current time string
**********************************************************************/
func getCurrentTime() (string) {
    return time.Now().Format("01-02|15:04:05")
}
