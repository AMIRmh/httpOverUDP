package myLib

import (
	"fmt"
	"os"
	"time"
	"math/rand"
	"strings"
	"unicode"
	"net/http"
	"io/ioutil"
	"github.com/go-errors/errors"
)

const (
	Npos = -1
)

var (
	letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

func CheckError(err error) {
	if err != nil {
		fmt.Println(err.(*errors.Error).ErrorStack())
		os.Exit(1)
	}
}


// reverse an array of type byte
func Reverse(s []byte) []byte {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}


// checks if the elem is is the array
// if yes returns the index of the element else returns myLib.Npos
func ContainsInt(arr []int, elem int) int {
	for i,e := range arr {
		if e == elem {
			return i
		}
	}
	return Npos
}

// remove an element by index from an int array
func RemoveInt(arr []int, index int) []int {
	return append(arr[:index], arr[index+1:]...)
}

func RandStringRunes(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func HTTPHeaderParser(httpHeader string) (map[string]string, error) {
	splited := strings.Split(httpHeader, "\n\r")
	result := make(map[string]string)



	getStr := strings.Split(splited[0], " ")
	if len(getStr) == 3 {
		result[getStr[0]] = getStr[1] + " " + getStr[2]
	} else if len(getStr) == 2 {
		result[getStr[0]] = getStr[1]
	} else {
		fmt.Println("im wrong 1")
		return nil, errors.New("wrong http format")
	}


	for i := 1; i < len(splited); i++ {
		str := splited[i]
		s := strings.Split(str, ":")
		if len(s) == 2 {
			val := strings.Map(func(r rune) rune {
				if unicode.IsSpace(r) {
					return -1
				}
				return r
			}, s[1])
			result[s[0]] = val
		} else if len(s) != 1 {
			return nil, errors.New("wrong http format")
		}
	}
	return result, nil
}

func SendHTTPRequest(request []byte) []byte {
	str := string(request)
	httpHeaders, err := HTTPHeaderParser(str)
	CheckError(err)

	resp, err := http.Get("http://" + httpHeaders["HOST"] + httpHeaders["GET"])
	CheckError(err)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	return body
}