package p2p

const (
	IncomingMessage byte = 0x0
	MsgStore        byte = 0x1
	MsgGet          byte = 0x2
	MsgFile         byte = 0x3
	MsgDelete       byte = 0x4
)

type RPC struct {
	From    string
	Type    byte
	Key     string
	Payload []byte
}
