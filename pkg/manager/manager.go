package manager

import (
    "fmt"
    "net"
    "time"

    "github.com/hjjg200/minecraft-forwarder/pkg/packet"
)

const (
    StateObscure = iota - 1
    StateStopped
    StatePending // server machine is booting up + minecraft server is not responding, prev state is stopped
    StateRunning
    StateStopping // minecraft server not responding, prev state is running
)

type Manager interface {
    Start() error
    State() (int, error)
    Dial() (net.Conn, error)
}

func dialTimeout(addr string, timeout time.Duration) (net.Conn, error) {

    c := make(chan error, 1)

    go func() {
        time.Sleep(timeout)
        c <- fmt.Errorf("Dial timeout")
    }()

    var dst net.Conn
    go func() {
        var err error
        dst, err = net.Dial("tcp", addr)
        if err != nil {
            c <-err
        }

        _, err = packet.Status(addr)
        c <- err
    }()

    return dst, <-c

}
