package wire

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	ProtocolVersion   uint32 = 70016
	NodeServices      uint64 = 0x01
	MessageHeaderSize        = 24
)

type MessageHeader struct {
	Magic    [4]byte
	Command  [12]byte
	Length   [4]byte
	Checksum [4]byte
}

type Message struct {
	Magic       [4]byte
	Command     [12]byte
	CommandName string
	Length      uint32
	Checksum    [4]byte
	Payload     []byte
}

type NetAddress struct {
	Time     uint32
	Services uint64
	Host     string
	Port     uint16
}

func UintToCompactSize() {
}

// создает Bitcoin Message для входящего payload
func MessageBinaryByCommandPayload(command string, payload []byte) []byte {
	var res []byte

	header := []byte{0xF9, 0xBE, 0xB4, 0xD9} // Start string: Mainnet

	cmdBin := append([]byte(command), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}...) // Command name: version + null padding
	cmdBin = cmdBin[:12]

	payload_size := make([]byte, 4) // Payload size
	binary.LittleEndian.PutUint32(payload_size, uint32(len(payload)))

	sha1 := sha256.Sum256(payload)
	sha2 := sha256.Sum256(sha1[:])
	checksum := sha2[:4] // Checksum

	header = append(header, cmdBin...)
	header = append(header, payload_size...)
	header = append(header, checksum...)

	res = append(res, header...)
	res = append(res, payload...)

	return res
}

func NewNetAddress(services uint64, host string, port uint16) NetAddress {
	var res NetAddress = NetAddress{0, services, host, port}
	res.Time = uint32(time.Now().Unix())
	return res
}

// преобразует NetAddress в массив байт для использования в сообщениях
func NetAddressToBinary(na NetAddress) ([]byte, error) {
	var res []byte

	t := make([]byte, 4) // time
	binary.LittleEndian.PutUint32(t, na.Time)

	s := make([]byte, 8) // services
	binary.LittleEndian.PutUint64(s, na.Services)

	h := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF} // IPv6 host
	// добавляем к фиксированной части IPv4 адрес
	hostSplit := strings.Split(na.Host, ".")
	for _, b := range hostSplit {
		i, err := strconv.Atoi(b)
		if err != nil {
			return nil, errors.New("Error (WI-010101): " + fmt.Sprintf("%s\n", err))
		}
		h = append(h, byte(i))
	}

	p := make([]byte, 2) // port
	binary.BigEndian.PutUint16(p, na.Port)

	res = append(res, t...)
	res = append(res, s...)
	res = append(res, h...)
	res = append(res, p...)

	return res, nil
}

// читает из r bitcoin message header, заполняет структуру Message
func ReadMessageHeader(r io.Reader) (Message, error) {
	hdr := Message{}

	var headerBytes [MessageHeaderSize]byte

	_, err := io.ReadFull(r, headerBytes[:])
	if err != nil {
		return hdr, errors.New("Error (WI-010201): " + fmt.Sprintf("%s\n", err))
	}

	for i := 0; i < len(hdr.Magic); i++ {
		hdr.Magic[i] = headerBytes[i]
	}
	for i := 0; i < len(hdr.Command); i++ {
		hdr.Command[i] = headerBytes[i+4]
	}
	hdr.CommandName = strings.ReplaceAll(string(hdr.Command[:]), string(0x00), "")

	lengthByte := make([]byte, 4)
	for i := 0; i < 4; i++ {
		lengthByte[i] = headerBytes[i+16]
	}
	hdr.Length = binary.LittleEndian.Uint32(lengthByte)

	for i := 0; i < len(hdr.Checksum); i++ {
		hdr.Checksum[i] = headerBytes[i+20]
	}

	return hdr, nil
}

// читает bitcoin message из r, возвращает структуру Message
func ReadMessage(r io.Reader) (Message, error) {

	msg, err := ReadMessageHeader(r)
	if err != nil {
		return msg, errors.New("Error (WI-010301): " + fmt.Sprintf("%s\n", err))
	}

	payloadBytes := make([]byte, msg.Length)

	_, err = io.ReadFull(r, payloadBytes)
	if err != nil {
		return msg, errors.New("Error (WI-010202): " + fmt.Sprintf("%s\n", err))
	}
	msg.Payload = payloadBytes

	return msg, nil
}
