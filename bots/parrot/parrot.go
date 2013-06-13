package main 

/*

Parrot is a simple tool that reports pushes to a configured branch and provides
a diff url on github.com

*/

const (
    gitdiffbranch = "develop"
)
import (
    "encoding/json"
    "bytes"
    "log"
    "fmt"
    "net/http"
    "text/template"
	zmq "github.com/pebbe/zmq3"
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

            m.CompBranch = gitdifbranch
            
            log.Printf("%v", m)
            log.Printf("%v pushed to github. %v -> %v repo %v", 
                m.Commits[0].Author.Name, m.Before, m.After, m.Repository)
        })

        log.Fatal(http.ListenAndServe(":8085", nil))
}


	"os"
	//"time"
	"log"
	"strings"
)

/* ZMQ */
const (
	NBR_CLIENTS  = 10
	NBR_WORKERS  = 1
	WORKER_READY = "\001" //  Signals worker is ready

    FE_URL = "tcp://127.0.0.1:5555"
	BE_URL = "ipc://backend.ipc"
)

/* GTK Webkit conversion */
const (
	DEST_FOLDER_PREFIX    = "/home/matt/Desktop/"
	OUTPUT_FORMAT_DEFAULT = "png"
)

var url22 = "http://www.flashntoes.com"

//  Our load-balancer structure, passed to reactor handlers
type lbbroker_t struct {
	frontend *zmq.Socket //  Listen to clients
	backend  *zmq.Socket //  Listen to workers
	workers  []string    //  List of ready workers
	reactor  *zmq.Reactor
}

//  Worker using REQ socket to do load-balancing
//
func worker_task() {
	var ret int = 1
	worker, _ := zmq.NewSocket(zmq.REQ)
	defer worker.Close()
	worker.Connect(BE_URL)

	//  Tell broker we're ready for work
	worker.SendMessage(WORKER_READY)

	//  Process messages as they arrive
	for {
		msg, e := worker.RecvMessage(0)
		if e != nil {
			log.Printf("Worker encountered error %v", e)
			break //  Interrupted
		}

		parts := strings.Split(msg[2], " ")
		if len(parts) == 2 {
			url := parts[0]
			file := parts[1]
			log.Printf("Conversion task accepted")
			log.Printf("Converting url %v to file %v", url, file)
            openUrl(url)
		}

		log.Printf("asd")
		if ret == 1 {
			msg[len(msg)-1] = "OK"
		} else {
			msg[len(msg)-1] = "FAIL"
		}
		worker.SendMessage(msg)
        for gtk.EventsPending() == true {
            gtk.MainIterationDo(false)
            log.Print("Gtk Main Iteration Doing..")
        }
	}
}

func openUrl(url string) {
	gtk.Init(nil)
	window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetTitle("Goburt")
	window.Connect("destroy", gtk.MainQuit)

	vbox := gtk.NewVBox(false, 1)

	entry := gtk.NewEntry()
	swin := gtk.NewScrolledWindow(nil, nil)
	swin.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)
	swin.SetShadowType(gtk.SHADOW_IN)

	webview := webkit.NewWebView()
	webview.Connect("load-committed", func() {
		entry.SetText(webview.GetUri())
	})
	webview.Connect("load-finished", func() {
		// capture pnd and quit
		log.Print("Outputting imag")
		gtk.MainQuit()
	})
	swin.Add(webview)

	vbox.Add(swin)
	entry.SetText(url)
	vbox.PackStart(entry, false, false, 0)
	entry.Connect("activate", func() {
		webview.LoadUri(entry.GetText())
	})

	window.Add(vbox)
	window.SetSizeRequest(600, 800)
	window.ShowAll()

	proxy := os.Getenv("HTTP_PROXY")
	if len(proxy) > 0 {
		soup_uri := webkit.SoupUri(proxy)
		webkit.GetDefaultSession().Set("proxy-uri", soup_uri)
		soup_uri.Free()
	}
	entry.Emit("activate")
	gtk.Main()
}

func main() {
	lbbroker := &lbbroker_t{}
	lbbroker.frontend, _ = zmq.NewSocket(zmq.ROUTER)
	lbbroker.backend, _ = zmq.NewSocket(zmq.ROUTER)
	defer lbbroker.frontend.Close()
	defer lbbroker.backend.Close()
	lbbroker.frontend.Bind(FE_URL)
	lbbroker.backend.Bind(BE_URL)

	// Client requests
	//	for client_nbr := 0; client_nbr < NBR_CLIENTS; client_nbr++ {
	//		go client_task()
	//	}

	for worker_nbr := 0; worker_nbr < NBR_WORKERS; worker_nbr++ {
		go worker_task()
	}

	//  Queue of available workers
	lbbroker.workers = make([]string, 0, 10)

	//  Prepare reactor and fire it up
	lbbroker.reactor = zmq.NewReactor()
	lbbroker.reactor.AddSocket(lbbroker.backend, zmq.POLLIN,
		func(e zmq.State) error { return handle_backend(lbbroker) })
	lbbroker.reactor.Run(-1)

}

//  In the reactor design, each time a message arrives on a socket, the
//  reactor passes it to a handler function. We have two handlers; one
//  for the frontend, one for the backend:

//  Handle input from client, on frontend
func handle_frontend(lbbroker *lbbroker_t) error {

	//  Get client request, route to first available worker
	msg, err := lbbroker.frontend.RecvMessage(0)
	if err != nil {
		return err
	}
	lbbroker.backend.SendMessage(lbbroker.workers[0], "", msg)
	lbbroker.workers = lbbroker.workers[1:]

	//  Cancel reader on frontend if we went from 1 to 0 workers
	if len(lbbroker.workers) == 0 {
		lbbroker.reactor.RemoveSocket(lbbroker.frontend)
	}
	return nil
}

//  Handle input from worker, on backend
func handle_backend(lbbroker *lbbroker_t) error {
	//  Use worker identity for load-balancing
	msg, err := lbbroker.backend.RecvMessage(0)
	if err != nil {
		return err
	}
	identity, msg := unwrap(msg)
	lbbroker.workers = append(lbbroker.workers, identity)

	//  Enable reader on frontend if we went from 0 to 1 workers
	if len(lbbroker.workers) == 1 {
		lbbroker.reactor.AddSocket(lbbroker.frontend, zmq.POLLIN,
			func(e zmq.State) error { return handle_frontend(lbbroker) })
	}

	//  Forward message to client if it's not a READY
	if msg[0] != WORKER_READY {
		lbbroker.frontend.SendMessage(msg)
	}

	return nil
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
