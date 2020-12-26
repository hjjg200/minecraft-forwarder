package packet

import (
    "fmt"
    "io"
    "net"
    "sync"

    "github.com/hjjg200/act"
)

// Handler
// Handlers expect only the Handshake part was read
type Handler interface {
    Serve(net.Conn, Handshake)
}
type HandlerFunc func(net.Conn, Handshake)
func(hf HandlerFunc) Serve(conn net.Conn, hs Handshake) {
    hf(conn, hs)
}

func Forward(src net.Conn, hs Handshake, dst net.Conn) {

    var wg sync.WaitGroup
    wg.Add(2)

    conncopy := func(to, from net.Conn) {
        defer from.Close()
        defer to.Close()
        io.Copy(to, from)
        wg.Done()
    }

    dst.Write(hs.Bytes())

    go conncopy(src, dst)
    go conncopy(dst, src)
    wg.Wait()

}

func ServeDisconnect(src net.Conn, hs Handshake, reason Chat) {

    if hs.NextState != StateLogin {
        src.Close()
        fmt.Println("Attempted to disconnect non-login packet")
        return
    }

    src.Write(Disconnect{reason}.Bytes())
    src.Close()

}

func ServeResponse(src net.Conn, hs Handshake, rsp Response) {

    if hs.NextState != StateStatus {
        src.Close()
        return
    }

    _, err := ReadRequest(src)
    act.Try(err)

    src.Write(rsp.Bytes())

    pp0, err := ReadPingPong(src)
    act.Try(err)

    src.Write(pp0.Bytes())
    src.Close()

}

func ListenAndServe(addr string, handler Handler) error {

    ln, err := net.Listen("tcp", addr)
    if err != nil {
        return err
    }

    for {

        conn, err := ln.Accept()
        if err != nil {
            fmt.Println("Connection exception:", err)
        }

        go func(src net.Conn) {

            hs, err := ReadHandshake(src)
            if err != nil {
                return
            }

            handler.Serve(src, hs)

        }(conn)

    }

    return fmt.Errorf("Server stopped")

}

