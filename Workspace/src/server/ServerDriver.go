package main

import (
	"httpOverUDP/Workspace/pkg/serverClient"
	"httpOverUDP/Workspace/pkg/myLib"
	"strconv"
)

const (
	numberOfThreads = 5
	UDPPortServer = "1313"
	TCPPortServer = "80"
)


func main() {
	//source := flag.String("s", "udp:127.0.0.1:3000", "PROTOCOL:IP:PORT")
	//destination := flag.String("d", "tcp", "protocol to use for ")


	serverClient.InitServer(UDPPortServer, TCPPortServer)
	go serverClient.ReadUDP()
	for client := range serverClient.Messages {
		message := client.Message
		response := myLib.SendHTTPRequest(message)
		host := client.ResponsePath.IP.String()
		port := strconv.Itoa(client.ResponsePath.Port)
		serverClient.InitClient(host, port, TCPPortServer)
		serverClient.Send(response, "UDP", numberOfThreads)
	}
}