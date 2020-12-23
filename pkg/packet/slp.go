package packet

import (
    "encoding/json"
    "fmt"
    "io"
    "net"
    "strconv"
    "sync"
    "time"

    "github.com/hjjg200/act"
)

const (
    IDHandshake = 0x00
    IDStatusSLP = 0x00
    IDStatusPingPong = 0x01
    StateStatus = 1
    StateLogin = 2
)

// Handshake
type Handshake struct {
    Protocol int32
    Address string
    Port uint16
    NextState int32
}

func ReadHandshake(rd io.Reader) (hs Handshake, err error) {

    defer act.CatchAndStore(&err)

    pr := NewPacketReader(IDHandshake, rd)

    hs.Protocol = pr.NextVarInt()
    hs.Address = pr.NextString()
    hs.Port = uint16(pr.NextInt(2))
    hs.NextState = pr.NextVarInt()

    return hs, nil

}

func(hs Handshake) Bytes() []byte {

    pk := NewPacket(IDHandshake)

    pk.PutVarInt(hs.Protocol)
    pk.PutString(hs.Address)
    pk.PutInt(int64(hs.Port), 2)
    pk.PutVarInt(hs.NextState)

    return pk.Bytes()

}

// Status request
type Request struct {
}

func ReadRequest(rd io.Reader) (req Request, err error) {

    defer act.CatchAndStore(&err)

    NewPacketReader(IDStatusSLP, rd)

    return req, nil

}

func(req Request) Bytes() []byte {

    pk := NewPacket(IDStatusSLP)

    return pk.Bytes()

}

// Status response
type (
    VersionStruct struct {
        Name string `json:"name"`
        Protocol int `json:"protocol"`
    }
    SampleStruct struct {
        Name string `json:"name"`
        ID string `json:"id"`
    }
    PlayersStruct struct {
        Max int `json:"max"`
        Online int `json:"online"`
        Sample []SampleStruct `json:"sample"`
    }
    Response struct {
        Version VersionStruct `json:"version"`
        Players PlayersStruct `json:"players"`
        Description Chat `json:"description"`
    }
)

func ReadResponse(rd io.Reader) (rsp Response, err error) {

    defer act.CatchAndStore(&err)

    pr := NewPacketReader(IDHandshake, rd)

    js := pr.NextString()
    err = json.Unmarshal([]byte(js), &rsp)
    act.Try(err)

    return rsp, nil

}

func Status(addr string) (rsp Response, err error) {

    defer act.CatchAndStore(&err)

    host, portstr, err := net.SplitHostPort(addr)
    act.Try(err)

    port, err := strconv.Atoi(portstr)
    act.Try(err)

    // Handshake
    hs := Handshake{
        Protocol: -1,
        Address: host,
        Port: uint16(port),
        NextState: StateStatus,
    }
    conn, err := net.Dial("tcp", addr)
    act.Try(err)
    defer conn.Close()

    conn.Write(hs.Bytes())

    // Status request
    req := Request{}

    conn.Write(req.Bytes())

    // Status response
    rsp, err = ReadResponse(conn)
    act.Try(err)

    // Ping
    pp0 := PingPong{time.Now().Unix()}

    conn.Write(pp0.Bytes())

    // Pong
    pp1, err := ReadPingPong(conn)
    act.Try(err)
    act.Assert(pp0.Payload == pp1.Payload, fmt.Errorf("Ping pong failed"))

    return rsp, nil

}

func(rsp Response) Bytes() []byte {

    pk := NewPacket(IDHandshake)

    js, err := json.Marshal(rsp)
    if err != nil {
        panic(err)
    }
    pk.PutString(string(js))

    return pk.Bytes()

}

// Status ping pong
type PingPong struct {
    Payload int64
}

func ReadPingPong(rd io.Reader) (pp PingPong, err error) {

    defer act.CatchAndStore(&err)

    pr := NewPacketReader(IDStatusPingPong, rd)

    pp.Payload = pr.NextInt(8)

    return pp, nil

}

func(pp PingPong) Bytes() []byte {

    pk := NewPacket(IDStatusPingPong)

    pk.PutInt(pp.Payload, 8)

    return pk.Bytes()

}

// Handler
type Handler interface {
    Serve(net.Conn, Handshake)
}
type HandlerFunc func(net.Conn, Handshake)
func(hf HandlerFunc) Serve(conn net.Conn, hs Handshake) {
    hf(conn, hs)
}

// Handlers expect only the Handshake part was read
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

