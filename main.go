package main

import (
	"os"
	"fmt"
	"bytes"
	"strings"
	"golang.org/x/crypto/ssh"
)

type ESHConfig struct {
	Hostname string
	Port string
	Username string
	Password string
	Key string
}

func MakeSession (esh_conf *ESHConfig) (session *ssh.Session, err error) {
	
	config := &ssh.ClientConfig{
		User: esh_conf.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(esh_conf.Password), // TODO: deal with key
		},
	}

	conn, err := ssh.Dial("tcp", esh_conf.Hostname + ":" + esh_conf.Port, config)

	if err != nil {
		return
	}

	session, err = conn.NewSession()

	return
}

func Command (cmd string) {
	session, err := MakeSession(&ESHConfig{Hostname: "192.168.0.11", Port: "22", Username: "pi", Password: "raspberry"})
	defer session.Close()

	if err != nil {
		panic("Error: " + err.Error())
		return
	}

	var stdoutBuf bytes.Buffer
    session.Stdout = &stdoutBuf
    session.Run(cmd)

    fmt.Println(stdoutBuf.String())
}

func ParseArgs (args []string) {

	if args[1] == "-n" {
		// args[2] will contain some ssh session name
		// args[3..n] will be the command
	} else if (args[1] == "list-all") {
		// list all saved ssh sessions
	} else {
		// args[1..n] is the command, because a session has been selected
		cmd := strings.Join(args[1:], " ")
		Command(cmd)
	}
}

func main() {
	ParseArgs(os.Args)
}