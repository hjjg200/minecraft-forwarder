package packet

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

func(c Chat) String() string {
    t := ""
    t += c.Text
    for _, sib := range c.Extra {
        t += sib.Text
    }
    return t
}
