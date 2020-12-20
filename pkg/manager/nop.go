package manager

import (
    "fmt"
    "net"
    "time"
)

var ErrNop = fmt.Errorf("Nop")

type NopManager struct {
}

func NewNopManager() *NopManager {
    return &NopManager{}
}

func(nop *NopManager) Start() error {
    return ErrNop
}

func(nop *NopManager) State() (int, error) {
    return StateObscure, ErrNop
}

func(nop *NopManager) Dial() (net.Conn, error) {
    return nil, ErrNop
}

func(nop *NopManager) DialTimeout(timeout time.Duration) (net.Conn, error) {
    return nil, ErrNop
}
