package main // import "github.com/simon-engledew/sftpcli/cmd/sftpcli"

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/pkg/sftp"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	usernameFlag = kingpin.Flag("username", "SFTP username").Default(os.Getenv("SFTP_USERNAME")).String()
	passwordFlag = kingpin.Flag("password", "SFTP password").Default(os.Getenv("SFTP_PASSWORD")).String()
	sizeFlag     = kingpin.Flag("size", "Max packet size").Default(strconv.Itoa(1 << 15)).Int()
	hostFlag     = kingpin.Flag("host", "Host").Default("localhost").String()
	portFlag     = kingpin.Flag("port", "Port").Default(strconv.Itoa(22)).Int()
	cpCommand    = kingpin.Command("cp", "copy a file")
	srcArg       = cpCommand.Arg("SRC", "Source").Required().String()
	dstArg       = cpCommand.Arg("DST", "Destination").Required().String()
)

func init() {
	kingpin.Version("0.0.0")
	kingpin.Parse()
}

func cp(client *sftp.Client, src, dst string) (int64, error) {
	directory, filename := filepath.Split(dst)

	if filename == "" {
		filename = filepath.Base(src)
	}

	if _, err := client.Stat(directory); os.IsNotExist(err) {
		err := client.MkdirAll(directory)
		if err != nil {
			return 0, err
		}
	}

	dst = filepath.Join(directory, filename)

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

	_, err = cp(client, *srcArg, *dstArg)
	if err != nil {
		log.Fatal(err)
	}
}
