package main

import (
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "net"
    "os"
    "sync"

    "github.com/hjjg200/minecraft-forwarder/pkg/packet"
    "github.com/hjjg200/minecraft-forwarder/pkg/manager"

    "github.com/hjjg200/act"
    "github.com/hjjg200/go-jsoncfg"
)

type (

    ServerConfig struct {
        Name string `json:"name"`
        Aliases []string `json:"aliases"`
        Port uint16 `json:"port"`
        Forward interface{} `json:"forward"`
    }

    MessageConfig struct {
        Stopped string `json:"stopped"`
        Pending string `json:"pending"`
        Stopping string `json:"stopping"`
        Obscure string `json:"obscure"`
        Started string `json:"started"`
        StartFailed string `json:"startFailed"`
    }

    Config struct {
        Listen []string `json:"listen"`
        Servers []ServerConfig `json:"servers"`
        Messages MessageConfig `json:"messages"`
    }

)

var DefaultServerConfig = ServerConfig{
    Aliases: []string{},
    Port: 25565,
    Forward: map[string] interface{}{
        "type": "nop",
    },
}

var DefaultConfig = Config{

    Listen: []string{
        ":25565",
    },

    Messages: MessageConfig{
        Stopped: "STOPPED\nAttempt login to start it up",
        Pending: "PENDING...",
        Stopping: "STOPPING...",
        Obscure: "STATE OBSCURE",
        Started: "Successfully started the server!",
        StartFailed: "Failed to start the server!",
    },

    Servers: []ServerConfig{
        {
            Name: "example.com",
            Aliases: []string{"mc.example.com"},
            Port: 25565,
            Forward: map[string] interface{}{
                "type": "nop",
            },
        },
    },

}

var appConfig Config
var managers = make(map[string] manager.Manager)

func main() {

    // Config
    cfgparser, err := jsoncfg.NewParser(&DefaultConfig)
    act.Try(err)
    act.Try(cfgparser.SetSubDefault(&DefaultServerConfig))

    cfgpath := "./config.json"
    cfgfile, err := os.OpenFile(cfgpath, os.O_RDONLY, 0600)
    if os.IsNotExist(err) {
        cfgfile, err = os.OpenFile(cfgpath, os.O_WRONLY | os.O_CREATE, 0600)
        act.Try(err)

        enc := json.NewEncoder(cfgfile)
        enc.SetIndent("", "  ")
        act.Try(enc.Encode(DefaultConfig))

        appConfig = DefaultConfig
    } else if err != nil {
        panic(err)
    } else {
        data, err := ioutil.ReadAll(cfgfile)
        act.Try(err)
        act.Try(cfgparser.Parse(data, &appConfig))
    }

    // Create managers
    for _, server := range appConfig.Servers {
        data, err := json.Marshal(server.Forward)
        act.Try(err)

        var m manager.Manager
        switch server.Forward.(map[string] interface{})["type"].(string) {
        case "nop":
            m = manager.NewNopManager()
        case "ec2":
            ec2, err := manager.NewEC2ManagerJson(data)
            act.Try(err)
            m = ec2
        default:
            panic("Unknown server forward type")
        }

        managers[server.uuid()] = m
    }

    // Loop
    var wg sync.WaitGroup
    wg.Add(len(appConfig.Servers))

    for _, each := range appConfig.Listen {
        go func(addr string) {

            handler := packet.HandlerFunc(func(src net.Conn, hs packet.Handshake) {

                // Catch panic
                defer act.Catch(func(err error) {
                    if err == io.EOF {
                        return
                    }
                    fmt.Println(err, act.Stack())
                })

                // Find matching server config
                var server *ServerConfig
Loop:
                for _, s := range appConfig.Servers {
                    if hs.Address == s.Name {
                        server = &s
                        break Loop
                    }
                    for _, alias := range s.Aliases {
                        if hs.Address == alias {
                            server = &s
                            break Loop
                        }
                    }
                }

                if server == nil {
                    src.Close()
                    fmt.Println("No server was found for", hs.Address)
                    return
                }

                // Check state
                m, ok := managers[server.uuid()]
                act.Assert(ok, fmt.Errorf("Manager for server is not found"))

                state, err := m.State()
                act.Try(err)

                // Handle each state
                respond := func(msg, color string) {
                    packet.ServeResponse(src, hs, packet.Response{
                        Version: packet.VersionStruct{
                            Name: "",
                            Protocol: -1,
                        },
                        Description: packet.Chat{
                            ChatElem: packet.ChatElem{
                                Text: msg,
                                Color: color,
                            },
                        },
                    })
                }
                switch state {
                case manager.StateStopped:
                    if hs.NextState == packet.StateLogin {
                        var c packet.Chat
                        if m.Start() != nil {
                            c.Text = appConfig.Messages.StartFailed
                            c.Color = "red"
                        } else {
                            c.Text = appConfig.Messages.Started
                            c.Color = "green"
                        }
                        packet.ServeDisconnect(src, hs, c)
                        return
                    }
                    respond(appConfig.Messages.Stopped, "red")
                    return
                case manager.StatePending:
                    respond(appConfig.Messages.Pending, "gold")
                    return
                case manager.StateRunning:
                    dst, err := m.Dial()
                    act.Try(err)
                    packet.Forward(src, hs, dst)
                    return
                case manager.StateStopping:
                    respond(appConfig.Messages.Stopping, "red")
                    return
                }

                respond(appConfig.Messages.Obscure, "gray")

            })

            packet.ListenAndServe(addr, handler)
            wg.Done()

        }(each)
    }

    wg.Wait()

}

func(scfg ServerConfig) uuid() string {
    return net.JoinHostPort(scfg.Name, fmt.Sprintf("%d", scfg.Port))
}
