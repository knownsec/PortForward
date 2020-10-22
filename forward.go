/**
* Filename: forward.go
* Description: the PortForward main logic implement, Three working modes are
*   provided:
*   1.Conn<=>Conn: dial A remote server, dial B remote server, connected
*   2.Listen<=>Conn: listen local server, dial remote server, connected
*   3.Listen<=>Listen: listen A local server, listen B local server, connected
* Author: knownsec404
* Time: 2020.09.23
*/

package main

import (
    "io"
    "net"
    "time"
)

//
const PORTFORWARD_PROTO_NIL   uint8 = 0x00
const PORTFORWARD_PROTO_TCP   uint8 = 0x10
const PORTFORWARD_PROTO_UDP   uint8 = 0x20
//
const PORTFORWARD_SOCK_NIL    uint8 = 0x00
const PORTFORWARD_SOCK_LISTEN uint8 = 0x01
const PORTFORWARD_SOCK_CONN   uint8 = 0x02

// the PortForward network interface
type Conn interface {
    // Read reads data from the connection.
    Read(b []byte) (n int, err error)
    // Write writes data to the connection.
    Write(b []byte) (n int, err error)
    // Close closes the connection.
    Close() (error)
    // RemoteAddr returns the remote network address.
    RemoteAddr() (net.Addr)
}

// the PortForward launch arguemnt
type Args struct {
    Protocol    uint8
    // sock1
    Method1     uint8
    Addr1       string
    // sock2
    Method2     uint8
    Addr2       string
}

var stop chan bool = nil

/**********************************************************************
* @Function: Launch(args Args)
* @Description: launch PortForward working mode by arguments
* @Parameter: args Args, the launch arguments
* @Return: nil
**********************************************************************/
func Launch(args Args)  {
    // initialize stop channel, the maximum number of coroutines that
    // need to be managed is 2 (listen-listen)
    stop = make(chan bool, 3)

    //
    if args.Method1 == PORTFORWARD_SOCK_CONN &&
        args.Method2 == PORTFORWARD_SOCK_CONN {
        // sock1 conn, sock2 conn
        ConnConn(args.Protocol, args.Addr1, args.Addr2)
    } else if args.Method1 == PORTFORWARD_SOCK_CONN &&
        args.Method2 == PORTFORWARD_SOCK_LISTEN {
        // sock1 conn, sock2 listen
        ListenConn(args.Protocol, args.Addr2, args.Addr1)
    } else if args.Method1 == PORTFORWARD_SOCK_LISTEN &&
        args.Method2 == PORTFORWARD_SOCK_CONN {
        // sock1 listen, sock2 conn
        ListenConn(args.Protocol, args.Addr1, args.Addr2)
    } else if args.Method1 == PORTFORWARD_SOCK_LISTEN &&
        args.Method2 == PORTFORWARD_SOCK_LISTEN {
        // sock1 listen , sock2 listen
        ListenListen(args.Protocol, args.Addr1, args.Addr2)
    } else {
        LogError("unknown forward method")
        return
    }
}


/**********************************************************************
* @Function: Shutdown()
* @Description: the shutdown PortForward
* @Parameter: nil
* @Return: nil
**********************************************************************/
func Shutdown() {
    stop <- true
}


/**********************************************************************
* @Function: ListenConn(proto uint8, addr1 string, addr2 string)
* @Description: "Listen<=>Conn" working mode
* @Parameter: proto uint8, the tcp or udp protocol setting
* @Parameter: addr1 string, the address1 "ip:port" string
* @Parameter: addr2 string, the address2 "ip:port" string
* @Return: nil
**********************************************************************/
func ListenConn(proto uint8, addr1 string, addr2 string) {
    // get sock launch function by protocol
    sockfoo1 := ListenTCP
    if proto == PORTFORWARD_PROTO_UDP {
        sockfoo1 = ListenUDP
    }
    sockfoo2 := ConnTCP
    if proto == PORTFORWARD_PROTO_UDP {
        sockfoo2 = ConnUDP
    }

    // launch socket1 listen
    clientc := make(chan Conn)
    quit := make(chan bool, 1)
    LogInfo("listen A point with sock1 [%s]", addr1)
    go sockfoo1(addr1, clientc, quit)

    var count int = 1
    for {
        // socket1 listen & quit signal
        var sock1 Conn = nil
        select {
        case <-stop:
            quit <- true
            return
        case sock1 = <-clientc:
            if sock1 == nil {
                // set stop flag when error happend
                stop <- true
                continue
            }
        }
        LogInfo("A point(link%d) [%s] is ready", count, sock1.RemoteAddr())
        // socket2 dial
        LogInfo("dial B point with sock2 [%s]", addr2)
        sock2, err := sockfoo2(addr2)
        if err != nil {
            sock1.Close()
            LogError("%s", err)
            continue
        }
        LogInfo("B point(sock2) is ready")

        // connect with sockets
        go ConnectSock(count, sock1, sock2)
        count += 1
    } // end for
}


/**********************************************************************
* @Function: ListenListen(proto uint8, addr1 string, addr2 string)
* @Description: the "Listen<=>Listen" working mode
* @Parameter: proto uint8, the tcp or udp protocol setting
* @Parameter: addr1 string, the address1 "ip:port" string
* @Parameter: addr2 string, the address2 "ip:port" string
* @Return: nil
**********************************************************************/
func ListenListen(proto uint8, addr1 string, addr2 string) {
    release := func(s1 Conn, s2 Conn) {
        if s1 != nil {
            s1.Close()
        }
        if s2 != nil {
            s2.Close()
        }
    }

    // get sock launch function by protocol
    sockfoo := ListenTCP
    if proto == PORTFORWARD_PROTO_UDP {
        sockfoo = ListenUDP
    }

    // launch socket1 listen
    clientc1 := make(chan Conn)
    quit1 := make(chan bool, 1)
    LogInfo("listen A point with sock1 [%s]", addr1)
    go sockfoo(addr1, clientc1, quit1)
    // launch socket2 listen
    clientc2 := make(chan Conn)
    quit2 := make(chan bool, 1)
    LogInfo("listen B point with sock2 [%s]", addr2)
    go sockfoo(addr2, clientc2, quit2)

    var sock1 Conn = nil
    var sock2 Conn = nil
    var count int = 1
    for {
        select {
        case <-stop:
            quit1 <- true
            quit2 <- true
            release(sock1, sock2)
            return
        case sock1 = <-clientc1:
            if sock1 == nil {
                // set stop flag when error happend
                stop <- true
                continue
            }
            LogInfo("A point(link%d) [%s] is ready", count, sock1.RemoteAddr())
        case sock2 = <-clientc2:
            if sock2 == nil {
                // set stop flag when error happend
                stop <- true
                continue
            }
            LogInfo("B point(link%d) [%s] is ready", count, sock2.RemoteAddr())
        case <-time.After(120 * time.Second):
            LogWarn("socket wait timeout, reset")
            release(sock1, sock2)
            continue
        }

        // wait another socket ready
        if sock1 == nil || sock2 == nil {
            continue
        }

        // the two socket is ready, connect with sockets
        go ConnectSock(count, sock1, sock2)
        count += 1
        // reset sock1 & sock2
        sock1 = nil
        sock2 = nil
    } // end for
}


/**********************************************************************
* @Function: ConnConn(proto uint8, addr1 string, addr2 string)
* @Description: the "Conn<=>Conn" working mode
* @Parameter: proto uint8, the tcp or udp protocol setting
* @Parameter: addr1 string, the address1 "ip:port" string
* @Parameter: addr2 string, the address2 "ip:port" string
* @Return: nil
**********************************************************************/
func ConnConn(proto uint8, addr1 string, addr2 string) {
    // get sock launch function by protocol
    sockfoo := ConnTCP
    if proto == PORTFORWARD_PROTO_UDP {
        sockfoo = ConnUDP
    }

    var count int = 1
    for {
        select {
        case <-stop:
            return
        default:
        }

        // socket1 dial
        LogInfo("dial A point with sock1 [%s]", addr1)
        sock1, err := sockfoo(addr1)
        if err != nil {
            LogError("%s", err)
            time.Sleep(16 * time.Second)
            continue
        }
        LogInfo("A point(sock1) is ready")

        // waiting for the first message sent by the A point(sock1)
        buf := make([]byte, 32 * 1024)
        n, err := sock1.Read(buf)
        if err != nil {
            LogError("A point: %s", err)
            continue
        }
        buf = buf[:n]

        // socket2 dial
        LogInfo("dial B point with sock2 [%s]", addr2)
        sock2, err := sockfoo(addr2)
        if err != nil {
            sock1.Close()
            LogError("%s", err)
            time.Sleep(16 * time.Second)
            continue
        }
        LogInfo("B point(sock2) is ready")

        // first pass in the first message above
        _, err = sock2.Write(buf)
        if err != nil {
            LogError("B point: %s", err)
            continue
        }

        // connect with sockets
        go ConnectSock(count, sock1, sock2)
        count += 1
    } // end for
}


/**********************************************************************
* @Function: ConnectSock(id int, sock1 Conn, sock2 Conn)
* @Description: connect two sockets, if an error occurs, the socket will
*   be closed so that the coroutine can exit normally
* @Parameter: id int, the communication link id
* @Parameter: sock1 Conn, the first socket object
* @Parameter: sock2 Conn, the second socket object
* @Return: nil
**********************************************************************/
func ConnectSock(id int, sock1 Conn, sock2 Conn) {
    exit := make(chan bool, 1)

    //
    go func() {
        _, err := io.Copy(sock1, sock2)
        if err != nil {
            LogError("ConnectSock%d(A=>B): %s", id, err)
        } else {
            LogInfo("ConnectSock%d(A=>B) exited", id)
        }
        exit <- true
    }()

    //
    go func() {
        _, err := io.Copy(sock2, sock1)
        if err != nil {
            LogError("ConnectSock%d(B=>A): %s", id, err)
        } else {
            LogInfo("ConnectSock%d(B=>A) exited", id)
        }
        exit <- true
    }()

    // exit when close either end
    <-exit
    // close all socket, so that "io.Copy" can exit
    sock1.Close()
    sock2.Close()
}
