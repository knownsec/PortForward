/**
* Filename: udp.go
* Description: the PortForward udp layer implement.
* Author: knownsec404
* Time: 2020.09.23
*/

package main

import (
    "errors"
    "net"
    "time"
)

// as UDP client Conn
type UDPDistribute struct {
    Established bool
    Conn        *(net.UDPConn)
    RAddr       net.Addr
    Cache       chan []byte
}


/**********************************************************************
* @Function: NewUDPDistribute(conn *(net.UDPConn), addr net.Addr) (*UDPDistribute)
* @Description: initialize UDPDistribute structure (as UDP client Conn)
* @Parameter: conn *(net.UDPConn), the udp connection object
* @Parameter: addr net.Addr, the udp client remote adddress
* @Return: *UDPDistribute, the new UDPDistribute structure pointer
**********************************************************************/
func NewUDPDistribute(conn *(net.UDPConn), addr net.Addr) (*UDPDistribute) {
    return &UDPDistribute{
        Established: true,
        Conn:        conn,
        RAddr:       addr,
        Cache:       make(chan []byte, 16),
    }
}


/**********************************************************************
* @Function: (this *UDPDistribute) Close() (error)
* @Description: set "Established" flag is false, the UDP service will
*   cleaned up according to certain conditions.
* @Parameter: nil
* @Return: error, the error
**********************************************************************/
func (this *UDPDistribute) Close() (error) {
    this.Established = false
    return nil
}


/**********************************************************************
* @Function: (this *UDPDistribute) Read(b []byte) (n int, err error)
* @Description: read data from connection, due to the udp implementation of
*   PortForward, read here will only produce a timeout error and closed error
*   (compared to the normal net.Conn object)
* @Parameter: b []byte, the buffer for receive data
* @Return: (n int, err error), the length of the data read and error
**********************************************************************/
func (this *UDPDistribute) Read(b []byte) (n int, err error) {
    if !this.Established {
        return 0, errors.New("udp distrubute has closed")
    }

    select {
    case <-time.After(16 * time.Second):
        return 0, errors.New("udp distrubute read timeout")
    case data := <-this.Cache:
        n := len(data)
        copy(b, data)
        return n, nil
    }
}


/**********************************************************************
* @Function: (this *UDPDistribute) Write(b []byte) (n int, err error)
* @Description: write data to connection by "WriteTo()"
* @Parameter: b []byte, the data to be sent
* @Return: (n int, err error), the length of the data write and error
**********************************************************************/
func (this *UDPDistribute) Write(b []byte) (n int, err error) {
    if !this.Established {
        return 0, errors.New("udp distrubute has closed")
    }
    return this.Conn.WriteTo(b, this.RAddr)
}


/**********************************************************************
* @Function: (this *UDPDistribute) RemoteAddr() (net.Addr)
* @Description: get remote address
* @Parameter: nil
* @Return: net.Addr, the remote address
**********************************************************************/
func (this *UDPDistribute) RemoteAddr() (net.Addr) {
    return this.RAddr
}


/**********************************************************************
* @Function: ListenUDP(address string, clientc chan Conn, quit chan bool)
* @Description: listen local udp service, and accept client connection,
*   initialize connection and return by channel.
*   since udp is running as a service, it only obtains remote data through
*   the Read* function cluster (different from tcp), so we need a temporary
*   table to record, so that we can use the temporary table to determine
*   whether to forward or create a new link
* @Parameter: address string, the local listen address
* @Parameter: clientc chan Conn, new client connection channel
* @Parameter: quit chan bool, the quit signal channel
* @Return: nil
**********************************************************************/
func ListenUDP(address string, clientc chan Conn, quit chan bool) {
    addr, err := net.ResolveUDPAddr("udp", address)
    if err != nil {
        LogError("udp listen error, %s", err)
        clientc <- nil
        return
    }
    serv, err := net.ListenUDP("udp", addr)
    if err != nil {
        LogError("udp listen error, %s", err)
        clientc <- nil
        return
    }
    defer serv.Close()

    // the udp distrubute table
    table := make(map[string]*UDPDistribute)
    // NOTICE:
    // in the process of running, the table will generate invalid historical
    // data, we have not cleaned it up(These invalid historical data will not
    // affect our normal operation).
    //
    // if you want to deal with invalid historical data, the best way is to
    // launch a new coroutine to handle net.Read, and another coroutine to
    // handle the connection status, but additional logic to exit the
    // coroutine is needed. Currently we think it is unnecessary.
    //
    // under the current code logic: if a udp connection exits, it will timeout
    // to trigger "Close()", and finally set "Established" to false, but there
    // is no logic to check this state to clean up, so invalid historical data
    // is generated (of course, we can traverse to clean up? every packet? no)
    //
    // PS: you can see that we called "delete()" in the following logic, but
    // this is only for processing. when a new udp client hits invalid
    // historical data, we need to return a new connection so that PortForward
    // can create a communication link.

    for {
        // check quit
        select {
        case <-quit:
            return
        default:
        }

        // set timeout, for check "quit" signal
        serv.SetDeadline(time.Now().Add(16 * time.Second))

        // just new 32*1024, in the outer layer we used "io.Copy()", which
        // can only handle the size of 32*1024
        buf := make([]byte, 32 * 1024)
        n, addr, err := serv.ReadFrom(buf)
        if err != nil {
            if err, ok := err.(net.Error); ok && err.Timeout() {
                continue
            }
            LogError("udp listen error, %s", err)
            clientc <- nil
            return
        }
        buf = buf[:n]

        // if the address in table, we distrubute message
        if d, ok := table[addr.String()]; ok {
            if d.Established {
                // it is established, distrubute message
                d.Cache <- buf
                continue
            } else {
                // we remove it when the connnection has expired
                delete(table, addr.String())
            }
        }
        // if the address not in table, we create new connection object
        conn := NewUDPDistribute(serv, addr)
        table[addr.String()] = conn
        conn.Cache <- buf
        clientc <- conn
    } // end for
}


/**********************************************************************
* @Function: ConnUDP(address string) (Conn, error)
* @Description: dial to remote server, and return udp connection
* @Parameter: address string, the remote server address that needs to be dialed
* @Return: (Conn, error), the udp connection and error
**********************************************************************/
func ConnUDP(address string) (Conn, error) {
    conn, err := net.DialTimeout("udp", address, 10 * time.Second)
    if err != nil {
        return nil, err
    }

    // send one byte(knock) to server, get "established" udp connection
    _, err = conn.Write([]byte("\x00"))
    if err != nil {
        return nil, err
    }

    // due to the characteristics of udp, when the udp server exits, we will
    // not receive any signal, it will be blocked at conn.Read();
    // here we set a timeout for udp
    conn.SetDeadline(time.Now().Add(60 * time.Second))
    return conn, nil
}
