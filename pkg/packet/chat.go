package packet

import (
    "encoding/json"
)

type ChatElem struct {
    Text string `json:"text"`
    Italic bool `json:"italic"`
    Underlined bool `json:"underlined"`
    Strikethrough bool `json:"strikethrough"`
    Obfuscated bool `json:"obfuscated"`
    Bold bool `json:"bold"`
    Color string `json:"color"`
}

type Chat struct {
    ChatElem
    Extra []ChatElem `json:"Extra"`
}

func ReadChat(pr *PacketReader) (Chat, error) {
    data := pr.NextString()
    var c Chat
    err := json.Unmarshal([]byte(data), &c)
    return c, err
}

func(c Chat) String() string {
    t := ""
    t += c.Text
    for _, sib := range c.Extra {
        t += sib.Text
    }
    return t
}

func(c Chat) Bytes() []byte {
    p, err := json.Marshal(c)
    if err != nil {
        return []byte{}
    }
    return p
}
