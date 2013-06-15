package main 

/*

Parrot is a simple tool that reports pushes to a configured branch and provides
a diff url on github.com

Ideally this bot will connect up to clients from GoBot over an agreed upon config
in the registry, GoBot will then receive replies with the payload into a dest
channel

*/
import (
    "encoding/json"
    "bytes"
    "log"
    "fmt"
    "net/http"
    "text/template"

	zmq "github.com/pebbe/zmq3"
)


// TODO: Initialize entry from centralized registry.

var commands map[string]map[int]func()interface{}

type RegEntry struct {
    Fend string
    Bend string
    Name string
    Port int16
}

var registry = RegEntry{ 
    Name: "parrot", 
    Port: 556,
    Fend: "tcp://127.0.0.1:556", 
    Bend: "ipc://parrotbackend.ipc"}

const (
    gitdiffbranch = "develop"
)

/* Structs to map to the git post-receiver web hook payload */
type GitAuthor struct {
    Name string
    Email string
}

type GitRepo struct {
    Name string
    Url string
    Owner GitAuthor
}

type GitCommit struct {
    Message string
    Timestamp string
    Url string
    Author GitAuthor
}

type GitWebHookPayload struct {
    Before string
    After string
    Commits []GitCommit
    Repository GitRepo
    CompBranch string
}
/* GitWebHookPayload Json */

/* String template for url functions */
const (
	githubTemplates = `
        {{ define "git-url" }}http://www.github.com{{ end }}
        {{ define "git-repo" }}{{ .Repository.Url }}{{ end }}
        {{ define "git-compare" }}{{ .Repository.Url }}/compare/{{ .CompBranch }}...{{ .After }}{{ end }}`
)
var tmplGit = template.Must(template.New("git").Parse(githubTemplates))
/* Url string template */

func info (msg string) { log.Printf("INFO (Parrot)-> %v", msg) } 

func (payload GitWebHookPayload) String() string {

    buff := bytes.NewBufferString("")

	if err := tmplGit.ExecuteTemplate(buff, "git-compare", payload); err != nil {
		log.Print("Error executing template")
        log.Print(err)
		return fmt.Sprintf("Error %v", err) // ERROR ! XXX: implement error handling fool
	}
	
    return fmt.Sprintf("%v", buff)
}

func main() {
    // TODO: Wire up c&c using zmq
    //  Implement a Bot-registry to contain common ports, functions and 
    // help/command strings including settings.. etc
    // use functional closures to possibly implement commands out of origin

    // Link up publisher socket, could use Multicast here..
    client, err := zmq.NewSocket(zmq.PUB)
    if err != nil { log.Fatal("FAiled to connect push front-end %v", registry.Bend)}
    defer client.Close()
    client.Bind(registry.Bend)
    // Handle the post-receive from github.com..
    http.HandleFunc("/post-receive", 
        func(w http.ResponseWriter, r *http.Request) {
            payload := r.FormValue("payload")
            log.Printf("Received github.com payload from github")

            var m GitWebHookPayload

            err := json.Unmarshal([]byte(payload), &m) 

            if err != nil {
                log.Println("Error unpacking json:", err)
            }

            m.CompBranch = gitdiffbranch
            var resp_str = fmt.Sprintf("(parrot) %v pushed changes to %v, %v", 
                m.Commits[0].Author.Name, m.Repository.Name, m)
           
            log.Printf("%v", resp_str)
            client.Send(resp_str, 0)
        })

        log.Fatal(http.ListenAndServe(":8085", nil))

}

//  Pops frame off front of message and returns it as 'head'
//  If next frame is empty, pops that empty frame.
//  Return remaining frames of message as 'tail'
func unwrap(msg []string) (head string, tail []string) {
	head = msg[0]
	if len(msg) > 1 && msg[1] == "" {
		tail = msg[2:]
	} else {
		tail = msg[1:]
	}
	return
}
