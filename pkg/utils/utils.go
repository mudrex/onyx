package utils

import (
	"bufio"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"regexp"
	"strings"

	"github.com/mudrex/onyx/pkg/logger"
)

var sources = []string{
	"https://checkip.amazonaws.com",
	"https://api.ipify.org?format=text",
	"https://api64.ipify.org/?format=text",
	"https://www.ipify.org",
	"https://myexternalip.com/raw",
}

func GetPublicIP() string {
	re, _ := regexp.Compile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\/\d{1,2}$`)

	for _, source := range sources {
		logger.Info("Getting IP address from %s", logger.Underline(source))
		resp, err := http.Get(source)
		if err != nil {
			logger.Error("Unable to get ip from %s. Error: %v", logger.Underline(source), err)
			continue
		}

		defer resp.Body.Close()
		ip, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Error("Unable to read ip from response %s. Error: %s", logger.Underline(source), err.Error())
			continue
		} else {
			cidr := fmt.Sprintf("%s/32", strings.TrimSpace(string(ip)))
			if re.MatchString(cidr) {
				logger.Success("Authorizing for CIDR: " + cidr)
				return cidr
			}
		}
	}

	logger.Fatal("Unable to determine ip")

	return ""
}

func GetChunks(arr []string, chunkSize int) [][]string {
	if len(arr) == 0 {
		return nil
	}
	chunks := make([][]string, (len(arr)+chunkSize-1)/chunkSize)
	prev := 0
	i := 0

	for prev < len(arr)-chunkSize {
		next := prev + chunkSize
		chunks[i] = arr[prev:next]
		prev = next
		i++
	}

	chunks[i] = arr[prev:]
	return chunks
}

func GetUserInput(message string) string {
	consoleReader := bufio.NewReader(os.Stdin)
	fmt.Print(message)
	input, _ := consoleReader.ReadString('\n')
	return input
}

func GetUser() string {
	currUser, err := user.Current()
	if err != nil {
		return ""
	}

	return currUser.Username
}

func GetStringAMinusB(a, b []string) []string {
	mapA := make(map[string]bool)
	for _, str := range a {
		mapA[str] = true
	}

	mapB := make(map[string]bool)
	for _, str := range b {
		mapB[str] = true
	}

	diff := make([]string, 0)
	for c := range mapA {
		if _, ok := mapB[c]; !ok {
			diff = append(diff, c)
		}
	}

	return diff
}

func AreStringArrayEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	if len(a) == 0 && len(b) == 0 {
		return true
	}

	mapA := make(map[string]bool)
	for _, str := range a {
		mapA[str] = true
	}

	mapB := make(map[string]bool)
	for _, str := range b {
		mapB[str] = true
	}

	for c := range mapA {
		delete(mapB, c)
	}

	return len(mapB) == 0
}

func GetSHA512Checksum(data []byte) string {
	hasher := sha512.New()
	hasher.Write(data)
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}
