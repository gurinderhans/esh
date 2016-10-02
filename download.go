package main


import (
	"os"
	"io"
	"path"
	"path/filepath"
	"strings"
	"github.com/pkg/sftp"
	"gopkg.in/cheggaaa/pb.v1"
)

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

