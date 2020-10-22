/**
* Filename: tcp.go
* Description: the PortForward tcp layer implement
* Author: knownsec404
* Time: 2020.09.23
*/

package main

import (
    "net"
    "time"
)


/**********************************************************************
* @Function: ListenTCP(address string, clientc chan Conn, quit chan bool)
* @Description: listen local tcp service, and accept client connection,
*   initialize connection and return by channel.
* @Parameter: address string, the local listen address
* @Parameter: clientc chan Conn, new client connection channel
* @Parameter: quit chan bool, the quit signal channel
* @Return: nil
**********************************************************************/
func ListenTCP(address string, clientc chan Conn, quit chan bool) {
    addr, err := net.ResolveTCPAddr("tcp", address)
    if err != nil {
        LogError("tcp listen error, %s", err)
        clientc <- nil
        return
    }
    serv, err := net.ListenTCP("tcp", addr)
    if err != nil {
        LogError("tcp listen error, %s", err)
        clientc <- nil
        return
    }
    // the "conn" has been ready, close "serv"
    defer serv.Close()

    for {
        // check quit
        select {
        case <-quit:
            return
        default:
        }

        // set "Accept" timeout, for check "quit" signal
        serv.SetDeadline(time.Now().Add(16 * time.Second))
        conn, err := serv.Accept()
        if err != nil {
            if err, ok := err.(net.Error); ok && err.Timeout() {
                continue
            }
            // others error
            LogError("tcp listen error, %s", err)
            clientc <- nil
            break
        }

        // new client is connected
        clientc <- conn
    } // end for
}


/**********************************************************************
* @Function: ConnTCP(address string) (Conn, error)
* @Description: dial to remote server, and return tcp connection
* @Parameter: address string, the remote server address that needs to be dialed
* @Return: (Conn, error), the tcp connection and error
**********************************************************************/
func ConnTCP(address string) (Conn, error) {
    conn, err := net.DialTimeout("tcp", address, 10 * time.Second)
    if err != nil {
        return nil, err
    }

    return conn, nil
}
