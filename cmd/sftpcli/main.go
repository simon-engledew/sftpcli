package main // import "github.com/simon-engledew/sftpcli/cmd/sftpcli"

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/pkg/sftp"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	usernameFlag = kingpin.Flag("username", "SFTP username").Required().Envar("SFTP_USERNAME").String()
	passwordFlag = kingpin.Flag("password", "SFTP password").Required().Envar("SFTP_PASSWORD").String()
	sizeFlag     = kingpin.Flag("size", "Max packet size").Default(strconv.Itoa(1 << 15)).Int()
	hostFlag     = kingpin.Flag("host", "Host").Default("localhost").String()
	portFlag     = kingpin.Flag("port", "Port").Default(strconv.Itoa(22)).Int()

	cpCommand = kingpin.Command("cp", "copy a file")
	cpSrcArg  = cpCommand.Arg("SRC", "Source").Required().String()
	cpDstArg  = cpCommand.Arg("DST", "Destination").Required().String()
)

func init() {
	kingpin.Version("0.0.0")
	kingpin.Parse()
}

func cpFile(client *sftp.Client, srcInfo os.FileInfo, src, dst string, baseDir string) (int64, error) {
	var directory, filename string

	if baseDir != "" {
		rel := strings.TrimPrefix(src, baseDir)

		directory = filepath.Dir(filepath.Join(dst, rel))

		filename = filepath.Base(src)
	} else {
		directory, filename = filepath.Split(dst)
		if filename == "" {
			filename = filepath.Base(src)
		}
	}

	if _, err := client.Stat(directory); os.IsNotExist(err) {
		err := client.MkdirAll(directory)
		if err != nil {
			return 0, err
		}
	}

	dst = filepath.Join(directory, filename)

	if destInfo, err := client.Stat(dst); err == nil {
		if srcInfo.ModTime().Before(destInfo.ModTime()) && srcInfo.Size() == destInfo.Size() {
			log.Printf("[skipped, mtime+size] %s -> %s", src, dst)
			return 0, nil
		}
	}

	log.Printf("%s -> %s", src, dst)

	srcFd, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer srcFd.Close()
	dstFd, err := client.Create(dst)
	if err != nil {
		return 0, err
	}
	defer dstFd.Close()
	return io.Copy(dstFd, srcFd)
}

func cp(client *sftp.Client, src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !srcInfo.IsDir() {
		_, err = cpFile(client, srcInfo, src, dst, "")
		return err
	}

	baseDir := src
	if !strings.HasSuffix(baseDir, string(os.PathSeparator)) {
		baseDir = filepath.Dir(baseDir)
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			_, err = cpFile(client, info, path, dst, baseDir)
		}
		return err
	})
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var auths []ssh.AuthMethod
	if authConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(authConn).Signers))
	}
	if passwordFlag != nil {
		auths = append(auths, ssh.Password(*passwordFlag))
	}

	config := ssh.ClientConfig{
		User:            *usernameFlag,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	addr := fmt.Sprintf("%s:%d", *hostFlag, *portFlag)
	conn, err := ssh.Dial("tcp", addr, &config)
	if err != nil {
		log.Fatalf("unable to connect to [%s]: %v", addr, err)
	}
	defer conn.Close()

	client, err := sftp.NewClient(conn, sftp.MaxPacket(*sizeFlag))
	if err != nil {
		log.Fatalf("unable to start sftp subsytem: %v", err)
	}
	defer client.Close()

	err = cp(client, *cpSrcArg, *cpDstArg)
	if err != nil {
		log.Fatal(err)
	}
}
