package main

import (
    "log"
//    "strings"
    "strconv"
    "time"

    irc "github.com/fluffle/goirc/client"
    redis "github.com/alphazero/Go-Redis"
)

const (
    BOTNAME = "GoBot"

    // irc commands
    JOIN = "JOIN"
    PART = "PART"
    PMSG = "PRIVMSG"
    QUIT = "QUIT"
)

type IrcUser struct {
    nick string // The nickname of the IRC user
    last int64 // The unixtime the user was last seen relative to the Channel start
}

type IrcChannelLogger struct {
    name string // The name of the channel
    time int64 // The unix time that the logging started
    host string // The hostname of the IRC server
    port int16   // The port number of the IRC service
    nick string // The bots nickname to use
    ssl bool    // SSL?
    listen bool // Listen for commands 

    members []IrcUser // A list of IrcUsers in the channel
    client *irc.Conn // The IRC Client for the channel
    redis *redis.Client // The redis client connection
    quit chan bool // A channel to signal close
}

func (ch IrcChannelLogger) log (msg string) error {
    log.Println("Logging message ", msg)
    return nil
}

func (ch *IrcChannelLogger) start ()  {
    ch.client.SSL = false

    log.Print("Starting " + ch.name + " channel logger")

    ch.client.AddHandler(irc.CONNECTED, ch.Connect)
    ch.client.AddHandler(JOIN, ch.Join)
    ch.client.AddHandler(PMSG, ch.PrivMsg)
    ch.client.AddHandler(PART, ch.Part)
    ch.client.AddHandler(QUIT, ch.Quit)

    if err := ch.client.Connect(ch.host); err != nil {
        log.Println("Failed to connect to the IRC Server: " + ch.host + " the error was: ", err)
        return
    }

    cli, err := redis.NewSynchClient()

    if err != nil { log.Println("Failed to connect to redis, the error was ", err); return }

    ch.redis = &cli

    <- ch.quit
}

// Define IRC handlers

func (ch *IrcChannelLogger) Quit (conn *irc.Conn, line *irc.Line) {
    log.Print(line.Nick + " has quit")
}

func (ch *IrcChannelLogger) Part (conn *irc.Conn, line *irc.Line) {
    // record the channel users last seen time
    log.Print("user(" + line.Nick + ") has left the " + ch.name + " channel, last seen timestamp is " + strconv.FormatInt(time.Now().Unix() - ch.time, 10))
}

func (ch *IrcChannelLogger) Join (conn *irc.Conn, line *irc.Line)  {
    log.Printf(line.Nick + " has joined the %s channel", ch.name)
    ch.time = time.Now().Unix()
}

func (ch *IrcChannelLogger) Connect (conn *irc.Conn, line *irc.Line)  {
    log.Print("Connected to IRC Server: " + ch.host)
    conn.Join("#" + ch.name)
}

func (ch *IrcChannelLogger) PrivMsg (conn *irc.Conn, line *irc.Line) {
    log.Print("Message received: " + line.Args[1])
}

func main() {

    // Join the command/control server
    cc := IrcChannelLogger{
            name: BOTNAME, 
            host: "127.0.0.1", 
            port: 6667,
            nick: BOTNAME,
            ssl: false,
            listen: true,
            client: irc.SimpleClient(BOTNAME),
        }

    cc.start()
}
