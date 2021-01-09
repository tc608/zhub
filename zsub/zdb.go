package zsub

var (
	chanMessages = make(chan Message, 1000) //接收到的 所有消息数据
)

// 数据封装
type Message struct {
	Conn *ZConn
	Rcmd []string
}
