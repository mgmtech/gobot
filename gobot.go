package main

/*
   GoBot is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.

*/

/*
    GoBot is a mix-mash of various Golang libraries to facilitate yet another
IRC Bot, with sub-bots and a registry.. I chose this path as my first real Golang
program as its something I find interesting at the moment (Irc)..

    Eventually GoBot should be able to start/stop the sub-bots and provides a
central configuration method for them (the registry). The registry contains the
appropriate channels, a commands map with contextual help, and settings related
to the bot.. This will hopefully allow adding/removing bots to be easy and pain
free. GoBot uses ZeroMq to speak to its sub-bots, eventually by name these bots
will be invoked using GoBot as their proxy. By virtue of ZeroMq there are a lot
os possible combinations and I am certain that things will be under heavy changes
most of the time.

*/

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

import redis "github.com/alphazero/Go-Redis"

import irc "github.com/fluffle/goirc/client"

// TODO: implement interfaces which use multi-methods? possible?

//import bots "github.com/mgmtech/gobots"

// use of interfaces methods, solve the issue of importing bots here!
import parrot "github.com/mgmtech/gobots/parrot"

func (ch *IrcChannelLogger) listentoparrot() {

	client := parrot.CliStart() // bots.Roster["parrot"].CliStart()
	defer client.Close()
	//log.Print("conntecting to ", bots.Registry["parrot"].Bend)

	for {
		msg, _ := client.Recv(0)
		log.Print("Git-parrot msg -> ", msg)
		ch.multilineMsg(msg, ch.name)
	}
}

const (
	// bots
	botname string = "GoBot"

	// irc commands
	join  string = "JOIN"
	part         = "PART"
	pmsg         = "PRIVMSG"
	quit         = "QUIT"
	names        = "NAMES"

	// rclient key suffixes
	sfxLast  = ":lastseen"
	sfxStart = ":starttime"

	msgDate = "1/2/06 15:04:05"
)

type IrcChannelLogger struct {
	name        string // The name of the channel
	time        int64  // The unix time that the logging started
	host        string // The hostname of the IRC server
	port        int16  // The port number of the IRC service
	nick        string // The bots nickname to use
	ssl         bool   // SSL?
	listen      bool   // Listen for commands
	initialized bool   // Channel initialized?

	client *irc.Conn    // The IRC Client for the channel
	rclient  redis.Client // The rclient client connection
	done   chan bool    // A channel to signal close
}

/* rclient key helpers */
func (ch *IrcChannelLogger) rkey() string            { return fmt.Sprintf("%s:%d:%s", ch.host, ch.port, ch.name) }
func (ch *IrcChannelLogger) ukey(user string) string { return fmt.Sprintf("%s:%s", ch.rkey(), user) }

/*
Bot commands and contextual help map

Commands are structured in a map[string][int] of functions which return a string.
-1 index is for the help message, the rest correspond to the number of arguments for the command.

*/
var botCommand = map[string]map[int]func(*IrcChannelLogger, []string, *irc.Line) string{
	"REDISCHECK": map[int]func(*IrcChannelLogger, []string, *irc.Line) string{
		-1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return "rclientCHECK - Basic rclient connection tests"
		},
		0: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			_, ok := ch.rclient.AllKeys()
			return fmt.Sprintf("rclient All keys command stats: (%v)", ok == nil)
		},
	},
	"BOTS": map[int]func(*IrcChannelLogger, []string, *irc.Line) string{
		-1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
            return "BOTS - :ninja:"
		},
		0: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
            return fmt.Sprintf("%v", ":#")
            //bots.Registry) // their the mots
		},
	},
	"WHO": map[int]func(*IrcChannelLogger, []string, *irc.Line) string{
		-1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
            return "WHO - :neckbeard:"
		},
		0: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
            return fmt.Sprintf("%v", ch.client.ST.GetChannel(ch.name).NicksStr())
            //bots.Registry) // their the mots
		},
	},
	"KEYS": map[int]func(*IrcChannelLogger, []string, *irc.Line) string{
		-1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return "KEYS - Show the rclient keys in play"
		},
		0: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return fmt.Sprintf("Channel key is (%v) user key for GoBot is (%v), channel start (%v) -> %v %v",
				ch.rkey(), ch.ukey("GoBot"), ch.rkey()+sfxLast, ch.rkey()+sfxStart, ch.time)
		},
	},
	"VARS": map[int]func(*IrcChannelLogger, []string, *irc.Line) string{
		-1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return "VARS - Show the channel configuration parameters"
		},
		0: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return fmt.Sprintf("Channel configuration (%v) ", ch)
		},
	},
	"DEATHKISS": map[int]func(*IrcChannelLogger, []string, *irc.Line) string{
		-1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return "Whatchoo ¿awkin` bought wilis¿ you need to guess the magic number fool!"
		},
		0: func(ch *IrcChannelLogger, args []string, line *irc.Line) string { ch.client.Quit(); return "" },
	},
	"HELP": map[int]func(*IrcChannelLogger, []string, *irc.Line) string{
		-1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return `
command [arg1] [arg2]
--------------------------------
HELP - Display this help message
HISTORY - Show users last 20 messages
LASTSAW - Show users last seen timestamp
DIE - Immediately close the channel logger (EXPERIMENTAL)
KEYS - Show the channels keys, or an example of them
TIMESTAMP - Show the channels timestamp
rclientCHECK - Test the rclient connection
VARS - Remit the local configuration parameters`
		},
		0: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return "HELP command - Show help for the command."
		},
		1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return "*special case* you shouldnt be here.." // The channel logger should handle this special case as self referenceing is not allowed
			//due to Initialization loop, break out into function or initialize a helpmap from -1 indexes. botCommand[args[0]][-1](ch, args, line)
		},
	},
	"HISTORY": map[int]func(*IrcChannelLogger, []string, *irc.Line) string{
		-1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return "HISTORY [n] - Display the last n messages, ommitting n defaults to 20."
		},
		0: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			// retrieve missed messages here and remit as multi-line string.
			rVal := ch.messages(
				int(ch.lastseen(line.Nick)), //start score
				int(ch.timestamp()))         // end score

			log.Printf("%v", rVal)

			var msg string
			for _, v := range rVal {
				msg += string(v)
			}

			return msg
		},
	},
	"LASTSAW": map[int]func(*IrcChannelLogger, []string, *irc.Line) string{
		-1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return "LASTSAW [user] - Display the user lastseen timestamp, nick of user sending by default."
		},
		0: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return fmt.Sprintf("%s, last saw you %s UNIX timestamp", line.Nick, ch.lastseenstr(line.Nick))
		},
		1: func(ch *IrcChannelLogger, args []string, line *irc.Line) string {
			return fmt.Sprintf("%s was last seen %s UNIX timestamp", args[0], ch.lastseenstr(args[0]))
		},
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

func (ch *IrcChannelLogger) user_left(user string) {
	ch.rclient.Set(ch.ukey(user)+sfxLast, []byte(ch.timestampstr()))
}

func (ch *IrcChannelLogger) lastseen(user string) int64 {
	v, _ := ch.rclient.Get(ch.ukey(user) + sfxLast)
	i, _ := strconv.ParseInt(string(v), 10, 64)
	log.Printf("Last seen for user %s was %d", user, i)
	return i
}

func (ch *IrcChannelLogger) missed(user string) int32 {
	value, err := ch.rclient.Zrangebyscore(ch.rkey(), float64(ch.lastseen(user)), float64(ch.timestamp()))
	if err != nil {
		log.Printf("Error was %v, value is %v", err, value)
	} else {
		return 0
	}
	return -1
}

func (ch *IrcChannelLogger) messages(start, end int) string {
	rVal, err := ch.rclient.Zrangebyscore(ch.rkey(), float64(start), float64(end))

	log.Printf("%v Message retrived", len(rVal))
	if err != nil {
		return fmt.Sprintf("Problem geting messages: %s %s (err = %v, %v)", start, end, err, rVal)
	} else {
		return string(bytes.Join(rVal, []byte{'\n'}))
	}
}

func (ch *IrcChannelLogger) lastseenstr(user string) string {
	return strconv.FormatInt(ch.lastseen(user), 10)
}

// TODO: refactor this duplicate code, allow it to respond to a different channel/person
func (ch *IrcChannelLogger) multilineMsg(msg, dest string) {
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
		log.Println("Failed to connect to rclient, the error was ", err)
		return
	}

	ch.rclient = cli
	<-ch.done
}

func (ch *IrcChannelLogger) recTime(conn *irc.Conn) {
	// Build initial list of users in channel
	for _, user := range conn.ST.GetChannel(ch.name).NicksStr() {
		log.Print("Recording " + user + " timestamp\n")
		ch.user_left(user)
	}
}

// Define IRC handlers
func (ch *IrcChannelLogger) quitChan(conn *irc.Conn, line *irc.Line) {
	log.Print(line.Nick + " has quit")

	if line.Nick == ch.name {
		log.Print("Im dying!!!")
		ch.done <- true
	}
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

	if line.Nick == ch.nick {
		log.Printf(
			"Gobot joined channel %v, recording all traffic and lastseen", ch.name)
		channelStart, _ := ch.rclient.Get(chKey)

		iStart, _ := strconv.ParseInt(string(channelStart), 10, 64)

		if ch.time != 0 { // if it currently has channel time
			return
		} else if iStart > 0 {
			log.Printf("I've been here before, my lasttime is %v", iStart)
			ch.time = iStart
		} else {
			log.Printf(
				"I have no recollection of this place(%v) logging time.", ch.time)
			ch.time = time.Now().Unix()
			ch.rclient.Set(chKey, []byte(strconv.FormatInt(ch.time, 10)))
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
	parts := strings.Fields(line.Args[1])
	target := parts[0]

	// Log the message
	log.Printf("Message received: %s, source: %s, nick: %s, channel: %s",
		line.Args[1],
		source,
		ch.nick,
		ch.name)

	// log the message with timestamp to rclient
	ch.rclient.Zadd(ch.rkey(), float64(
		ch.timestamp()), []byte(fmt.Sprintf(
		"%v> %s: %s", time.Now().Format(msgDate),
		line.Nick, line.Args[1])))

	log.Printf("privmsg function, source(%v) parts(%v) ", source, parts)
	if len(parts) < 1 {
		// wtf does this even do? when is it ever going to get called??
		log.Print("Hi")
		ch.done <- true
	} else if source == ch.nick || target == ch.nick && len(parts) > 0 {
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
		} else if target == ch.nick && len(parts) >= 2 {
			command = strings.ToUpper(parts[1])
			args = parts[2:]
			dest = ch.name
		}

		if command != "" {
			log.Printf("Command received: %s and arguments(%d): %s", command, len(args), args)
			go ch.multilineMsg(ch.command(command, args, line), dest)
		}
	}
}

func main() {

	// Join the command/control server
	cc := IrcChannelLogger{
		name:   "#flashnotes-dev",
		host:   "127.0.0.1",
		port:   6667,
		nick:   botname,
		ssl:    false,
		listen: true,
		done:   make(chan bool),
	}

	//bots.Start()

	go cc.listentoparrot()
	cc.start()
}
