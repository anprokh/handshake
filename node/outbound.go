package node

import (
	"fmt"
	"github.com/fatih/color"
	"handshake/wire"
	"net"
	"time"
)

type Peer struct {
	Conn           net.Conn
	VerAckReceived bool
}

// Handshake выполняет описанную в протоколе процедуру установки соединения
// отправка сообщения Version и получение ответного сообщения VerAck
func (p *Peer) Handshake(done chan struct{}) {

	servHosts := []string{"71.201.9.208", "27.33.160.196", "104.62.47.181", "94.154.159.99"}

	for _, servHost := range servHosts {
		servAddr := fmt.Sprintf("%s:%s", servHost, "8333")
		fmt.Fprintf(color.Output, "%s %s %s %s\n", color.GreenString("[info]"), color.CyanString(time.Now().Format("2006-01-02 15:04:05")), color.GreenString("Trying to connect:"), servHost)

		tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
		if err != nil {
			fmt.Printf("Receiving node ResolveTCPAddr failed (%s): %s\n", servAddr, err.Error())
			continue
		}

		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			fmt.Printf("Dial failed (%s): %s\n", servAddr, err.Error())
			conn.Close()
			continue
		}
		fmt.Fprintf(color.Output, "%s %s %s %s\n", color.GreenString("[info]"), color.CyanString(time.Now().Format("2006-01-02 15:04:05")), color.GreenString("Connect OK:"), conn.RemoteAddr().String())

		err = wire.SendMsgVersion(conn)
		if err != nil {
			fmt.Printf("Send Version message (%s): %s\n", servAddr, err.Error())
			conn.Close()
			continue
		}
		fmt.Fprintf(color.Output, "%s %s %s %s\n", color.GreenString("[info]"), color.CyanString(time.Now().Format("2006-01-02 15:04:05")), color.GreenString("Send Version message OK:"), "waiting for a response...")

		version := make(chan wire.MsgVersion)

		go wire.ReadMessageVersion(conn, version)

		select {
		case msgVersion := <-version:
			// при получении ответного сообщения просто сообщим параметры удаленного узла
			fmt.Fprintf(color.Output, "%s %s %s %s\n", color.GreenString("[info]"), color.CyanString(time.Now().Format("2006-01-02 15:04:05")), color.GreenString("Command received:"), "version")
			fmt.Fprintf(color.Output, "%s Response Version message received!\n", color.GreenString("[info]"))
			fmt.Fprintf(color.Output, "%s Protocol version: %d\n", color.GreenString("[info]"), msgVersion.Version)
			fmt.Fprintf(color.Output, "%s User agent: %s\n", color.GreenString("[info]"), msgVersion.UserAgent)
			fmt.Fprintf(color.Output, "%s Start height: %d\n", color.GreenString("[info]"), msgVersion.StartHeight)

		case <-time.After(time.Second * 1):
			fmt.Fprintf(color.Output, "%s %s %s\n", color.RedString("[error]"), color.CyanString(time.Now().Format("2006-01-02 15:04:05")), color.YellowString("Response Version message failed: version timeout"))
			conn.Close()
			continue
		}

		// принимаем поступающие команды до получения команды verack
		anotherCommand := true
		for anotherCommand {

			ch := make(chan wire.Message)

			// просто заглушка - последовательно получаем поступающие команды
			go func() {
				msg, err := wire.ReadMessage(conn)
				if err != nil {
					return
				}
				ch <- msg
			}()

			select {
			case msg := <-ch:

				fmt.Fprintf(color.Output, "%s %s %s %s\n", color.GreenString("[info]"), color.CyanString(time.Now().Format("2006-01-02 15:04:05")), color.GreenString("Command received:"), msg.CommandName)
				if msg.CommandName == "verack" {
					anotherCommand = false
					p.Conn = conn
					p.VerAckReceived = true
					done <- struct{}{}
					return
				}

			case <-time.After(time.Second * 1):
				fmt.Println("Response VerAck message failed: verack timeout")
				conn.Close()
				anotherCommand = false
			}

		}
	}

}

// просто закрываем соединение
func (p *Peer) Disconnect() {
	if p.Conn == nil {
		return
	}
	err := p.Conn.Close()
	if err == nil {
		fmt.Fprintf(color.Output, "%s %s %s\n", color.GreenString("[info]"), color.CyanString(time.Now().Format("2006-01-02 15:04:05")), color.GreenString("Disconnect OK"))
	}
}
