package main

import (
	"os"
	"io"
	"io/ioutil"
	"fmt"
	"path"
	"bytes"
	"strings"
	"syscall"
	"github.com/tucnak/store"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/alecthomas/kingpin.v2"
)


type ESHSessionConfig struct {
	Name string
	Hostname string
	Port string
	Username string
	Password string
	KeyPath string
	IsCurrentSession bool
	WorkingDir string
}


var (
	esh_cli		= kingpin.New("esh", "easy SSH")

	/// ----
	add 		= esh_cli.Command("add", "Adds a SSH session to config.")
	addname		= add.Flag("name", "Name of session.").Required().String() 
	serverIP 	= add.Flag("server", "Server address.").Short('s').Required().PlaceHolder("127.0.0.1").String()
	user 		= add.Flag("user", "Username to connect with.").Short('u').Required().String()
	port 		= add.Flag("port", "Port to connect to.").Short('p').Default("22").String()
	keyPath 	= add.Flag("key", "Path to key.").PlaceHolder("/path/to/key").String()

	/// -----
	use 		= esh_cli.Command("use", "Use a specific ssh session")
	usename 	= use.Arg("name", "Name of session.").Required().String()

	/// ----
	listall		= esh_cli.Command("list-all", "List all saved SSH sessions.")

	/// ----
	clear		= esh_cli.Command("clear", "Clear all saved SSH sessions.")


	/// ----
	get			= esh_cli.Command("get", "Get some file or folder.")
	getpath		= get.Arg("getpath", "Path of file | folder to download.").Required().String()


	/// ----
	put			= esh_cli.Command("put", "Put some file or folder.")

)


var applicationConfig []*ESHSessionConfig // array holding saved sessions


/// MARK: - Core funcs

func MakeSSHClient(esh_conf *ESHSessionConfig) (client *ssh.Client, err error) {
	config := &ssh.ClientConfig{
		User: esh_conf.Username,
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
	} else {
		config.Auth = []ssh.AuthMethod{ssh.Password(esh_conf.Password)}
	}

	client, err = ssh.Dial("tcp", esh_conf.Hostname + ":" + esh_conf.Port, config)

	return
}


/// MARK: - GET | PUT funcs

func GetPath(path string) {
}

func PutPath(path string) {
	sess, err := MakeLiveSession(CurrentSession())
	if err != nil {
		panic("0. Error: " + err.Error())
	}

	defer sess.Close()

	lfile, err := os.Open("/usr/src/esh/p.zip")
	if err != nil {
		panic("1. Error: " + err.Error())
	}

	lfileStats, err := lfile.Stat()
	if err != nil {
		panic("2. Error: " + err.Error())
	}

	go func() {
		wrt, _ := sess.StdinPipe()

		fmt.Println("goroutine runnin")

		fmt.Fprintln(wrt, "C0644", lfileStats.Size(), "p.zip")

		if lfileStats.Size() > 0 {
			io.Copy(wrt, lfile)
			fmt.Fprint(wrt, "\x00")
			wrt.Close()
		} else {
			fmt.Fprint(wrt, "\x00")
			wrt.Close()
		}
	}()

	if err := sess.Run(fmt.Sprintf("scp -t %s", "/home/pi/test/p.zip")); err != nil {
		panic("3. Error: " + err.Error())
	}
}


/// MARK: - Session management funcs

func CurrentSession() (*ESHSessionConfig) {
	for _, val := range applicationConfig {
		if val.IsCurrentSession == true {
			return val
		}
	}
	return nil
}

func UseSession(sessionName string) {
	for _, val := range applicationConfig {
		if val.Name == sessionName {
			val.IsCurrentSession = true
		} else {
			val.IsCurrentSession = false
		}
	}
}

func ChangeSessionDir(toDir string) {
	sess := CurrentSession()
	
	prevPath := sess.WorkingDir
	if string(toDir[0]) == "/" {
		prevPath = "/"
	}
	
	sess.WorkingDir = path.Join(prevPath, toDir)
}

func ListSavedSessions() {
	for i, val := range applicationConfig {
		fmt.Println((i+1), "-" ,val.Name)
	}
}

func ClearCurrentSession() {
	for _, val := range applicationConfig {
		val.IsCurrentSession = false
	}
}

func AddSession(name, ip, port, user, keyPath string) {

	// pre-check if session already exists
	for _, val := range applicationConfig {
		if val.Name == name && val.Hostname == ip && val.Username == user {
			fmt.Println("Session already in config.")
			return
		}
	}

	new_sess := &ESHSessionConfig {
		Name: name,
		Hostname: ip,
		Port: port,
		Username: user,
		KeyPath: keyPath,
	}

	if new_sess.KeyPath == "" { // keypath wasn't provided, ask for password
		fmt.Print("Device Password: ")
		
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			panic("Error: " + err.Error())
		}

		fmt.Println()
		new_sess.Password = string(bytePassword)
	}

	applicationConfig = append(applicationConfig, new_sess)
}

func MakeLiveSession(esh_conf *ESHSessionConfig) (session *ssh.Session, err error) {	

	client, err := MakeSSHClient(esh_conf)

	if err != nil {
		return
	}

	session, err = client.NewSession()

	return
}


/// MARK: - Command line funcs

func ExecuteCommand(cmd_args []string, esh_conf *ESHSessionConfig) {
	cmd := strings.Join(cmd_args, " ")

	session, err := MakeLiveSession(esh_conf)

	if err != nil {
		panic("Error: " + err.Error())
	}

	defer session.Close()

	var stdoutBuf bytes.Buffer
    session.Stdout = &stdoutBuf
    session.Run("cd " + esh_conf.WorkingDir + ";" + cmd)

    fmt.Print(stdoutBuf.String())
}

func ParseArgs(args []string) {

	// check if theres is an `arg[1..n]` and if that arg is not one of the registered
	// app commands, and if current session isn't nil either, then execute given
	// command on ssh device
	if len(args) > 1 {
		command := args[1]

		/// TODO: better way to do this?
		if command != "list-all" && command != "use" && command != "add" && command != "clear" && command != "get" && command != "put" && command != "help" && command != "--help" {
			current_sess := CurrentSession()
			if current_sess != nil {
				// special case for 'cd' command
				if command == "cd" {
					ChangeSessionDir(args[2])
				} else {
					ExecuteCommand(args[1:], current_sess)
				}
			} else {
				fmt.Println("Switch to a session first.")
			}
			return
		}
	}

	switch kingpin.MustParse(esh_cli.Parse(os.Args[1:])) {
		// Add session
		case add.FullCommand():
			AddSession(*addname, *serverIP, *port, *user, *keyPath)

		// Use session
		case use.FullCommand():
			UseSession(*usename)

		// List all saved sessions
		case listall.FullCommand():
			ListSavedSessions()

		// Clear current sessions
		case clear.FullCommand():
			ClearCurrentSession()

		case get.FullCommand():
			GetPath(*getpath)

		case put.FullCommand():
			PutPath("")
	}
}

func main() {
	store.SetApplicationName("esh")

	// load config, errors are ignored for now as config file may not exist on first program run
	store.Load("config.json", &applicationConfig)

	ParseArgs(os.Args)

	err := store.Save("config.json", applicationConfig)
	if err != nil {
		panic("Error: " + err.Error())
	}
}
