package main

import (
	"os"
	"fmt"
	"bytes"
	"strings"
	"syscall"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"github.com/tucnak/store"
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
	app 		= kingpin.New("esh", "easy SSH")

	/// ----
	listall		= app.Command("list-all", "List all saved SSH sessions.")

	/// ----
	add 		= app.Command("add", "Adds a SSH session to config.")
	addname		= add.Arg("name", "Name of session.").Required().String() 
	serverIP 	= add.Flag("server", "Server address.").Short('h').Required().PlaceHolder("127.0.0.1").String()
	user 		= add.Flag("user", "Username to connect with.").Short('u').Required().String()
	port 		= add.Flag("port", "Port to connect to.").Short('p').Default("22").String()
	keyPath 	= add.Flag("key", "Path to key.").PlaceHolder("/path/to/key").String()

	/// -----
	use 		= app.Command("use", "Use a specific ssh session")
	usename 	= use.Arg("name", "Name of session.").Required().String()

	/// -----
	chdir 		= app.Command("cd", "Change working directory")
	chdirpath	= chdir.Arg("path", "new path").Required().String()
)


var applicationConfig []*ESHSessionConfig // array holding saved sessions


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
		}
	}
}

func ChangeSessionDir(toDir string) {
	CurrentSession().WorkingDir = toDir
}

func ListSavedSessions() {
	fmt.Println("-> All sessions")
	for i, val := range applicationConfig {
		fmt.Println((i+1), "-" ,val.Name)
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

	if keyPath == "" { // ask for password
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


/// MARK: - Command line funcs

func ExecuteCommand(cmd_args []string, esh_conf *ESHSessionConfig) {
	cmd := strings.Join(cmd_args, " ")

	session, err := MakeLiveSession(esh_conf)
	defer session.Close()

	if err != nil {
		panic("Error: " + err.Error())
		return
	}

	var stdoutBuf bytes.Buffer
    session.Stdout = &stdoutBuf
    session.Run("cd " + esh_conf.WorkingDir + ";" + cmd)

    fmt.Print(stdoutBuf.String())
}

func ParseArgs(args []string) {

	if len(args) > 1 && args[1] != "list-all" && args[1] != "use" && args[1] != "cd" && args[1] != "help" && args[1] != "--help" {
		if CurrentSession() != nil {
			// run custom command
			ExecuteCommand(args[1:], applicationConfig[0])
			return
		} else {
			// fmt.Println("Current session is `nil`. Try `esh use <name>`.")
		}
	}

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	// Add session
	case add.FullCommand():
		AddSession(*addname, *serverIP, *port, *user, *keyPath)

	// Use session
	case use.FullCommand():
		UseSession(*usename)
	
	// Change directory
	case chdir.FullCommand():
		ChangeSessionDir(*chdirpath)

	// List all saved sessions
	case listall.FullCommand():
		ListSavedSessions()
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
