package main

import (
	"fmt"
	"bytes"
	"golang.org/x/crypto/ssh"
)

type ESHConfig struct {
	Hostname string
	Port string
	Username string
	Password string
	Key string
}

func MakeSession(esh_conf *ESHConfig) (session *ssh.Session, err error) {
	
	config := &ssh.ClientConfig{
		User: esh_conf.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(esh_conf.Password),
		},
	}

	conn, err := ssh.Dial("tcp", esh_conf.Hostname + ":" + esh_conf.Port, config)

	if err != nil {
		return
	}

	session, err = conn.NewSession()

	return
}

func main() {
	fmt.Println("hello, lets go!")

	session, err := MakeSession(&ESHConfig{Hostname: "192.168.0.11", Port: "22", Username: "pi", Password: "raspberry"})
	defer session.Close()

	if err != nil {
		panic("Error: " + err.Error())
		return
	}
	var stdoutBuf bytes.Buffer
    session.Stdout = &stdoutBuf
    session.Run("ls")

    fmt.Println(stdoutBuf.String())
}