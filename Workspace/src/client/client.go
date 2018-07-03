package client

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"encoding/binary"
	"time"
	"udp-go/Workspace/pkg/myLib"
)

const (
	DataSize = 1500
	PartSize = 4
	DefaultId = "1234567890"
	ReadTimeOut = 500 * time.Millisecond
)

var (
	httpClient = HTTPCleint{
		specialMessage: 4294967295,
		id: make([]byte, 10),
	}
)

type HTTPCleint struct {
	specialMessage int
	udpAddr *net.UDPAddr
	windowsSize int
	id []byte
}

type Packet struct {
	partNumber int
	data []byte
}

func InitClient(host ,port string) {
	httpClient.id = []byte(DefaultId)
	service := host + ":" + port
	httpClient.udpAddr, _ = net.ResolveUDPAddr("udp4", service)
	httpClient.getId()
}

func (hc HTTPCleint) getId() {
	go hc.fillId()
	for {
		if string(hc.id) == DefaultId {
			fmt.Println("retying to connect")
			time.Sleep(1000 * time.Millisecond)
		} else {
			break
		}
	}
}

func (hc HTTPCleint) fillId() {
	connection := hc.createConnection()
	hc.asyncSendUDP(httpClient.id,httpClient.specialMessage, connection)
	connection.SetReadDeadline(time.Now().Add(ReadTimeOut))
	connection.Read(httpClient.id)
	if string(httpClient.id) == DefaultId {
		hc.fillId()
	}
}

func Send(data []byte, typeOfRequest string, nth int) {
	if typeOfRequest == "DNS" {

	} else if typeOfRequest == "HTTP" {
		httpClient.windowsSize = nth
		fmt.Println("sending size")
		httpClient.sendSize(len(data))
		fmt.Println("sending chuncks")
		httpClient.sendChunk(data)
	} else {
		fmt.Println("request not supported.")
	}
}

func (hc HTTPCleint) sendChunk(input []byte) {
	// padding
	var appendArray []byte
	if len(input) > DataSize {
		appendArray = make([]byte, DataSize-len(input)%DataSize)
	} else {
		appendArray = make([]byte, DataSize-len(input))
	}
	fmt.Println(len(input) % DataSize)
	input = append(input, appendArray...)
	parts := len(input) / DataSize
	fmt.Println(len(input) % DataSize)
	
	// queue created
	packetsChannel := make(chan Packet, parts)
	go hc.partsQueue(input, packetsChannel, parts)
	
	// sending parts in windows size
	var wg sync.WaitGroup
	wg.Add(httpClient.windowsSize)
	for i := 0; i < hc.windowsSize; i++ {
		go hc.sendThreadParts(packetsChannel, &wg)
	}
	wg.Wait()

	// end message to close the connection.
	fmt.Println("sending end to server")
	hc.syncSendUDP([]byte("end"), hc.specialMessage, hc.createConnection())
	fmt.Println("finished sending")
}

func (hc HTTPCleint) partsQueue(input []byte, ch chan Packet, parts int) {
	for i := 0; i < parts; i++ {
		ch <- Packet{i, input[i * DataSize: (i + 1) * DataSize]}
	}
	close(ch)
}

func (hc HTTPCleint) sendThreadParts(ch chan Packet, wgThread *sync.WaitGroup) {
	defer wgThread.Done()
	conn := hc.createConnection()
	for packet := range ch {
		hc.syncSendUDP(packet.data, packet.partNumber, conn)
	}
	fmt.Println("send part finished")
}

func (hc HTTPCleint) asyncSendUDP(dataUdp []byte, part int, udpConn *net.UDPConn) {
	arr := make([]byte, PartSize)
	binary.BigEndian.PutUint32(arr, uint32(part))
	arr = append(arr, dataUdp...)
	data := hc.id
	data = append(data, arr...)
	_, err := udpConn.Write(data)
	if err != nil {
		fmt.Println("here1")
		fmt.Println(err)
	}
}

func (hc HTTPCleint) syncSendUDP(data []byte, part int, udpConn *net.UDPConn) {
	hc.asyncSendUDP(data, part, udpConn)
	var wg sync.WaitGroup
	wg.Add(1)
	hc.retrySendUDP(data, part, udpConn, &wg)
	wg.Wait()
}

func (hc HTTPCleint) retrySendUDP(data []byte, part int, udpConn *net.UDPConn, wg *sync.WaitGroup) {
	buf := make([]byte, 10)
	udpConn.SetReadDeadline(time.Now().Add(ReadTimeOut))
	_, err := udpConn.Read(buf[0:])

	if err != nil {
		hc.asyncSendUDP(data, part, udpConn)
		go hc.retrySendUDP(data, part, udpConn, wg)
	} else {
		wg.Done()
	}
}

func (hc HTTPCleint) sendSize(size int) {
	hc.syncSendUDP([]byte(strconv.Itoa(size)), hc.specialMessage, hc.createConnection())
}

func (hc HTTPCleint) createConnection() *net.UDPConn {
	connection, err := net.DialUDP("udp", nil, hc.udpAddr)
	myLib.CheckError(err)
	return connection
}