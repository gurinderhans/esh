package main
import (
	"fmt"
	"path"
	"path/filepath"
	"syscall"
	"golang.org/x/crypto/ssh/terminal"
)

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


// TODO: add edit session config


