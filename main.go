package main

import (
	"os"
	"io"
	"io/ioutil"
	"fmt"
	"bytes"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"github.com/tucnak/store"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/cheggaaa/pb.v1"
	"github.com/pkg/sftp"
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

type ProgressTracker struct{
	Length int64
	ProgressInt int
	Name string
	Progress *pb.ProgressBar
}

var applicationConfig []*ESHSessionConfig // array holding saved sessions


var (
	esh_cli		= kingpin.New("esh", "easy SSH")

	/// ----
	add		= esh_cli.Command("add", "Adds a SSH session to config.")
	addname		= add.Flag("name", "Name of session.").Required().String()
	serverIP	= add.Flag("server", "Server address.").Short('s').Required().PlaceHolder("127.0.0.1").String()
	user		= add.Flag("user", "Username to connect with.").Short('u').Required().String()
	port		= add.Flag("port", "Port to connect to.").Short('p').Default("22").String()
	KeyPath		= add.Flag("key", "Path to key.").PlaceHolder("/path/to/key").String()

	/// -----
	use		= esh_cli.Command("use", "Use a specific ssh session")
	usename		= use.Arg("name", "Name of session.").Required().String()

	/// ----
	listall		= esh_cli.Command("list-all", "List all saved SSH sessions.")

	/// ----
	logout		= esh_cli.Command("logout", "Logout from current session.")

	/// ----
	remove		= esh_cli.Command("remove", "Remove a given session with name.")
	removename	= remove.Arg("name", "Name of session to remove.").Required().String()


	/// ----
	get			= esh_cli.Command("get", "Get some file or folder.")
	getpath		= get.Arg("getpath", "Path of file | folder to download.").Required().String()


	/// ----
	put		= esh_cli.Command("put", "Put some file or folder.")
	putpath		= put.Arg("putpath", "Path fo file | folder to upload.").Required().String()

)

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

/// MARK: Get / Put progress tracking

func (pt *ProgressTracker) Write(data []byte) (int, error) {
	pt.ProgressInt += len(data)
	pt.Progress.Set(pt.ProgressInt)
	return len(data), nil
}

/// MARK: Get Funcs

func GetFile(client *sftp.Client, getfilepath string, prbar *pb.ProgressBar) string {

	l_sess := CurrentSession()
	remoteFile, err := client.Open(getfilepath)
	if err != nil {
		panic("Error: " + err.Error())
	}
	defer remoteFile.Close()

	rfileStats, err := remoteFile.Stat()
	if err != nil {
		panic("Error: " + err.Error())
	}

	cprog := &ProgressTracker{ Length: rfileStats.Size(), Name: getfilepath, Progress: prbar }
	remote_progressbarreader := io.TeeReader(remoteFile, cprog)

	remote_file_path_components := strings.Split(getfilepath, string(os.PathSeparator))
	remote_wd_components := strings.Split(l_sess.WorkingDir, string(os.PathSeparator))
	relative_remote_path := remote_file_path_components[len(remote_wd_components) - 1:]

	cwd, err := os.Getwd()
	if err != nil {
		panic("Error: " + err.Error())
	}
	write_path := strings.Join(append([]string{cwd}, relative_remote_path...), string(os.PathSeparator))
	write_path_folder := filepath.Dir(write_path)

	err = os.MkdirAll(write_path_folder, 0777)
	if err != nil {
		panic("Error: " + err.Error())
	}

	local_writer, _ := os.Create(write_path)
	io.Copy(local_writer, remote_progressbarreader)

	return getfilepath
}

func BatchDownload(client *sftp.Client, getfiles []string, pbars []*pb.ProgressBar) {
	pool, err := pb.StartPool(pbars...)
	if err != nil {
		panic("Error: " + err.Error())
	}

	results := make(chan string)
	for i := 0; i < len(getfiles); i++ {
		go func (idx int) {
			results <- GetFile(client, getfiles[idx], pbars[idx])
		}(i)
	}

	// wait for all uploads to finish before exiting
	for _ = range getfiles {
        <-results
    }

    pool.Stop()
}

func GetPath(getpath string) {

	l_sess := CurrentSession()

	client, err := MakeSSHClient(l_sess)
	if err != nil {
		panic("Error: " + err.Error())
	}

	sftp, err := sftp.NewClient(client)
	if err != nil {
		panic("Error: " + err.Error())
	}
	defer sftp.Close()

	remoteWalkingPath := path.Clean(strings.Join([]string {l_sess.WorkingDir, getpath}, string(os.PathSeparator)))

	var progressBars []*pb.ProgressBar
	var getfiles []string

	walker := sftp.Walk(remoteWalkingPath)
	for walker.Step() {
		if walker.Stat().IsDir() || walker.Err() != nil {
			continue
		}

		getfiles = append(getfiles, walker.Path())
		bar := pb.New(int(walker.Stat().Size())).SetUnits(pb.U_BYTES).Prefix(path.Base(walker.Path()))
		progressBars = append(progressBars, bar)
	}

	// batch downloads of `n`
	for i := 0; i < len(getfiles); i = i + 3 {
		max_index := i+3
		if max_index > len(getfiles) {
			max_index = len(getfiles)
		}

		BatchDownload(sftp, getfiles[i:max_index], progressBars[i:max_index])
	}
}

/// MARK: - PUT funcs

func PutFile(client *ssh.Client, putfilepath string, prbar *pb.ProgressBar) string {

	l_sess := CurrentSession()
	sess, err := client.NewSession()
	if err != nil {
		panic("Error: " + err.Error())
	}
	defer sess.Close()

	lfile, err := os.Open(putfilepath)
	if err != nil {
		panic("Error: " + err.Error())
	}

	lfileStats, err := lfile.Stat()
	if err != nil {
		panic("Error: " + err.Error())
	}

	cprog := &ProgressTracker{ Length: lfileStats.Size(), Name: putfilepath, Progress: prbar }
	progressbarreader := io.TeeReader(lfile, cprog)

	local_file_path_components := strings.Split(putfilepath, string(os.PathSeparator))
	local_file_name := local_file_path_components[len(local_file_path_components) - 1]

	go func() {

		wrt, _ := sess.StdinPipe()

		fmt.Fprintln(wrt, "C0644", lfileStats.Size(), local_file_name)

		if lfileStats.Size() > 0 {
			io.Copy(wrt, progressbarreader)
			fmt.Fprint(wrt, "\x00")
			wrt.Close()
		} else {
			fmt.Fprint(wrt, "\x00")
			wrt.Close()
		}
	}()

	// since local and remote paths differ and we're not really given a remote path by user,
	// we compute the remote path here which is the current working directory on remote

	cwd, err := os.Getwd()
	if err != nil {
		panic("Error: " + err.Error())
	}

	cwd_components := strings.Split(cwd, string(os.PathSeparator))

	relative_file_path_components := local_file_path_components[len(cwd_components):]
	relative_file_path_components = append([]string{l_sess.WorkingDir}, relative_file_path_components...)

	remote_file_path := strings.Join(relative_file_path_components, string(os.PathSeparator))
	remote_file_path_folder := strings.Join(relative_file_path_components[:len(relative_file_path_components) - 1], string(os.PathSeparator))

	// this retrieves the uploaded stream, and writes it to a file, also creating the folder path beforehand
	if err := sess.Run(fmt.Sprintf("mkdir -p \"%s\"; scp -t \"%s\"", remote_file_path_folder, remote_file_path)); err != nil {
		panic("Error: " + err.Error())
	}

	return putfilepath
}

func BatchUpload(client *ssh.Client, putfiles []string, pbars []*pb.ProgressBar) {

	pool, err := pb.StartPool(pbars...)
	if err != nil {
		panic("Error: " + err.Error())
	}

	results := make(chan string)
	for i := 0; i < len(putfiles); i++ {
		go func (idx int) {
			results <- PutFile(client, putfiles[idx], pbars[idx])
		}(i)
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
	filepath.Walk(putpath, func (fpath string, info os.FileInfo, err error) error {
		// walk the given path and store all files in an array
		if !info.IsDir() {
			fAbsPath, err := filepath.Abs(fpath) // absolute paths are easier to deal with in `PutFile` func when joining paths with remote
			if err == nil {
				putfiles = append(putfiles, fAbsPath)
				bar := pb.New(int(info.Size())).SetUnits(pb.U_BYTES).Prefix(path.Base(fpath))
				progressBars = append(progressBars, bar)
			}
		}
		return nil
	})

	// batch uploads of `n`, since opening too many SSH sessions at same time, causes connection disruption
	for i := 0; i < len(putfiles); i = i + 3 {
		max_index := i+3
		if max_index > len(putfiles) {
			max_index = len(putfiles)
		}

		BatchUpload(client, putfiles[i:max_index], progressBars[i:max_index])
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

	sess.WorkingDir = filepath.Clean(path.Join(prevPath, toDir))
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
		WorkingDir: "/",
	}

	// keypath wasn't provided, ask for password
	if new_sess.KeyPath == "" {
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

/// MARK: - Command line funcs

func ParseArgs(args []string) {
	// WHAT? -> check if theres is an `arg[1..n]` and if that arg is not one of the registered
	// app commands, and if current session isn't nil either, then execute given command on ssh device
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
					cmd := strings.Join(args[1:], " ")
					out := ExecuteCommand(cmd, current_sess)
					fmt.Print(out.String())
				}
			} else {
				fmt.Println("Switch to a session first.")
			}
			return
		}
	}

	switch kingpin.MustParse(esh_cli.Parse(os.Args[1:])) {
		case add.FullCommand(): AddSession(*addname, *serverIP, *port, *user, *keyPath)
		case use.FullCommand(): UseSession(*usename)
		case listall.FullCommand(): ListSavedSessions()
		case logout.FullCommand(): LogoutCurrentSession()
		case remove.FullCommand(): RemoveSession(*removename)
		case get.FullCommand(): GetPath(*getpath)
		case put.FullCommand(): PutPath(*putpath)
	}
}

func main() {

	// support for -h flag
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
		panic("Error: " + err.Error())
	}
}
