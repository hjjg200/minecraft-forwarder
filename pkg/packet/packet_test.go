package packet

import (
    "bytes"
    "testing"
)

func TestVarInts(t *testing.T) {

    // SAMPLE VARINTS
    // Value        Hex bytes
    // 0            0x00
    // 1            0x01
    // 2            0x02
    // 127          0x7f
    // 128          0x80 0x01
    // 255          0xff 0x01
    // 2097151      0xff 0xff 0x7f
    // 2147483647   0xff 0xff 0xff 0xff 0x07
    // -1           0xff 0xff 0xff 0xff 0x0f
    // -2147483648  0x80 0x80 0x80 0x80 0x08

    samples := map[int32] []byte{
        0:           []byte{0x00},
        1:           []byte{0x01},
        2:           []byte{0x02},
        127:         []byte{0x7f},
        128:         []byte{0x80, 0x01},
        255:         []byte{0xff, 0x01},
        2097151:     []byte{0xff, 0xff, 0x7f},
        2147483647:  []byte{0xff, 0xff, 0xff, 0xff, 0x07},
        -1:          []byte{0xff, 0xff, 0xff, 0xff, 0x0f},
        -2147483648: []byte{0x80, 0x80, 0x80, 0x80, 0x08},
    }

    success := true

    for x, p := range samples {
        v := VarInt(x)
        result := bytes.Compare(p, v) == 0
        success = success && result
        t.Logf("%d) %x == %x, %t", x, p, v, result)
    }

    if !success {
        t.Fail()
    }

}
