package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net"
	"os"
)

const (
	DEFAULT_PORT = 61613
	BANNER       = `
███████╗██╗  ██╗███████╗██╗    ██╗███████╗███████╗██████╗ ██╗   ██╗███████╗██████╗ 
██╔════╝██║ ██╔╝██╔════╝██║    ██║██╔════╝██╔════╝██╔══██╗██║   ██║██╔════╝██╔══██╗
███████╗█████╔╝ █████╗  ██║ █╗ ██║███████╗█████╗  ██████╔╝██║   ██║█████╗  ██████╔╝
╚════██║██╔═██╗ ██╔══╝  ██║███╗██║╚════██║██╔══╝  ██╔══██╗╚██╗ ██╔╝██╔══╝  ██╔══██╗
███████║██║  ██╗███████╗╚███╔███╔╝███████║███████╗██║  ██║ ╚████╔╝ ███████╗██║  ██║
╚══════╝╚═╝  ╚═╝╚══════╝ ╚══╝╚══╝ ╚══════╝╚══════╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═╝
`
	STRAPLINE = "STOMP 1.2 Compatible message queueing server"
)

func main() {
	initLogging()

	fmt.Println(BANNER)
	fmt.Println(STRAPLINE)
	fmt.Println("\n")

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", DEFAULT_PORT))
	if err != nil {
		log.Error(fmt.Sprintf("Error listening on port %d: %s", DEFAULT_PORT, err.Error()))
		os.Exit(1)
	}
	log.Info(fmt.Sprintf("Listening on port %d...", DEFAULT_PORT))
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error(fmt.Sprintf("Error processing incoming connection: %s", err.Error()))
			os.Exit(1)
		}
		go handleIncomingConnection(conn)
	}
}

func initLogging() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
}

func handleIncomingConnection(conn net.Conn) {
	log.Info(fmt.Sprintf("Handling incoming connection from %s", conn.RemoteAddr()))
}
