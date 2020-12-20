package manager

import (
    "net"
    "time"
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
    return net.DialTimeout("tcp", addr, timeout)
}
