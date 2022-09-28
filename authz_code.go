package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/google/uuid"
)

type tokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	Scope       string `json:"scope"`
	IdToken     string `json:"id_token"`
}

var (
	uuidState    = ""
	server       = ""
	clientId     = ""
	clientSecret = ""
	redirectUri  = ""

	user = ""
	pwd  = ""

	authzErr = ""
	code     = ""
	state    = ""

	cancel context.CancelFunc
	ctx    context.Context
)

/***
 *
 * http://localhost:8080/authorization-code/callback
 * 	?code=x_OU7lcUEqjxtcDmrDUigG5s-d7HYNoKmeMHHhJSyW8
 *	&state=E21FC8D0-080D-4007-8799-7FDBCC27AC1B
 */
func handler(w http.ResponseWriter, r *http.Request) {
	ru, _ := url.Parse(redirectUri)
	if ru.Path != r.URL.Path {
		fmt.Fprintf(os.Stderr, "Wrong URL exiting...")
		os.Exit(99)
	}
	fmt.Fprintf(w, "Hi there, I love %s! URI [%s]\n", r.URL.Path[1:], r.URL.Path)

	code = r.URL.Query().Get("code")
	fmt.Printf("GOT code=[%s]\n", code)
	authzErr = r.URL.Query().Get("error")
	fmt.Printf("GOT error=[%s]\n", authzErr)
	state = r.URL.Query().Get("state")
	fmt.Printf("GOT state=[%s]\n", state)

	validate()

	if obtainToken() {
		w.Write([]byte("OK"))
		fmt.Println("Before Calling Cancel...")
		cancel()
	}
}

// Kills server if something is not good enough
func validate() {
	if authzErr != "" {
		fmt.Fprintf(os.Stderr, "Authorization Error %s \n", authzErr)
		os.Exit(1)
	}
	if state != uuidState {
		fmt.Fprintf(os.Stderr, "Wrong state! Expected %s; Got: %s\n", uuidState, state)
		os.Exit(2)
	}
}

func obtainToken() (gotToken bool) {
	gotToken = false

	uri := "https://" + server + "/oauth2/default/v1/token"

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	fmt.Printf("REDIRECT URI %s\n", redirectUri)
	data.Set("redirect_uri", redirectUri)
	data.Set("client_id", clientId)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)

	fmt.Printf("Submitting data: %+v\nEncoded: %+v\n", data, data.Encode())

	r, err := http.NewRequest("POST", uri, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "POST request for token failed %s\n", err.Error())
		os.Exit(100)
	}

	//set headers to the request
	// r.SetBasicAuth(clientId, clientSecret)
	r.Header.Add("Accept", "application/json")
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded") //this is a must for form data encoded request

	fmt.Fprintf(os.Stdout, "Request: %+v\n", r)

	//send request and get the response
	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed while obtaining token %s\n", err.Error())
		os.Exit(200)
	}
	fmt.Println(res.Status)

	if res.StatusCode == 200 {
		defer res.Body.Close()
		jsonRes := tokenResp{}
		json.NewDecoder(res.Body).Decode(&jsonRes)
		fmt.Fprintf(os.Stdout, "\n\nSUCCESS: Got token: %+v\n\n", jsonRes)
		gotToken = true
	}

	return gotToken
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Can't open browser %s\n", err.Error())
	}
}

func authz() {
	uri := "https://" + server + "/oauth2/default/v1/authorize" +
		"?client_id=" + clientId +
		"&response_type=code" +
		"&scope=photo" +
		"&redirect_uri=" + redirectUri +
		"&state=" + uuidState

	redirectUri, _ = url.QueryUnescape(redirectUri)

	openbrowser(uri)
}

func main() {
	// $TARGET_HOST $CLIENT_ID $CLIENT_SECRET $REDIRECT_URI $USER $PWD
	server = url.QueryEscape(os.Args[1])
	clientId = url.QueryEscape(os.Args[2])
	clientSecret = url.QueryEscape(os.Args[3])
	redirectUri = url.QueryEscape(os.Args[4])
	user = os.Args[5]
	pwd = os.Args[6]

	uuidState = uuid.New().String()

	authz()

	http.HandleFunc("/", handler)

	ctx, cancel = context.WithCancel(context.Background())

	srv := &http.Server{Addr: ":8080"}
	go func() {
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "ERROR: Server Close error %s\n", err.Error())
		}
	}()

	<-ctx.Done()

	// Gracefully shutdown our server
	err := srv.Shutdown(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Server Shutdown failed %s\n", err.Error())
	}

	fmt.Println("--- Done. ---")
	// log.Fatal(http.ListenAndServe(":8080", nil))
}
