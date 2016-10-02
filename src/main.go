package main

import (
	"os"
	"io/ioutil"
	"fmt"
	"bytes"
	"strings"
	"github.com/tucnak/store"
	"golang.org/x/crypto/ssh"
	"gopkg.in/alecthomas/kingpin.v2"
)


var reservedCommands = [...]string {"list-all", "use", "add", "logout", "remove", "get", "put", "help", "--help", "-h"}

var applicationConfig []*ESHSessionConfig // array holding saved sessions

var (
	esh_cli		= kingpin.New("esh", "easy SSH")

	/// command and args for adding new ssh device
	add			= esh_cli.Command("add", "Adds a SSH session to config.")
	addname		= add.Flag("name", "Name of session.").Required().String()
	serverIP	= add.Flag("server", "Server address.").Short('s').Required().PlaceHolder("127.0.0.1").String()
	user		= add.Flag("user", "Username to connect with.").Short('u').Required().String()
	port		= add.Flag("port", "Port to connect to.").Short('p').Default("22").String()
	keyPath		= add.Flag("key", "Path to key.").PlaceHolder("/path/to/key").String()

	/// command and args for switching to a ssh connection
	use			= esh_cli.Command("use", "Use a specific ssh session")
	usename		= use.Arg("name", "Name of session.").Required().String()

	/// command for listing all ssh sessions
	listall		= esh_cli.Command("list-all", "List all saved SSH sessions.")

	/// command for logging out of ssh session
	logout		= esh_cli.Command("logout", "Logout from current session.")

	/// command and args for removing a ssh device
	remove		= esh_cli.Command("remove", "Remove a given session with name.")
	removename	= remove.Arg("name", "Name of session to remove.").Required().String()


	/// command and args for downloading a file/folder
	get			= esh_cli.Command("get", "Get some file or folder.")
	getpath		= get.Arg("getpath", "Path of file | folder to download.").Required().String()


	/// command and args for uploading a file/folder
	put			= esh_cli.Command("put", "Put some file or folder.")
	putpath		= put.Arg("putpath", "Path fo file | folder to upload.").Required().String()

)

/// MARK: Get / Put progress tracking

func (pt *ProgressTracker) Write(data []byte) (int, error) {
	pt.ProgressInt += len(data)
	pt.Progress.Set(pt.ProgressInt)
	return len(data), nil
}

/// MARK: - Core funcs

func MakeSSHClient(esh_conf *ESHSessionConfig) (client *ssh.Client, err error) {
	config := &ssh.ClientConfig{
		User: esh_conf.Username,
		Auth: []ssh.AuthMethod{ssh.Password(esh_conf.Password)},
	}

	if esh_conf.KeyPath != "" {

		keyBytes, er := ioutil.ReadFile(esh_conf.KeyPath)

		if er != nil {
			return
		}

		signer, er := ssh.ParsePrivateKey(keyBytes)
		if er != nil {
			return
		}

		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	}

	client, err = ssh.Dial("tcp", esh_conf.Hostname + ":" + esh_conf.Port, config)

	return
}

func ExecuteCommand(cmd string, esh_conf *ESHSessionConfig) bytes.Buffer {

	client, err := MakeSSHClient(esh_conf)
	if err != nil {
		panic("Error: " + err.Error())
	}

	session, err := client.NewSession()
	if err != nil {
		panic("Error: " + err.Error())
	}

	defer session.Close()

	var stdoutBuf bytes.Buffer
    session.Stdout = &stdoutBuf
    session.Stderr = &stdoutBuf
    session.Run("cd " + esh_conf.WorkingDir + ";" + cmd)

    return stdoutBuf
}

/// MARK: - Command line funcs

func ParseArgs(args []string) {
	// WHAT? -> check if theres is an `arg[1..n]` and if that arg is not one of the reservedCommands
	// app commands, and if current session isn't nil either, then execute given command on ssh device
	if len(args) > 1 {
		command := args[1]

		for _, _cmd := range reservedCommands {
			if command != _cmd {
				current_sess := CurrentSession()
				if current_sess != nil {
					// special case for `cd` command, basically we locally cd just to save server round-trip time
					if command == "cd" {
						ChangeSessionDir(args[2])
					} else {
						cmd := strings.Join(args[1:], " ")
						out := ExecuteCommand(cmd, current_sess)
						fmt.Print(out.String())
					}
				} else {
					fmt.Println("No valid session found, try switching to one first.")
				}

				return
			}
		}
	}

	switch kingpin.MustParse(esh_cli.Parse(os.Args[1:])) {

		case add.FullCommand():
			AddSession(*addname, *serverIP, *port, *user, *keyPath)

		case use.FullCommand():
			UseSession(*usename)

		case listall.FullCommand():
			ListSavedSessions()

		case logout.FullCommand():
			LogoutCurrentSession()

		case remove.FullCommand():
			RemoveSession(*removename)

		case get.FullCommand():
			GetPath(*getpath)

		case put.FullCommand():
			PutPath(*putpath)
	}
}

func main() {

	// add support for -h flag
	esh_cli.HelpFlag.Short('h')

	// store config name
	store.SetApplicationName("esh")

	// load config, errors are ignored for now as config file may not exist on first program run
	store.Load("config.json", &applicationConfig)

	// try to make something out of the given arguments
	ParseArgs(os.Args)

	// save config before exit, and panic if save fails
	err := store.Save("config.json", applicationConfig)
	if err != nil {
		panic("Unable to save config before exiting. Please report the error below.")
		panic("Error: " + err.Error())
	}
}
