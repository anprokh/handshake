package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"time"
)

type MsgVersion struct {
	Version        uint32
	Services       uint64
	Timestamp      uint32
	ReceivingNA    NetAddress
	TransmittingNA NetAddress
	//	Nonce			uint64
	UserAgent   string
	StartHeight uint32
	Relay       int
}

// NewMsgVersion подготавливает и возвращает новую заполненную структуру MsgVersion
// ra = Receiving node, ta = Transmitting node, h = start_height
func NewMsgVersion(ra NetAddress, ta NetAddress, h uint32) MsgVersion {
	var res MsgVersion
	res.Version = ProtocolVersion
	res.Services = NodeServices
	res.Timestamp = uint32(time.Now().Unix())
	res.ReceivingNA = ra
	res.TransmittingNA = ta
	res.StartHeight = h
	res.Relay = 1
	return res
}

// SendMsgVersion подготавливает и отправляет сообщение Version через указанное соединение
func SendMsgVersion(conn *net.TCPConn) error {

	tcpAddr, err := net.ResolveTCPAddr("tcp", conn.RemoteAddr().String())
	if err != nil {
		return errors.New("Error (WI-020201): " + fmt.Sprintf("%s\n", err))
	}
	ra := NewNetAddress(0xFFFFFFFFFFFFFFFF, tcpAddr.IP.String(), uint16(tcpAddr.Port))

	localAddr, err := net.ResolveTCPAddr("tcp", conn.LocalAddr().String())
	if err != nil {
		return errors.New("Error (WI-020202): " + fmt.Sprintf("%s\n", err))
	}
	ta := NewNetAddress(NodeServices, "0.0.0.0", uint16(localAddr.Port))

	msgVersion := NewMsgVersion(ra, ta, 436904)

	payload, err := MsgVersionPayload(msgVersion)
	if err != nil {
		return errors.New("Error (WI-020203): " + fmt.Sprintf("%s\n", err))
	}

	version_message := MessageBinaryByCommandPayload("version", payload)

	_, err = conn.Write(version_message)
	if err != nil {
		return errors.New("Error (WI-020204) Write to server failed: " + fmt.Sprintf("%s\n", err))
	}

	return nil
}

// MsgVersionPayload преобразует MsgVersion в массив байт для использования в сообщении
func MsgVersionPayload(msg MsgVersion) ([]byte, error) {
	var payload []byte

	protocolBinary := make([]byte, 4) // Protocol version
	binary.LittleEndian.PutUint32(protocolBinary, ProtocolVersion)

	servicesBinary := make([]byte, 8) // Services
	binary.LittleEndian.PutUint64(servicesBinary, NodeServices)

	timestampBinary := make([]byte, 8) // [Epoch time]
	binary.LittleEndian.PutUint32(timestampBinary, uint32(time.Now().Unix()))

	recBinary, err := NetAddressToBinary(msg.ReceivingNA) // Receiving node's "network address"
	if err != nil {
		return nil, errors.New("Error (WI-020301) Receiving node's address binary transformation failed: " + fmt.Sprintf("%s\n", err))
	}
	recBinary = recBinary[4:] // time отсутствует в сообщении version

	transBinary, err := NetAddressToBinary(msg.TransmittingNA) // Transmitting node's "network address"
	if err != nil {
		return nil, errors.New("Error (WI-020302) Transmitting node's address binary transformation failed: " + fmt.Sprintf("%s\n", err))
	}
	transBinary = transBinary[4:] // time отсутствует в сообщении version

	rand.Seed(time.Now().UnixNano()) // рандомизируем генератор случ. чисел
	nonceBinary := make([]byte, 8)   // Nonce
	binary.LittleEndian.PutUint64(nonceBinary, rand.Uint64())

	agentBinary := []byte{0x10, 0x2F, 0x53, 0x61, 0x74, 0x6F, 0x73, 0x68, 0x69, 0x3A, 0x32, 0x32, 0x2E, 0x30, 0x2E, 0x30, 0x2F} // User agent: /Satoshi:22.0.0/

	startHeightBinary := make([]byte, 4) // Start height
	binary.LittleEndian.PutUint32(startHeightBinary, msg.StartHeight)

	payload = append(payload, protocolBinary...)
	payload = append(payload, servicesBinary...)
	payload = append(payload, timestampBinary...)
	payload = append(payload, recBinary...)
	payload = append(payload, transBinary...)
	payload = append(payload, nonceBinary...)
	payload = append(payload, agentBinary...)
	payload = append(payload, startHeightBinary...)
	payload = append(payload, byte(msg.Relay))

	return payload, nil
}

// пытается получить из r сообщение Version
func ReadMessageVersion(r io.Reader, version chan MsgVersion) {

	msg, err := ReadMessage(r)
	if err != nil {
		fmt.Printf("Read Version failed: %s\n", err.Error())
		return
	}
	if string(msg.Command[0:7]) != "version" {
		fmt.Printf("Read Version failed: '%s' received\n")
		return
	}

	msgVersion, err := MsgVersionDecode(msg.Payload)
	if err != nil {
		// казалось бы, что может пойти не так...
	}
	version <- msgVersion
}

// заполняет структуру MsgVersion из массива байт
// todo: ReceivingNA, TransmittingNA
func MsgVersionDecode(payload []byte) (MsgVersion, error) {
	var msgVersion MsgVersion

	protocolBinary := make([]byte, 4) // Protocol version
	for i := 0; i < 4; i++ {
		protocolBinary[i] = payload[i]
	}
	msgVersion.Version = binary.LittleEndian.Uint32(protocolBinary)

	servicesBinary := make([]byte, 8) // Services
	for i := 0; i < 8; i++ {
		servicesBinary[i] = payload[i+4]
	}
	msgVersion.Services = binary.LittleEndian.Uint64(servicesBinary)

	timestampBinary := make([]byte, 8) // [Epoch time]
	for i := 0; i < 8; i++ {
		timestampBinary[i] = payload[i+12]
	}
	msgVersion.Timestamp = binary.LittleEndian.Uint32(timestampBinary)

	recBinary := make([]byte, 26)
	for i := 0; i < 26; i++ {
		recBinary[i] = payload[i+20]
	}
	//fmt.Printf("  recBinary: %x\n", recBinary)

	transBinary := make([]byte, 26)
	for i := 0; i < 26; i++ {
		transBinary[i] = payload[i+46]
	}
	//fmt.Printf("transBinary: %x\n", transBinary)

	nonceBinary := make([]byte, 8) // Nonce
	for i := 0; i < 8; i++ {
		nonceBinary[i] = payload[i+72]
	}

	agentBinary := []byte{}
	for i := 0; i < int(payload[80]); i++ {
		agentBinary = append(agentBinary, payload[i+81])
	}
	msgVersion.UserAgent = string(agentBinary)

	offset := 81 + len(agentBinary)
	startHeightBinary := make([]byte, 4) // Start height
	for i := 0; i < 4; i++ {
		startHeightBinary[i] = payload[i+offset]
	}
	msgVersion.StartHeight = binary.LittleEndian.Uint32(startHeightBinary)
	msgVersion.Relay = int(payload[len(payload)-1])

	return msgVersion, nil
}
