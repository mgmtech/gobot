package registry

/*

 The registry is a centralized configuration for gobots

 There should be a corresponding RegEntry in each bot source file, with the
variable name Registry. This entry contains the zeromq configuration, along
with the bots commandmap structure and settings specific to it.

 Bots are registered with the system by adding them here.. Hope to change this
in the future and add some type of discovery using Reflection? And somehow
dynamicall load them (not possible without forking?)
*/

// TODO: Define interface for ServerStart (NewServer) and ClientStart (NewClient)
type commandMap map[string]map[int]func() interface{}
type settings map[string]string

type RegEntry struct {
	Fend     string     // the 0mq front-end (*If applicable*)
	Bend     string     // the 0mq back-end
	Name     string     // the name of the bot (lowercase)
	Port     int16      // the tcp port (*if applicable *) to communicate with the bot (command)
	Commands commandMap // a list of commands, with contextual help
	Settings settings   // settings specific to the gobot
    WorkerReady string  // Worker Ready signal
    Workers  int      // Number of workers
}

type BotRegistry map[string]RegEntry

/* Example RegEntry: parrot
 parrot registry entry, as this does not respond to commands and does not have
 a request socket commands and Frontend is nil.
var Registry = RegEntry{
    Name: "parrot",
    Port: 556,
    Fend: "",
    Bend: "ipc://parrotbackend.ipc",
    Commands: nil,
    Settings: map[string]  string{
        "GITPUSHPORT": "8085",
        "GITDIFFBRANCH": "develop",
    },
}
*/
