package main

import (
	"os"
	"io"
	"io/ioutil"
	"fmt"
	"path"
	"path/filepath"
	"bytes"
	"strings"
	"syscall"
	"github.com/tucnak/store"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/cheggaaa/pb.v1"
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


type UploadProgressTracker struct{
	Length int64
	Uploaded int
	Name string
	pbar *pb.ProgressBar
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
	logout		= esh_cli.Command("logout", "Logout from current session.")

	/// ----
	remove		= esh_cli.Command("remove", "Remove a given session with name.")
	removename 	= remove.Arg("name", "Name of session to remove.").Required().String()


	/// ----
	get			= esh_cli.Command("get", "Get some file or folder.")
	getpath		= get.Arg("getpath", "Path of file | folder to download.").Required().String()


	/// ----
	put			= esh_cli.Command("put", "Put some file or folder.")
	putpath 	= put.Arg("putpath", "Path fo file | folder to upload.").Required().String()

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
	// TODO: implement
}

func (pt *UploadProgressTracker) Write(data []byte) (int, error) {
	pt.Uploaded += len(data)
	pt.pbar.Set(pt.Uploaded)
	// fmt.Printf("\r%.2f%%", ((float32(pt.Uploaded) / float32(pt.Length))*float32(100)))
	return len(data), nil
}

func PutFile(fpath, root string, client *ssh.Client, prbar *pb.ProgressBar) string {

	l_sess := CurrentSession()
	sess, err := client.NewSession()
	if err != nil {
		panic("Error: " + err.Error())
	}
	defer sess.Close()

	lfile, err := os.Open(fpath)
	if err != nil {
		panic("Error: " + err.Error())
	}

	lfileStats, err := lfile.Stat()
	if err != nil {
		panic("Error: " + err.Error())
	}

	cprog := &UploadProgressTracker{ Length: lfileStats.Size(), Name: fpath, pbar: prbar }
	progressbarreader := io.TeeReader(lfile, cprog)
	// bar := pb.StartNew(int(lfileStats.Size())).SetUnits(pb.U_BYTES).Prefix("Prefix")
	// pb.New(200).Prefix("First ")

	go func() {

		wrt, _ := sess.StdinPipe()

		fmt.Fprintln(wrt, "C0644", lfileStats.Size(), "p.zip")

		if lfileStats.Size() > 0 {
			// io.Copy(wrt, bar.NewProxyReader(lfile))
			io.Copy(wrt, progressbarreader)
			fmt.Fprint(wrt, "\x00")
			wrt.Close()
		} else {
			fmt.Fprint(wrt, "\x00")
			wrt.Close()
		}
	}()
	
	a_components := strings.Split(root, string(os.PathSeparator))
	b_components := strings.Split(fpath, string(os.PathSeparator))
	diffs := b_components[len(a_components)-1:]
	diffs = append([]string{l_sess.WorkingDir, filepath.Base(root)}, diffs...)

	remotepath := strings.Join(diffs, string(os.PathSeparator))

	folder := path.Dir(remotepath)

	// here we escape the path string by wrapping it in quotes, so any space characters dont bite
	if err := sess.Run(fmt.Sprintf("mkdir -p \"%s\"; scp -t \"%s\"", folder, remotepath)); err != nil {
		panic("Error: " + err.Error())
	}

	return fpath
}

func DispatchUploads(putpath string, client *ssh.Client, putfiles []string, pbars []*pb.ProgressBar) {
	pool, _ := pb.StartPool(pbars...)

	results := make(chan string)
	for i, fl := range putfiles {
		go func (fl string, i int) {
			results <- PutFile(fl, putpath, client, pbars[i])
		}(fl, i)
	}

	// wait for all uploads to finish before exiting
	for _ = range putfiles {
        <-results
    }

    pool.Stop()
}

func PutPath(putpath string) {

	client, err := MakeSSHClient(CurrentSession())
	if err != nil {
		panic("Error: " + err.Error())
	}

	var progressBars []*pb.ProgressBar
	var putfiles []string
	filepath.Walk(putpath, func (fpath string,  info os.FileInfo, err error) error {
		if !info.IsDir() {
			putfiles = append(putfiles, fpath)
			bar := pb.New(int(info.Size())).SetUnits(pb.U_BYTES) //.Prefix("Prefix")
			progressBars = append(progressBars, bar)
		}
		return nil
	})

	// batch uploads of `n`, since opening too many SSH sessions at same time, causes connection disruption
	for i := 0; i < len(putfiles); i = i + 3 {
		max_index := i+3
		if max_index > len(putfiles) {
			max_index = len(putfiles)
		}
		DispatchUploads(putpath, client, putfiles[i:max_index], progressBars[i:max_index])
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
	
	current_sess := CurrentSession()

	for i, val := range applicationConfig {
		print_str := fmt.Sprintf("%d - %s", (i+1), val.Name)
		if current_sess != nil && current_sess.Name == val.Name {
			print_str = print_str + " " + "*"
		}
		fmt.Println(print_str)
	}
}

func LogoutCurrentSession() {
	for _, val := range applicationConfig {
		val.IsCurrentSession = false
	}
}

func RemoveSession(name string) {
	var deleteIndex int
	for i, val := range applicationConfig {
		if val.Name == name {
			deleteIndex = i
			break
		}
	}

	applicationConfig = append(applicationConfig[:deleteIndex], applicationConfig[deleteIndex+1:]...)
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
		if command != "list-all" && command != "use" && command != "add" && command != "logout" && command != "remove" && command != "get" && command != "put" && command != "help" && command != "--help" && command != "-h" {
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
	// support for -h flag
	esh_cli.HelpFlag.Short('h')

	store.SetApplicationName("esh")

	// load config, errors are ignored for now as config file may not exist on first program run
	store.Load("config.json", &applicationConfig)

	ParseArgs(os.Args)

	err := store.Save("config.json", applicationConfig)
	if err != nil {
		panic("Error: " + err.Error())
	}
}
