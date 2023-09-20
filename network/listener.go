package network

import (
	"fmt"
	"os"
)

// NewConnectionListener 新连接事件监听器
type NewConnectionListener struct{}

func (l *NewConnectionListener) OnEvent(event *Event) {
	fmt.Println("New connection:", event.Conn.RemoteAddr().String())
}

// DataReceivedListener 数据接收事件监听器
type DataReceivedListener struct{}

func (l *DataReceivedListener) OnEvent(event *Event) {

	// 解包
	pgResData, err := event.Protocol.Decode(event.Conn, event.Data)
	if err != nil {
		fmt.Println("数据解包出错:", err.Error())
		return
	}

	// 发送应答响应
	echo, err := event.Protocol.Encode(pgResData)
	if len(echo) > 0 && err == nil {
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to encode data: %v\n", err)
			return
		}
		_, err := event.Conn.Write(echo)
		if err != nil {
			fmt.Println("Error occurred:", err.Error())
			return
		}
	}
	return
}

// ConnectionClosedListener 连接关闭事件监听器
type ConnectionClosedListener struct{}

func (l *ConnectionClosedListener) OnEvent(event *Event) {
	fmt.Println("Connection closed:", event.Conn.RemoteAddr().String())
}