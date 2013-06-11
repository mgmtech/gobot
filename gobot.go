package main

/* Primitive IRC Logging bot, written in Go

*/

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"text/template"
	"time"

	redis "github.com/alphazero/Go-Redis"
	irc "github.com/fluffle/goirc/client"
)

const (
	botname = "GoBot"

	// irc commands
	join = "JOIN"
	part = "PART"
	pmsg = "PRIVMSG"
	quit = "QUIT"
    names = "NAMES"

	// redis key suffixes
	sfxLast  = ":lastseen"
	sfxStart = ":starttime"

    msgDate = "1/2/06 15:04:05"
)

/* 
Bot commands and contextual help map

Commands are structured in a map[string][int] of functions which return a string.
-1 index is for the help message, the rest correspond to the number of arguments for the command.

*/
var botCommand = map[string]map[int]func(*IrcChannelLogger, []string, *irc.Line) string{
	"DIE": map[int]func(*IrcChannelLogger, []string, *irc.Line) string{
		-1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string { return "Whatchoo ¿awkin` bought wilis¿" },
		0: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {  <- ch.quit; return "l8tr" },
    },
    "HELP": map[int]func(*IrcChannelLogger, []string, *irc.Line) string{
		-1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return `
command [arg1] [arg2]
--------------------------------
HELP - Display this help message
HISTORY - Show users last 20 messages
LASTSAW - Show users last seen timestamp`
		},
		0: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return "HELP command - Show help for the command."
		},
		1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
           return "*special case* you shouldnt be here.." // The channel logger should handle this special case as self referenceing is not allowed
           //due to Initialization loop, break out into function or initialize a helpmap from -1 indexes. botCommand[args[0]][-1](ch, args, line)
        },
	},
	"HISTORY": map[int]func(*IrcChannelLogger, []string, *irc.Line) string {
		-1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return "HISTORY [n] - Display the last n messages, ommitting n defaults to 20."
		},
		0: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			// retrieve missed messages here and remit as multi-line string.
            rVal := ch.messages(
               int(ch.lastseen(line.Nick)), //start score
               int(ch.timestamp()))        // end score

            log.Printf("%v", rVal)

            var msg string
            for _, v := range rVal { msg += string(v) }
            
            return msg
		},
	},
	"LASTSAW": map[int]func(*IrcChannelLogger, []string, *irc.Line) string{
		-1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return "LASTSAW [user] - Display the user lastseen timestamp, nick of user sending by default."
		},
		0: func(ch *IrcChannelLogger, args []string, line *irc.Line) string { return fmt.Sprintf("%s, last saw you %s UNIX timestamp", line.Nick, ch.lastseenstr(line.Nick)) },
		1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string { return fmt.Sprintf("%s was last seen %s UNIX timestamp", args[0], ch.lastseenstr(args[0])) },
	},
	"TIMESTAMP": map[int]func(*IrcChannelLogger, []string, *irc.Line) string{
		-1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string { 
            return "TIMESTAMP - Returns the channels current timestamp, logging start and uptime in seconds"
        },
		0: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
            return fmt.Sprintf("Current UNIX timestamp %d, start %d and uptime %d (%d min)",
                time.Now().Unix(), ch.time, ch.timestamp(), ch.timestamp()/60)
        },
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
	name   string // The github repo name
	user   string // The github user
	branch string // The branch to diff against
}

func (repo GitRepo) String() string {
	buff := bytes.NewBufferString("")

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

type IrcChannelLogger struct {
	name        string // The name of the channel
	time        int64  // The unix time that the logging started
	host        string // The hostname of the IRC server
	port        int16  // The port number of the IRC service
	nick        string // The bots nickname to use
	ssl         bool   // SSL?
	listen      bool   // Listen for commands
    initialized bool // Channel initialized?

	client      *irc.Conn    // The IRC Client for the channel
	redis       redis.Client // The redis client connection
	quit        chan bool    // A channel to signal close
}


func (ch *IrcChannelLogger) rkey() string { return fmt.Sprintf("%s:%d:%s:", ch.host, ch.port, ch.name) }
func (ch *IrcChannelLogger) ukey(user string) string { return fmt.Sprintf("%s%s", ch.rkey(), user) }

func (ch *IrcChannelLogger) user_left(user string) {
    ch.redis.Set(ch.ukey(user) + sfxLast, []byte(ch.timestampstr()))
}

func (ch *IrcChannelLogger) lastseen(user string) int64 {
	v, _ := ch.redis.Get(ch.ukey(user) + sfxLast)
	i, _ := strconv.ParseInt(string(v), 10, 64)
    log.Printf("Last seen for user %s was %d", user, i)
	return i
}

func (ch *IrcChannelLogger) missed(user string) int32 {
    value, err := ch.redis.Zrangebyscore(ch.rkey(), float64(ch.lastseen(user)), float64(ch.timestamp()))
    if err != nil {
        log.Printf("Error was %v, value is %v", err, value)
    } else {
        return 0
    }
    return -1
}

func (ch *IrcChannelLogger) messages (start, end int) string {
    rVal, err := ch.redis.Zrangebyscore(ch.rkey(), float64(start), float64(end))
    
    log.Printf("%v Message retrived", len(rVal))
    if err != nil {
        return fmt.Sprintf("Problem geting messages: %s %s (err = %v, %v)", start, end, err, rVal)
    } else {
        return string (bytes.Join(rVal, []byte{'\n'}))
    }
}

func (ch *IrcChannelLogger) lastseenstr(user string) string {
    return strconv.FormatInt(ch.lastseen(user), 10)
}

// TODO: refactor this duplicate code, allow it to respond to a different channel/person 
func (ch *IrcChannelLogger) multilineMsg(msg, dest string){
	// parse a multiline message and remit to channel
	for i, v := range strings.Split(msg, "\n") {
		if i == 0 && v == "" { // Skip leading blank lines
        continue
		}
		ch.client.Privmsg(dest, v)
	}
}

func (ch *IrcChannelLogger) noticemultilineMsg(msg, dest string) {
	// parse a multiline message and remit to channel
	for i, v := range strings.Split(msg, "\n") {
		if i == 0 && v == "" { // Skip leading blank lines
        continue
		}
		ch.client.Notice(dest, v)
	}
}
// end TODO

func (ch *IrcChannelLogger) command(command string, args []string, line *irc.Line) string {
	// Prob not a bad idea to do some upper bounds checking to avoid overflow?
	cmdMap, cmdok := botCommand[command]
	_, argok := cmdMap[len(args)]
	log.Printf("Command(%s) ok? %v ", command, cmdok)
	log.Printf("Arguments(%s) length(%d) ok? %v ", args, len(args), argok)

	// Check to see if command with args exists
	if cmdok != true {
		return fmt.Sprintf(
			"Uknown command %s \n %s", command, botCommand["HELP"][0](ch, []string{}, line))
	} else if argok != true {
		return fmt.Sprintf(
			"Wrong Number of arguments. \n %s", botCommand[command][-1](ch, []string{}, line))
	}

    // handle the special case for the help command to avoid initialization loop
    // in command map
    if command == "HELP" && len(args) == 1 {
        if _, cmdOk := botCommand[strings.ToUpper(args[0])]; cmdOk != true {
            log.Println("Invalid command")
            return "Invalid command"
        } else {
            log.Print("Displaying HELP for command ", args[0])
            return botCommand[strings.ToUpper(args[0])][-1](ch, []string{}, line)
        }
           
    }
    
    rMsg := botCommand[command][len(args)](ch, args, line)

    log.Printf("Received (%s) from command %s with args %s", rMsg, command, args)
   
    return rMsg
}

func (ch IrcChannelLogger) timestamp() int64 {
	return int64(time.Now().Unix() - ch.time)
}

func (ch IrcChannelLogger) timestampstr() string {
	return strconv.FormatInt(ch.timestamp(), 10)
}

func (ch *IrcChannelLogger) start() {
	ch.client = irc.SimpleClient(botname)
	ch.client.SSL = false
    ch.client.EnableStateTracking()

	log.Print("Starting " + ch.rkey() + " channel logger")

	ch.client.AddHandler(irc.CONNECTED, ch.connectIRC)
	ch.client.AddHandler(join, ch.joinChan)
	ch.client.AddHandler(pmsg, ch.privMsg)
	ch.client.AddHandler(part, ch.partChan)
	ch.client.AddHandler(quit, ch.quitChan)

    if err := ch.client.Connect(fmt.Sprintf("%s:%d", ch.host, ch.port)); err != nil {
		log.Println("Failed to connect to the IRC Server: "+ch.host+" the error was: ", err)
		return
	}

	cli, err := redis.NewSynchClient()

	if err != nil {
		log.Println("Failed to connect to redis, the error was ", err)
		return
	}

	ch.redis = cli

	<-ch.quit
}

func (ch *IrcChannelLogger) recTime (conn *irc.Conn) {
    // Build initial list of users in channel
    for _, user := range conn.ST.GetChannel(ch.name).NicksStr() {
        log.Print("Recording " + user + " timestamp\n")
        ch.user_left(user)
    }
}

// Define IRC handlers
func (ch *IrcChannelLogger) quitChan(conn *irc.Conn, line *irc.Line) {
	log.Print(line.Nick + " has quit")
	ch.user_left(line.Nick)
}

func (ch *IrcChannelLogger) partChan(conn *irc.Conn, line *irc.Line) {
	// record the channel users last seen time
	log.Printf(
		"user(%s) has left the %s channel, last seen timestamp is now %d",
		line.Nick, ch.name, ch.timestampstr())
	ch.user_left(line.Nick)
}

func (ch *IrcChannelLogger) joinChan(conn *irc.Conn, line *irc.Line) {
    // This function should only record the channelstart if the botname joins.
	var chKey = ch.rkey() + sfxStart

	log.Printf(line.Nick+" has joined the %s channel", ch.name)

    if (line.Nick == ch.nick) {
        log.Printf(
            "Gobot joined channel %v, recording all traffic and lastseen", ch.name)
        channelStart, _ := ch.redis.Get(chKey)

        iStart, _ := strconv.ParseInt(string(channelStart), 10, 64)
        log.Print("Channel start: ", iStart)

        if iStart > 0 {
            log.Printf("I've been here before, my lasttime is %v", ch.time)
            ch.time = iStart
        } else {
            log.Printf(
                "I have no recollection of this place(%v) logging time.", ch.time)
            ch.time = time.Now().Unix()
            ch.redis.Set(chKey, []byte(strconv.FormatInt(ch.time, 10)))
        }
    }
    if ch.initialized == false {
        log.Printf(
            "I have no recollection of these people here(%v).", ch.name)

        // Build initial list of users in channel
        ch.recTime(conn)
        ch.initialized = true
    } else {
        ch.user_left(line.Nick) // Not really but record the time anyway
    }

}

func (ch *IrcChannelLogger) connectIRC(conn *irc.Conn, line *irc.Line) {
	log.Print("Connected to IRC Server: " + ch.host)
	ch.client.Join(ch.name)
}

func (ch *IrcChannelLogger) privMsg(conn *irc.Conn, line *irc.Line) {
    source := line.Args[0]
    parts  := strings.Fields(line.Args[1])
    target := parts[0]

    // Log the message
    log.Printf("Message received: %s, source: %s, nick: %s, channel: %s", 
        line.Args[1],
        source,
        ch.nick,
        ch.name)

    // log the message with timestamp to redis
    ch.redis.Zadd(ch.rkey(), float64(
        ch.timestamp()), []byte(fmt.Sprintf(
            "%v> %s: %s", time.Now().Format(msgDate),
            line.Nick, line.Args[1])))

    log.Printf("privmsg function, source(%v) parts(%v) ", source, parts)
    if len(parts) < 1 {
        // wtf does this even do? when is it ever going to get called?? 
		ch.quit <- true
	} else if source == ch.nick || target == ch.nick {
        /* 
         If either the source of the message was for the tracked channel name 
        or a private message to the bot, or in the channel saying the bots name.
        
        <<Process as a command >>
        */
        var dest string
        var command string
        var args []string
        
        // check for a source match, if so send a command with args
        // if a target match ..
        if source == ch.nick { 
            command = strings.ToUpper(parts[0])
            args = parts[1:]
            dest = line.Nick 
        } else { 
            command = strings.ToUpper(parts[1])
            args = parts[2:] 
            dest = ch.name
        }

        log.Printf("Command received: %s and arguments(%d): %s", command, len(args), args)
        go ch.multilineMsg(ch.command(command, args, line), dest)
    }
}

func main() {

    // remote shutdown to avoid embarrassing moments ;)
    mainquit := make(chan bool)

	// Join the command/control server
	cc := IrcChannelLogger{
		name:   "#flashnotes-dev",// + botname,
		host:   "127.0.0.1",
		port:   6669,
		nick:   botname,
		ssl:    false,
		listen: true,
	}

	cc.start()
    <- mainquit
}
