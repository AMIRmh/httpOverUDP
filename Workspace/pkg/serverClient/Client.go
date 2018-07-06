package serverClient

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"encoding/binary"
	"time"
	"httpOverUDP/Workspace/pkg/myLib"
	"net/http"
	"bytes"
	"io/ioutil"
)

const (
	ReadTimeOut = 500 * time.Millisecond
)

var (
	udpClient = UDPClient{
		specialMessage: 4294967295,
		id: make([]byte, 10),
	}
	tcpClient = TCPClient{}
)

type UDPClient struct {
	specialMessage int
	udpAddr *net.UDPAddr
	udpListen *net.UDPConn
	windowsSize int
	id []byte
}

type TCPClient struct {
	tcpAddr *net.TCPAddr
}

type Packet struct {
	partNumber int
	data []byte
}

func InitClient(host ,UDPPort, TCPPort string) *net.UDPConn {
	udpClient.id = []byte(DefaultId)
	service := host + ":" + UDPPort
	udpClient.udpAddr, _ = net.ResolveUDPAddr("udp4", service)
	udpClient.udpListen = udpClient.createConnection(nil)
	udpClient.getId()

	service = host + ":" + TCPPort
	tcpClient.tcpAddr, _ = net.ResolveTCPAddr("tcp", service)
	return udpClient.udpListen
}

func (tc TCPClient) sendDNSQuery(data []byte) []byte {
	r := bytes.NewReader(data)
	resp, err := http.Post("http://" + tc.tcpAddr.String() + "/", "application/json", r)
	myLib.CheckError(err)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	return body
}

func (hc UDPClient) getId() {
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

func (hc UDPClient) fillId() {
	connection := hc.createConnection(nil)
	hc.asyncSendUDP(udpClient.id,udpClient.specialMessage, connection)
	connection.SetReadDeadline(time.Now().Add(ReadTimeOut))
	connection.Read(udpClient.id)
	if string(udpClient.id) == DefaultId {
		hc.fillId()
	}
}

func Send(data []byte, typeOfRequest string, nth int) {
	if typeOfRequest == "TCP" {
		result := tcpClient.sendDNSQuery(data)
		fmt.Println(string(result))
		Messages <- Client{Message: result, Id: "", ResponsePath: nil}
	} else if typeOfRequest == "UDP" {
		udpClient.windowsSize = nth
		fmt.Println("sending size")
		udpClient.sendSize(len(data))
		fmt.Println("sending chuncks")
		udpClient.sendChunk(data)
	} else {
		fmt.Println("request not supported.")
	}
}

func (hc UDPClient) sendChunk(input []byte) {
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
	wg.Add(udpClient.windowsSize)
	for i := 0; i < hc.windowsSize; i++ {
		go hc.sendThreadParts(packetsChannel, &wg)
	}
	wg.Wait()

	// end message to close the connection.
	fmt.Println("sending end to server")
	hc.syncSendUDP([]byte("end"), hc.specialMessage, hc.udpListen)
	hc.udpListen.Close()
	fmt.Println("finished sending")
}

func (hc UDPClient) partsQueue(input []byte, ch chan Packet, parts int) {
	for i := 0; i < parts; i++ {
		ch <- Packet{i, input[i * DataSize: (i + 1) * DataSize]}
	}
	close(ch)
}

func (hc UDPClient) sendThreadParts(ch chan Packet, wgThread *sync.WaitGroup) {
	defer wgThread.Done()
	conn := hc.createConnection(nil)
	for packet := range ch {
		hc.syncSendUDP(packet.data, packet.partNumber, conn)
	}
	conn.Close()
	fmt.Println("send part finished")
}

func (hc UDPClient) asyncSendUDP(dataUdp []byte, part int, udpConn *net.UDPConn) {
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

func (hc UDPClient) syncSendUDP(data []byte, part int, udpConn *net.UDPConn) {
	hc.asyncSendUDP(data, part, udpConn)
	var wg sync.WaitGroup
	wg.Add(1)
	hc.retrySendUDP(data, part, udpConn, &wg)
	wg.Wait()
}

func (hc UDPClient) retrySendUDP(data []byte, part int, udpConn *net.UDPConn, wg *sync.WaitGroup) {
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

func (hc UDPClient) sendSize(size int) {
	hc.syncSendUDP([]byte(strconv.Itoa(size)), hc.specialMessage, hc.createConnection(nil))
}

func (hc UDPClient) createConnection(laddr *net.UDPAddr) *net.UDPConn {
	connection, err := net.DialUDP("udp", laddr, hc.udpAddr)
	myLib.CheckError(err)

	return connection
}