package fb

import (
	"bytes"
	"compress/gzip"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"main.go/minecraft"
	"strings"
	"time"

	websocket "github.com/gorilla/websocket"
)

const authServer = "wss://api.fastbuilder.pro:2053/"

type AuthRequest struct {
	Action         string `json:"action"`
	ServerCode     string `json:"serverCode"`
	ServerPassword string `json:"serverPassword"`
	Key            string `json:"publicKey"`
	FBToken        string
	FBVersion      string
}

type Client struct {
	privateKey   *ecdsa.PrivateKey
	rsaPublicKey *rsa.PublicKey

	salt   []byte
	client *websocket.Conn

	encryptor      *encryptionSession
	serverResponse chan map[string]interface{}

	closed bool
}

func (client *Client) Close() {
	client.closed = true
	client.client.Close()
}

func handleMsg(authClient *Client, encryptedInfoLock chan struct{}) {
	defer func() {
		authClient.closed = true
	}()
	for {
		_, msg, err := authClient.client.ReadMessage()
		if err != nil {
			break
		}
		var message map[string]interface{}
		var outbuf bytes.Buffer
		var inbuf bytes.Buffer
		inbuf.Write(msg)
		reader, _ := gzip.NewReader(&inbuf)
		reader.Close()
		io.Copy(&outbuf, reader)
		msg = outbuf.Bytes()
		if authClient.encryptor != nil {
			authClient.encryptor.decrypt(msg)
		}
		json.Unmarshal(msg, &message)

		// action
		msgaction, _ := message["action"].(string)
		if msgaction == "encryption" {
			// set up encryptor
			spub := new(ecdsa.PublicKey)
			keyb64, _ := message["publicKey"].(string)
			keydata, _ := base64.StdEncoding.DecodeString(keyb64)
			spp, _ := x509.ParsePKIXPublicKey(keydata)
			ek, _ := spp.(*ecdsa.PublicKey)
			*spub = *ek
			authClient.encryptor = &encryptionSession{
				serverPrivateKey: authClient.privateKey,
				clientPublicKey:  spub,
				salt:             authClient.salt,
			}
			authClient.encryptor.init()
			close(encryptedInfoLock)
			continue
		} else if msgaction == "world_chat" {
			continue
		} else {
			select {
			case authClient.serverResponse <- message:
				break
			default:
				continue
			}
		}

	}
}

func CreateClient() (*Client, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		fmt.Printf("FBClient: Fail to generate private Key when connect to fb server(%v)\n", err)
		return nil, err
	}
	salt := []byte("bushe nmsl wrnmb")
	authClient := &Client{
		privateKey:     privateKey,
		salt:           salt,
		serverResponse: make(chan map[string]interface{}),
		closed:         false,
	}
	wsClient, _, err := websocket.DefaultDialer.Dial(authServer, nil)
	if err != nil {
		fmt.Printf("FBClient: Fail to create websocket when connect to fb server(%v)\n", err)
		return nil, err
	}
	authClient.client = wsClient
	encryptedInfoLock := make(chan struct{})
	go handleMsg(authClient, encryptedInfoLock)

	// generate and send public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		fmt.Printf("FBClient: Fail to generate public Key when connect to fb server(%v)\n", err)
		return nil, err
	}
	publicKeyStr := base64.StdEncoding.EncodeToString(publicKeyBytes)
	var inbuf bytes.Buffer
	writer := gzip.NewWriter(&inbuf)
	writer.Write([]byte(`{"action":"enable_encryption","publicKey":"` + string(publicKeyStr) + `"}`))
	writer.Close()
	publicKeyMsg := inbuf.Bytes()
	err = wsClient.WriteMessage(websocket.BinaryMessage, publicKeyMsg)
	if err != nil {
		return nil, err
	}
	// wait until encryption setted
	for {
		select {
		case <-encryptedInfoLock:
			return authClient, nil
		}
	}
}

func (client *Client) canSendMessage() bool {
	return client.encryptor != nil && !client.closed
}

func (client *Client) sendMessage(data []byte) error {
	if client.encryptor == nil {
		return fmt.Errorf("FBClient: Encryptor NOT Init")
	}
	if client.closed {
		fmt.Println("FBClient: Error: SendMessage: Connection Closed")
		return fmt.Errorf("FBClient: Error: SendMessage: Connection Closed")
	}
	client.encryptor.encrypt(data)
	var inbuf bytes.Buffer
	wr := gzip.NewWriter(&inbuf)
	wr.Write(data)
	wr.Close()
	err := client.client.WriteMessage(websocket.BinaryMessage, inbuf.Bytes())
	if err != nil {
		fmt.Printf("FBClient: Fail to Write Message to Server (%v)\n", err)
		return err
	}
	return nil
}

type FEncRequest struct {
	Action  string `json:"action"`
	Content string `json:"content"`
	Uid     string `json:"uid"`
}

func (client *Client) TransferData(content string, uid string) (string, error) {
	rspreq := &FEncRequest{
		Action:  "phoenix::transfer-data",
		Content: content,
		Uid:     uid,
	}
	msg, err := json.Marshal(rspreq)
	if err != nil {
		fmt.Printf("FBClient: An error occour when fb client Transfer Data (Failed to encode json) (%v)\n", err)
		return "", fmt.Errorf("FBClient: Failed to encode json (%v)", err)
	}
	err = client.sendMessage(msg)
	if err != nil {
		fmt.Printf("FBClient: An error occour when fb client Transfer Data (Failed to send data) (%v)\n", err)
		return "", fmt.Errorf("FBClient: Failed to send data")
	}
	resp, err := client.getResponse(5 * time.Second)
	if err != nil {
		fmt.Printf("FBClient: An error occour when fb client Transfer Data (Failed to get response) (%v)\n", err)
		return "", fmt.Errorf("FBClient: Failed to get response")
	}
	code, _ := resp["code"].(float64)
	if code != 0 {
		fmt.Printf("FBClient: An error occour when fb client Transfer Data (Failed to transfer start type) (%v)\n", err)
		return "", fmt.Errorf("FBClient: Failed to transfer start type")
	}
	data, _ := resp["data"].(string)
	return data, nil
}

func (client *Client) getResponse(duration time.Duration) (map[string]interface{}, error) {
	select {
	case resp, _ := <-client.serverResponse:
		return resp, nil
	case <-time.After(duration):
		fmt.Printf("FBClient: Wait for fb server response timeout\n")
		return nil, fmt.Errorf("FBClient: Wait for fb server response timeout")
	}

}

type FTokenRequest struct {
	Action   string `json:"action"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type FBPlainToken struct {
	EncryptToken bool   `json:"encrypt_token"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

func (client *Client) GetToken(username string, password string) (string, error) {
	tokenstruct := &FBPlainToken{
		EncryptToken: true,
		Username:     username,
		Password:     password,
	}
	tmpToken, err := json.Marshal(tokenstruct)
	if err != nil {
		fmt.Printf("FB Client: Failed to generate temp token (%v)\n", err)
		return "", fmt.Errorf(fmt.Sprintf("Failed to generate temp token %v", err))
	}
	rspreq := &FTokenRequest{
		Action:   "phoenix::get-token",
		Username: "",
		Password: string(tmpToken),
	}
	msg, err := json.Marshal(rspreq)
	if err != nil {
		fmt.Printf("FB Client: Failed to encode json (%v)\n", err)
		return "", fmt.Errorf("FB Client: Failed to encode json (%v)", err)
	}
	err = client.sendMessage(msg)
	if err != nil {
		fmt.Printf("FB Client: Failed to SendMessage (%v)\n", err)
		return "", fmt.Errorf("FB Client: Failed to SendMessage (%v)", err)
	}
	resp, err := client.getResponse(time.Second * 5)
	if err != nil {
		fmt.Printf("FB Client: Failed to Get Response (%v)\n", err)
		return "", fmt.Errorf("FB Client: Failed to Get Response  (%v)", err)
	}
	code, _ := resp["code"].(float64)
	if code != 0 {
		fmt.Println("FB Client: FB server return empty token, login fail")
		return "", fmt.Errorf("FB Client: FB server return empty token, login fail, Maybe Password is Incorrect")
	}
	userToken, _ := resp["token"].(string)
	return userToken, nil
}

type Identify struct {
	FBToken        string
	FBVersion      string
	ServerCode     string
	ServerPassword string
}

func (client *Client) authNetEase(identify *Identify) (*minecraft.LoginToken, error) {
	key, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	data, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	pubKeyData := base64.StdEncoding.EncodeToString(data)
	authreq := &AuthRequest{
		Action:         "phoenix::login",
		ServerCode:     identify.ServerCode,
		ServerPassword: identify.ServerPassword,
		Key:            pubKeyData,
		FBToken:        identify.FBToken,
		FBVersion:      identify.FBVersion,
	}
	msg, err := json.Marshal(authreq)
	if err != nil {
		fmt.Printf("FB Client: Failed to encode json (%v)\n", err)
		return nil, fmt.Errorf("FB Client: Failed to encode json (%v)", err)
	}
	err = client.sendMessage(msg)
	if err != nil {
		fmt.Printf("FB Client: Failed to SendMessage (%v)\n", err)
		return nil, fmt.Errorf("FB Client: Failed to SendMessage (%v)", err)
	}
	resp, err := client.getResponse(time.Second * 5)
	if err != nil {
		fmt.Printf("FB Client: Failed to GetResponse (%v)\n", err)
		return nil, fmt.Errorf("FB Client: Failed to GetResponse (%v)", err)
	}
	code, _ := resp["code"].(float64)
	if code != 0 {
		err, _ := resp["message"].(string)
		trans, hasTrans := resp["translation"].(float64)
		if hasTrans {
			fmt.Printf("FB Client: Error code=%v msg=%v trans=%v", code, err, trans)
			return nil, fmt.Errorf(fmt.Sprintf("FB Client: error code=%v msg=%v trans=%v", code, err, trans))
		} else {
			fmt.Printf("FB Client: Error code=%v msg=%v", code, err)
			return nil, fmt.Errorf(fmt.Sprintf("FB Client: error code=%v msg=%v", code, err))
		}
	}
	chainAddr, _ := resp["chainInfo"].(string)
	chainAndAddr := strings.Split(chainAddr, "|")
	chainData := chainAndAddr[0]
	address := chainAndAddr[1]
	return &minecraft.LoginToken{
		PrivateKey: key,
		UserToken:  chainData,
		Address:    address,
		NetWork:    "raknet",
	}, nil
}
