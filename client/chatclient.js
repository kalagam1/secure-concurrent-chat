const net = require('net');
const readlineSync = require('readline-sync');
const client = new net.Socket();
const HOST = 'localhost';
const PORT = 8080;
let username = '';
let authenticated = false;
let buffer = '';
function sendMessage(message) {
    const data = JSON.stringify(message) + '\n';
    client.write(data);
}
function login() {
    const usernameInput = readlineSync.question('Username:');
   const passwordInput = readlineSync.question('Password:', { hideEchoBack: true, mask: '*' });
    username = usernameInput;
    const message = { type: 'LOGIN', username: usernameInput, password: passwordInput };
    sendMessage(message);
}

function handleChat() {
    console.log("Type '[To:Receiver] Message' to send to a specific user.");
    console.log("Type '.userlist' to request latest online users.");
    console.log("Type '.exit' to logout and close the connection");
    const input = readlineSync.question('', { keepWhitespace: true }).trim();

    if (input.startsWith('[To:')) {
        const match = input.match(/\[To:([^\]]+)\]\s*(.+)/);
        if (match) {
            const target = match[1];
            const message = match[2];
            sendMessage({ type: 'PRIVATE', target, message });
        } else {
            console.log('Invalid private message format. Use [To:Receiver] Message');
        }
    } else if (input === '.userlist') {
        sendMessage({ type: 'USERLIST' });
    } else if (input === '.exit') {
        sendMessage({ type: 'LOGOUT' });
        client.end();
        return false;
    } else if (input) {
        sendMessage({ type: 'PUBLIC', message: input });
    }
    return true;
}

client.connect(PORT, HOST, () => {
    console.log(Simple chatclient.js developed by Phu Phung, SecAD);
    console.log(Connected to: ${HOST}:${PORT});
    console.log('You need login before sending/receiving message.');
    login();
});

client.on('data', (data) => {
    buffer += data.toString();
    const messages = buffer.split('\n');
    buffer = messages.pop();

    for (const msg of messages) {
        if (msg.trim() === '') continue;
        try {
            const parsed = JSON.parse(msg);
            if (parsed.type === 'AUTH_SUCCESS') {
                console.log(You have logged in successfully with username ${username});
                console.log('WELCOME to the Chat System. Type anything to send to public chat.');
                authenticated = true;
            } else if (parsed.type === 'AUTH_FAILED') {
                console.log('AUTHENTICATION FAILED. Please try again');
                login();
            } else if (parsed.type === 'PUBLIC') {
                console.log(Received data:PUBLIC message from '${parsed.sender}': ${parsed.message});
            } else if (parsed.type === 'PRIVATE') {
                console.log(Received data:PRIVATE message from '${parsed.sender}': ${parsed.message});
            } else if (parsed.type === 'USERLIST') {
                console.log(Received data: Online users: [${parsed.users.join(', ')}]);
            } else if (parsed.type === 'NOTIFY') {
                console.log(Received data:${parsed.message});
            } else if (parsed.type === 'ERROR') {
                console.log(Received data:${parsed.message});
            }
        } catch (err) {
            console.log('Error parsing message:', msg, err);
        }
    }
});

client.on('close', () => {
    console.log('Connection closed');
});

client.on('error', (err) => {
    console.error('Connection error:', err.message);
});

function startChatLoop() {
    if (authenticated) {
        if (!handleChat()) {
            return;
        }
    }
    setImmediate(startChatLoop);
}

client.on('connect', () => {
    startChatLoop();
});
