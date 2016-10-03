package main

import (
	"os"
	"io"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"golang.org/x/crypto/ssh"
	"gopkg.in/cheggaaa/pb.v1"
)

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

	fmt.Println("remote path: " + remote_file_path)

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
	var putFiles []string
	filepath.Walk(putpath, func (fpath string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			absPath, err := filepath.Abs(fpath) // absolute paths are easier to deal with in `PutFile` func when joining paths with remote
			if err == nil {
				putFiles = append(putFiles, absPath)
				bar := pb.New(int(info.Size())).SetUnits(pb.U_BYTES).Prefix(path.Base(fpath))
				progressBars = append(progressBars, bar) // store a pointer to the progress bar which is one per file
			}
		}
		return nil
	})

	// batch uploads of `n`, since opening too many SSH sessions at same time, causes connection disruption
	for i := 0; i < len(putFiles); i = i + 3 {
		max_index := i+3
		if max_index > len(putFiles) {
			max_index = len(putFiles)
		}

		BatchUpload(client, putFiles[i:max_index], progressBars[i:max_index])
	}
}

