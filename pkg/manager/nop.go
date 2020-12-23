package manager

import (
    "fmt"
    "net"
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

func(nop *NopManager) Addr() string {
    return ""
}

func(nop *NopManager) Dial() (net.Conn, error) {
    return nil, ErrNop
}
