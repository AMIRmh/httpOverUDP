package serverClient

import (
	"fmt"
	"net"
	"strconv"
	"encoding/binary"
	"udp-go/Workspace/pkg/myLib"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"github.com/miekg/dns"

)

const (
	IdSize = 10
)

var (
	specialMessage = 4294967295
	udpAddr *net.UDPAddr
	pc *net.UDPConn
	arr = make([]string, 0)
	clientFiles =  make(map[string][][]byte)
	clientSizes = make(map[string]int)
	Messages = make(chan Client)
	UDPPortServer, TCPPortServer string
)

type Client struct {
	ResponsePath *net.UDPAddr
	Id string
	Message []byte
}


func InitServer(UDPPort, TCPPort string) {
	UDPPortServer = UDPPort
	udpAddr , err := net.ResolveUDPAddr("udp4", ":" + UDPPortServer)

	if err != nil {
		fmt.Println(err)
		return
	}

	pc, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println(err)
		return
	}


	TCPPortServer = TCPPort
	server := http.NewServeMux()
	server.HandleFunc("/", handleDNSQuery)
	go func () {
		err := http.ListenAndServe(":" + TCPPortServer, server)
		myLib.CheckError(err)
	}()
}

func handleDNSQuery(res http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	myLib.CheckError(err)
	var raw map[string]interface{}
	json.Unmarshal(body, &raw)
	target := raw["target"].(string)
	typeOfQuery := raw["type"].(string)


	if typeOfQuery == "CNAME" {
		cname, err := net.LookupCNAME(target)
		myLib.CheckError(err)
		res.Write([]byte(cname))
	} else if typeOfQuery == "DNS" {
		ips, auth := queryDNS(target)
		resMap := map[string]string{"ips" : ips, "auth" : strconv.FormatBool(auth)}
		result, err := json.Marshal(resMap)
		myLib.CheckError(err)
		res.Write([]byte(result))
	}
}

func queryDNS(target string) (string, bool){
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(target), dns.TypeA)

	ips := ""
	in, err := dns.Exchange(m, "8.8.8.8:53")
	myLib.CheckError(err)
	for _, t := range in.Answer {
		ips += t.(*dns.A).A.String() + ", "
	}
	return ips, in.Authoritative
}

func ReadUDP() {
	buf := make([]byte, DataSize+PartSize+IdSize)
	for {
		_, remoteAddr, _ := pc.ReadFromUDP(buf[0:])
		go processUDP(buf, remoteAddr)
		buf = make([]byte, DataSize+PartSize+IdSize)
	}
}

func processUDP(buffer []byte, remoteAddr *net.UDPAddr) {
	id := buffer[0:IdSize]
	partBuffer := buffer[IdSize:IdSize+PartSize]
	part := int(binary.BigEndian.Uint32(partBuffer))
	data := buffer[IdSize+PartSize:]

	if part == specialMessage {
		specialMessageHandler(data, id, partBuffer, remoteAddr)
	} else {
		defer func() {
			if r := recover(); r == nil {
				fmt.Println("sending part")
				pc.WriteToUDP(partBuffer, remoteAddr)
			} else {
				fmt.Println("panic happend but recovered!!!!")
			}
		}()
		fmt.Println("part came: ", part)
		go putInArray(id, data, part)
	}
}

func specialMessageHandler(data, id, partBuffer []byte, remoteAddr *net.UDPAddr) {

	if trimNullString(data) == "end" {

		fmt.Println(string(id), " end recieved")
		writeToFileArray := make([]byte, 0)
		for _,d := range clientFiles[string(id)] {
			writeToFileArray = append(writeToFileArray, d...)
		}
		Messages <- Client{
			ResponsePath: remoteAddr,
			Id: string(id),
			Message: writeToFileArray,
			}
		delete(clientFiles, string(id))
		delete(clientSizes, string(id))
		pc.WriteToUDP(partBuffer, remoteAddr)

	} else if string(id) == DefaultId {

		newId := myLib.RandStringRunes(IdSize)
		pc.WriteToUDP([]byte(newId), remoteAddr)

	} else if size, err := strconv.Atoi(trimNullString(data)); err == nil {

		clientSizes[string(id)] = size

		if size < DataSize {
			clientFiles[string(id)] = make([][]byte, 1)
			clientFiles[string(id)][0] = make([]byte, DataSize)
		} else {
			s := (size + (DataSize-size%DataSize)) / DataSize
			clientFiles[string(id)] = make([][]byte, s)
			for i := 0; i < s; i++ {
				clientFiles[string(id)][i] = make([]byte, DataSize)
			}
			fmt.Println(len(clientFiles[string(id)]))
		}
		pc.WriteToUDP(partBuffer, remoteAddr)
	}
}

func putInArray(id, data []byte, part int) {
	clientFiles[string(id)][part] = data
}

func trimNullString(str []byte) string {
	var index int
	for i, s := range str {
		if s == byte(0) {
			index = i
			break
		}
	}
	return string(str[0:index])
}