package main

import (
	"httpOverUDP/Workspace/pkg/serverClient"
	"fmt"
	"strings"
)

const (
	Host = "127.0.0.1"
	UDPPort = "1313"
	TCPPort = "80"
	NumberOfThreads = 5
)

func main() {
	conn := serverClient.InitClient(Host, UDPPort, TCPPort)
	//serverClient.Send([]byte("GET /\n\rHOST: ceit.aut.ac.ir\n\r"), "UDP", NumberOfThreads)
	serverClient.Send([]byte("{\"target\": \"google.com\", \"type\": \"DNS\"}"), "TCP", NumberOfThreads)
	fmt.Println(conn.LocalAddr().String())
	s := strings.Split(conn.LocalAddr().String(), ":")

	serverClient.InitServer(s[1], TCPPort)
	go serverClient.ReadUDP()

	for client := range serverClient.Messages {
		fmt.Println(string(client.Message))
	}
}