package packet

import (
    "encoding/json"
    "fmt"
    "testing"
)

func TestSLPDo(t *testing.T) {

    req := Request{
        Protocol: -1,
        Address: "localhost",
        Port: 25565,
        NextState: StateStatus,
    }

    rsp, err := Do(req)
    t.Log(string(rsp.Bytes()), err)

}

func TestSLPHandler(t *testing.T) {

    example := `{"version":{"name":"Spigot 1.16.4","protocol":754},
"players":{"max":20,"online":0,"sample":null},
"description":{"extra":[{"text":"A Minecraft Server"}],"text":""}}`

    var rsp Response
    err := json.Unmarshal([]byte(example), &rsp)
    if err != nil {
        panic(err)
    }

    handler := HandlerFunc(func(rw ResponseWriter, req Request) {
        fmt.Println(req)
    })

    ListenAndServe("localhost:25565", handler)

}
