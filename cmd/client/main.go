package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("failed, %v", err)
	}
}

func run() error {
	conn, err := net.Dial("tcp", "localhost:8999")
	if err != nil {
		return err
	}

	conn.Write([]byte("message from client"))
	_, err = fmt.Fprintf(conn, "command arg1 arg2")
	if err != nil {
		return err
	}

	status, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return err
	}

	fmt.Println("received ", status)

	return nil
}
