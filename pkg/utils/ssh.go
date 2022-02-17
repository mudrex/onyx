package utils

import (
	"bytes"
	"io/ioutil"

	"github.com/mudrex/onyx/pkg/logger"
	"golang.org/x/crypto/ssh"
)

func getKeyFile(privateKey string) (key ssh.Signer, err error) {
	buf, err := ioutil.ReadFile(privateKey)
	if err != nil {
		return
	}

	key, err = ssh.ParsePrivateKey(buf)
	if err != nil {
		return
	}

	return
}

func CheckIfUserAbleToLogin(privateKey, host, user string) bool {
	key, err := getKeyFile(privateKey)
	if err != nil {
		logger.Error("Unable to parse private key %s", privateKey)
		return false
	}

	c := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", host+":22", c)
	if err != nil {
		logger.Error("Unable to connect to host %s@%s. Please check username or private key", user, host)
		return false
	}

	session, err := client.NewSession()
	if err != nil {
		logger.Error("Failed to create session: " + err.Error())
		return false
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run("/usr/bin/whoami"); err != nil {
		logger.Error("Failed to run: " + err.Error())
		return false
	}

	return true
}
