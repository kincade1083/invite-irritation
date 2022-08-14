package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

//Endpoints
const (
	BaseEndpoint          = "https://api.vrchat.cloud/api/1"
	ApiEndpoint           = BaseEndpoint + "/config"
	LoginEndpoint         = BaseEndpoint + "/auth/user"
	RequestInviteEndpoint = BaseEndpoint + "/requestInvite/"
	LogoutEndpoint        = BaseEndpoint + "/logout"
)

type user struct {
	apiKey    string
	userName  string
	password  string
	authToken string
}

func (u *user) authenticateUser() error {
	client := &http.Client{}
	request, err := http.NewRequest("GET", LoginEndpoint, nil)
	request.SetBasicAuth(u.userName, u.password)

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	auth := parseCookieValue("auth", response)
	if auth == "" {
		return fmt.Errorf("Unable to obtain authentication key. Check provided credentials.")
	}
	u.authToken = auth

	return nil
}

func (u *user) logOut() error {
	client := &http.Client{}
	request, err := http.NewRequest("PUT", LogoutEndpoint, nil)
	request.Header.Set("Cookie", "auth="+u.authToken)

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	if response.StatusCode != 200 {
		return fmt.Errorf("Unable to log out.")
	}

	return nil
}

func main() {
	username := os.Args[1]
	password := os.Args[2]
	target := os.Args[3]
	requestCount, err := strconv.Atoi(os.Args[4])
	if err != nil {
		panic(err)
	}

	var apiKey string

	//Fetch API key.
	apiKey, err = fetchApiKey()
	if err != nil {
		panic(err)
	}

	user := user{
		apiKey:   apiKey,
		userName: username,
		password: password,
	}

	//Log in.
	err = user.authenticateUser()
	if err != nil {
		panic(err)
	}
	//Put your friend on blast.
	err = sendRequests(user, target, requestCount)
	if err != nil {
		panic(err)
	}

	//We've delivered our payload, let's get out of here.
	err = user.logOut()
	if err != nil {
		panic(err)
	}

	fmt.Println("Finished sending requests")
}

func fetchApiKey() (string, error) {
	var apiKey string
	response, err := http.Get(ApiEndpoint)
	if err != nil {
		return "", err
	}

	apiKey = parseCookieValue("apiKey", response)
	if apiKey == "" {
		return "", fmt.Errorf("Unable to parse API Key.")
	}

	return apiKey, nil
}

func sendRequests(user user, target string, requestCount int) error {
	client := &http.Client{}

	json := []byte(`{"messageSlot": 0}`)

	request, _ := http.NewRequest("POST", RequestInviteEndpoint+target, bytes.NewBuffer(json))
	request.Header.Set("Cookie", "apiKey="+user.apiKey+"; auth="+user.authToken)
	request.Header.Set("Content-Type", "application/json")

	ticker := time.NewTicker(30 * time.Second)

	requestsSent := 0
	err := func() error {
		for {
			select {
			case <-ticker.C:
				response, err := client.Do(request)
				if err != nil {
					return err
				}

				if response.StatusCode != 200 {
					return fmt.Errorf("Unable to send request.")
				}

				fmt.Println("Sent request.")

				requestsSent++
				if requestsSent >= requestCount {
					return nil
				}
			}
		}
	}()

	if err != nil {
		return err
	}

	return nil
}

func parseCookieValue(cookieName string, response *http.Response) string {
	var value string
	for _, cookie := range response.Cookies() {
		if cookie.Name == cookieName {
			value = cookie.Value
		}
	}
	return value
}
