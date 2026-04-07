package main
import (
    "encoding/json"
    "fmt"
    "net"
    "os"
    "strings"
    "sync"
)
var accounts = map[string]string{
    "mahitha1": "test",
    "mahitha2": "test",
    "user": "test",
}
const BUFFERSIZE int = 1024
type Message struct {
    Type     string   json:"type"
    Username string   json:"username,omitempty"
    Password string   json:"password,omitempty"
    Message  string   json:"message,omitempty"
    Target   string   json:"target,omitempty"
    Sender   string   json:"sender,omitempty"
    Users    []string json:"users,omitempty"
}
type Client struct {
    conn     net.Conn
    username string
   loggedIn bool
}

var authClients = struct {
    sync.Mutex
    clients map[*Client]bool
}{clients: make(map[*Client]bool)}

var newclient = make(chan *Client)
var lostclient = make(chan *Client)

func main() {
    if len(os.Args) != 2 {
        fmt.Printf("Usage: %s <port>\n", os.Args[0])
        os.Exit(0)
    }
    port := os.Args[1]
    if len(port) > 5 {
        fmt.Println("Invalid port value. Try again!")
        os.Exit(1)
    }
    server, err := net.Listen("tcp", ":"+port)
    if err != nil {
        fmt.Printf("Cannot listen on port '%s'!\n", port)
        os.Exit(2)
    }
    fmt.Printf("ChatServer in GoLang developed by Phu Phung, revised by Mahitha Kalaga \n")
   fmt.Printf("ChatServer is listening on port '%s' ...\n", port)

    go func() {
        for {
            conn, err := server.Accept()
            if err != nil {
                fmt.Println("Error accepting connection:", err)
                continue
            }
            client := &Client{conn: conn, loggedIn: false}
            fmt.Printf("A new client is connected from %s. Waiting for login!\n", conn.RemoteAddr().String())
            newclient <- client
        }
    }()

    for {
        select {
        case client := <-newclient:
            go clientGoroutine(client)
        case client := <-lostclient:
            authClients.Lock()
            if _, exists := authClients.clients[client]; exists {
                delete(authClients.clients, client)
                if client.loggedIn {
                    broadcast(Message{Type: "NOTIFY", Message: fmt.Sprintf("New user '%s' logged out to Chat System from %s. Online users: %v", client.username, client.conn.RemoteAddr().String(), getUserList())}, true)
                }
            }
            authClients.Unlock()
            fmt.Printf("# of connected clients: %d\n", len(authClients.clients))
        }
    }
}

func clientGoroutine(client *Client) {
    defer func() {
        client.conn.Close()
        lostclient <- client
    }()

    authClients.Lock()
    authClients.clients[client] = true
    authClients.Unlock()
    fmt.Printf("# of connected clients: %d\n", len(authClients.clients))

    var buffer [BUFFERSIZE]byte
    for {
        byteReceived, readErr := client.conn.Read(buffer[0:])
        if readErr != nil {
            return
        }

        clientData := strings.TrimSpace(string(buffer[:byteReceived]))
        fmt.Printf("Received data: %s\n", clientData)
        fmt.Printf("DEBUG->data size = %d\n", len(clientData))

        var msg Message
        if err := json.Unmarshal([]byte(clientData), &msg); err != nil {
            fmt.Printf("DEBUG->strings.Compare: non-login data\n")
            sendToClient(client, Message{Type: "ERROR", Message: "non-login data. Please send login data first!"})
            continue
        }

        fmt.Printf("DEBUG->Got data: %v\n", msg)

        if !client.loggedIn {
            if msg.Type != "LOGIN" {
                fmt.Printf("DEBUG->strings.Compare: non-login data\n")
                sendToClient(client, Message{Type: "ERROR", Message: "non-login data. Please send login data first!"})
                continue
            }

            if valid, message := checkLogin(msg); valid {
                client.username = msg.Username
                client.loggedIn = true
                fmt.Printf("DEBUG->getUserlist()->%d, user.Username=%s\n", len(getUserList()), client.username)
                sendToClient(client, Message{Type: "AUTH_SUCCESS", Message: "Welcome to the chat system!"})
                broadcast(Message{Type: "NOTIFY", Message: fmt.Sprintf("New user '%s' logged in to Chat System from %s. Online users: %v", client.username, client.conn.RemoteAddr().String(), getUserList())}, true)
            } else {
                fmt.Printf("DEBUG->Invalid username or password\n")
                sendToClient(client, Message{Type: "AUTH_FAILED", Message: message})
            }
        } else {
            switch msg.Type {
            case "PUBLIC":
                fmt.Printf("DEBUG->ChatMessage->public: '%s'. Message: '%s'!\n", client.username, msg.Message)
                broadcast(Message{Type: "PUBLIC", Sender: client.username, Message: msg.Message}, false)
            case "PRIVATE":
                fmt.Printf("DEBUG->ChatMessage->private: '%s'. Message: '%s' to %s\n", client.username, msg.Message, msg.Target)
                sendPrivate(client, client.username, msg.Target, msg.Message)
            case "LOGOUT":
                return
            case "USERLIST":
                sendToClient(client, Message{Type: "USERLIST", Users: getUserList()})
            default:
                sendToClient(client, Message{Type: "ERROR", Message: "Unknown command"})
            }
        }
    }
}

func checkLogin(msg Message) (bool, string) {
    if storedPass, exists := accounts[msg.Username]; exists && storedPass == msg.Password {
        return true, "Authentication successful"
    }
    return false, "Invalid username or password"
}

func sendToClient(client *Client, msg Message) {
    data, err := json.Marshal(msg)
    if err != nil {
        fmt.Println("Error marshaling JSON:", err)
        return
    }
    data = append(data, '\n')
    _, writeErr := client.conn.Write(data)
    if writeErr != nil {
        fmt.Printf("Error sending to %s: %v\n", client.conn.RemoteAddr().String(), writeErr)
    }
}

func broadcast(msg Message, includeSender bool) {
    data, err := json.Marshal(msg)
    if err != nil {
        fmt.Println("Error marshaling JSON:", err)
        return
    }
    data = append(data, '\n')

    authClients.Lock()
    defer authClients.Unlock()

    for client := range authClients.clients {
        if !client.loggedIn {
            continue
        }
        if !includeSender && msg.Sender == client.username {
            continue
        }
        fmt.Printf("Data sent to %s: %s", client.conn.RemoteAddr().String(), string(data))
        _, writeErr := client.conn.Write(data)
        if writeErr != nil {
            fmt.Printf("Error broadcasting to %s: %v\n", client.conn.RemoteAddr().String(), writeErr)
            go func(c *Client) { lostclient <- c }(client)
        }
    }
}

func sendPrivate(senderClient *Client, senderUsername, targetUsername, message string) {
    authClients.Lock()
    defer authClients.Unlock()

    targetFound := false
    for client := range authClients.clients {
        if client.loggedIn && client.username == targetUsername {
            targetFound = true
            msg := Message{Type: "PRIVATE", Sender: senderUsername, Message: message}
            data, err := json.Marshal(msg)
            if err != nil {
                fmt.Println("Error marshaling JSON:", err)
                continue
            }
            data = append(data, '\n')
            fmt.Printf("Data sent to %s: %s", client.conn.RemoteAddr().String(), string(data))
            _, writeErr := client.conn.Write(data)
            if writeErr != nil {
                fmt.Printf("Error sending private to %s: %v\n", client.conn.RemoteAddr().String(), writeErr)
            }
        }
    }
    if !targetFound {
        sendToClient(senderClient, Message{Type: "ERROR", Message: fmt.Sprintf("User %s not found", targetUsername)})
    }
}

func getUserList() []string {
    authClients.Lock()
    defer authClients.Unlock()
    var usernames []string
    for client := range authClients.clients {
        if client.loggedIn {
            usernames = append(usernames, client.username)
        }
    }
    return usernames
}
