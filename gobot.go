package main

import (
    "bytes"
    "fmt"
    "log"
    "strings"
    "strconv"
    "time"
    "text/template"

    irc "github.com/fluffle/goirc/client"
    redis "github.com/alphazero/Go-Redis"
)

const (
    botname = "GoBot"

    // irc commands
    join = "JOIN"
    part = "PART"
    pmsg = "PRIVMSG"
    quit = "QUIT"

    // help message
    helpmsg = " command [arg1] [arg2]\n--------------------\nHELP - Display this help message\nHISTORY - Show users last 20 messages\nLASTSAW - Show users last seen timestamp\n"
)

// the bots commands
var botCommand = map[string]map[int] func([]string, *irc.Line) string{
    "HELP": map[int] func([]string, *irc.Line) string {
       -1: func (args []string, line *irc.Line) string { return helpmsg },
       0: func (args []string, line *irc.Line) string { return helpmsg },
    },
}


/* github post-receive stuff, trying to keep it close together
 until I learn about module and restructure it out of this file */
const (
    githubTemplates = `
        {{ define "git-url" }}http://www.github.com{{ end }}
        {{ define "git-repo" }}{{ template "git-url" }}/{{ .user }}/{{ .name }}{{ end }}
        {{ define "git-compare" }}{{ template "git-repo" }}/{{ .branch }}..{{ .commit }}{{ end }}`
)

// templates
var tmplGit = template.Must(template.New("git").Parse(githubTemplates))

type GitRepo struct {
    name string // The github repo name 
    user string // The github user
    branch string // The branch to diff against
}

func (repo GitRepo) String() string {
    buff  := bytes.NewBufferString("")

    if err := tmplGit.ExecuteTemplate(buff, "git-url", repo); err != nil {
        log.Print("Error executing template")
        return "" // ERROR ! XXX: implement error handling fool
    }
    
    return fmt.Sprintf("%p", buff)
}

func (repo GitRepo) diff(commit string) string {
    return ""
}

/* End Git Module */

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
    redis redis.Client // The redis client connection
    quit chan bool // A channel to signal close
}

func (ch *IrcChannelLogger) command (command string, args []string, line *irc.Line) string {
    // Prob not a bad idea to do some upper bounds checking to avoid overflow?
    cmdMap, cmdok := botCommand[command]
    _, argok := cmdMap[len(args)]

    log.Printf("Command(%s) ok? %v ", command, cmdok)
    log.Printf("Arguments(%s) length(%d) ok? %v ", args, len(args), argok)

    // Check to see if command with args exists
    if cmdok != true { 
       return fmt.Sprintf(
           "Uknown command %s \n %s", command, botCommand["HELP"][-1]([]string{}, line)) 
    } else if argok != true {
        return fmt.Sprintf(
            "Wrong Number of arguments. \n %s", botCommand[command][-1]([]string{}, line))
    }

    return botCommand[command][len(args)](args, line) 
}

func (ch IrcChannelLogger) timestamp() int64 { 
    return int64(time.Now().Unix() - ch.time)
}

func (ch IrcChannelLogger) timestampstr() string { 
    return strconv.FormatInt(ch.timestamp(), 10) 
}

func (ch *IrcChannelLogger) start ()  {
    ch.client = irc.SimpleClient(botname)
    ch.client.SSL = false

    log.Print("Starting " + ch.name + " channel logger")

    ch.client.AddHandler(irc.CONNECTED, ch.connectIRC)
    ch.client.AddHandler(join, ch.joinChan)
    ch.client.AddHandler(pmsg, ch.privMsg)
    ch.client.AddHandler(part, ch.partChan)
    ch.client.AddHandler(quit, ch.quitChan)

    if err := ch.client.Connect(ch.host); err != nil {
        log.Println("Failed to connect to the IRC Server: " + ch.host + " the error was: ", err)
        return
    }

    cli, err := redis.NewSynchClient()

    if err != nil { log.Println("Failed to connect to redis, the error was ", err); return }

    ch.redis = cli

    <- ch.quit
}

// Define IRC handlers

func (ch *IrcChannelLogger) quitChan (conn *irc.Conn, line *irc.Line) {
    log.Print(line.Nick + " has quit")
}

func (ch *IrcChannelLogger) partChan (conn *irc.Conn, line *irc.Line) {
    // record the channel users last seen time
    log.Printf(
        "user(%s) has left the %s channel, last seen timestamp is now %d",
        line.Nick, ch.name, ch.timestampstr())

}

func (ch *IrcChannelLogger) joinChan (conn *irc.Conn, line *irc.Line)  {

    var chKey = ch.name + ":starttime"

    log.Printf(line.Nick + " has joined the %s channel", ch.name)

    channelStart, _ := ch.redis.Get(chKey)

    log.Print(channelStart)

    ch.time = time.Now().Unix()
    ch.redis.Set(chKey, []byte(strconv.FormatInt(ch.time, 10)))
}

func (ch *IrcChannelLogger) connectIRC (conn *irc.Conn, line *irc.Line)  {
    log.Print("Connected to IRC Server: " + ch.host)
    ch.client.Join(ch.name)
}

func (ch *IrcChannelLogger) privMsg (conn *irc.Conn, line *irc.Line) {
    parts := strings.Fields(line.Args[1])
    if len(parts) < 1 { return }

    log.Print("Inside privmsg handler", parts)
    target := strings.ToLower(parts[0])

    args := []string{}

    // if the botname wasnt the first word received
    if target != strings.ToLower(ch.nick) { 
        log.Print("Message received: " + line.Args[1])

        // log the message with timestamp to redis 
        ch.redis.Zadd(ch.name, float64(
            ch.timestamp()), []byte(line.Nick + ":" + line.Args[1]))
        return 
    }

    // if its just the logbots name, do nothing. return help?
    if len(parts) <= 1 {
        ch.client.Privmsg(ch.name, ch.command("HELP", []string{}, line)) 
        return  
    } else if len(parts) == 2 {
        args = []string{}
    } else { args = parts[2:] }

    command := strings.ToUpper(parts[1])

    log.Printf("Command received: %s and arguments(%d): %s", command, len(args), args)

    ch.client.Privmsg(ch.name, ch.command(command, args, line))
}

func main() {

    // Join the command/control server
    cc := IrcChannelLogger{
            name: "#" + botname, 
            host: "127.0.0.1", 
            port: 6667,
            nick: botname,
            ssl: false,
            listen: true,
        }

    cc.start()
}
