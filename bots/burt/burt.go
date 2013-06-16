package main

/*

Burt is a Basic Url Tranformation using Matz GtkWebKit go library and zeromq

This the first bot that got me thinking about using a task registry and zeromq
to wire up command and controlled bot which using various zermomq and native go
routines can easily scale.

1 issue is that most of this is not threadsafe so a process broker might be
needed.

Ideally a REQ-DEALER or REP-ROUTER pattern should be used, or the MT example
for a multi-threaded example.

*/

import (
	gtk "github.com/mattn/go-gtk/gtk"
	"github.com/mattn/go-webkit/webkit"
	zmq "github.com/pebbe/zmq3"

	"log"
	"os"
	"strings"
	"time"
)

import registry "github.com/mgmtech/gobot/bots"

var Registry = registry.RegEntry{
	Name:     "burt",
	Port:     557,
	Fend:     "tcp://127.0.0.1:557",
	Bend:     "ipc://burtbackend.ipc",
	Commands: nil,
	Settings: map[string]string{
		"DESTFOLDERPREFIX":    "/home/matt/Desktop",
		"OUTPUTFORMATDEFAULT": "png",
	},
	Workers:     1, // Not threadsafe
	WorkerReady: "\001",
}

//  Our load-balancer structure, passed to reactor handlers
type lbbroker_t struct {
	frontend *zmq.Socket //  Listen to clients
	backend  *zmq.Socket //  Listen to workers
	workers  []string    //  List of ready workers
	reactor  *zmq.Reactor
}

func retire_window(w *gtk.Window, chd chan bool) {
	<-chd
	defer w.Emit("destroy")
}

//  Worker using REQ socket to do load-balancing
//
func worker_task() {
	var chDone = make(chan bool)
	var ret int = 1
	worker, _ := zmq.NewSocket(zmq.REQ)
	defer worker.Close()
	worker.Connect(Registry.Bend)

	//  Tell broker we're ready for work
	worker.SendMessage(Registry.WorkerReady)

	//  Process messages as they arrive
	for {
		msg, e := worker.RecvMessage(0)
		if e != nil {
			log.Printf("Worker encountered error %v", e)
			break //  Interrupted
		}

		parts := strings.Split(msg[2], " ")
		if len(parts) == 2 {
			// Gtk Initialize
			gtk.Init(nil)

			// Create new GTK Window
			worker_window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
			worker_window.SetTitle("Goburt Worker ")
			worker_window.Connect("destroy", gtk.MainQuit)

			url := parts[0]
			file := parts[1]
			log.Printf("Conversion task accepted")
			log.Printf("Converting url %v to file %v", url, file)

			vbox := openUrl(chDone, url)
			worker_window.Add(vbox)
			worker_window.SetSizeRequest(600, 800)
			worker_window.ShowAll()
			defer worker_window.Emit("destroy")

			gtk.MainIteration()
			log.Print("First Gtk Loop iteration complete, sending done")
		}

		if ret == 1 {
			msg[len(msg)-1] = "OK"
		} else {
			msg[len(msg)-1] = "FAIL"
		}

		worker.SendMessage(msg)
	}
}

func gtkLoop() {
	for gtk.EventsPending() == true {
		gtk.MainIterationDo(false)
		log.Print(".")
	}
}

func openUrl(chDone chan bool, url string) *gtk.VBox {
	vbox := gtk.NewVBox(false, 1)

	entry := gtk.NewEntry()
	swin := gtk.NewScrolledWindow(nil, nil)
	swin.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)
	swin.SetShadowType(gtk.SHADOW_IN)

	webview := webkit.NewWebView()
	webview.Connect("load-committed", func() {
		entry.SetText(webview.GetUri())
		gtkLoop()
	})
	webview.Connect("load-finished", func() {
		// capture pnd and quit
		gtkLoop()
		log.Print("Outputting imag")
		chDone <- true
	})
	swin.Add(webview)

	vbox.Add(swin)
	entry.SetText(url)
	vbox.PackStart(entry, false, false, 0)
	entry.Connect("activate", func() {
		webview.LoadUri(entry.GetText())
		gtkLoop()
	})

	proxy := os.Getenv("HTTP_PROXY")
	if len(proxy) > 0 {
		soup_uri := webkit.SoupUri(proxy)
		webkit.GetDefaultSession().Set("proxy-uri", soup_uri)
		soup_uri.Free()
	}

	entry.Emit("activate")
	gtkLoop()
	return vbox
}

func main() {

	lbbroker := &lbbroker_t{}
	lbbroker.frontend, _ = zmq.NewSocket(zmq.ROUTER)
	lbbroker.backend, _ = zmq.NewSocket(zmq.ROUTER)
	defer lbbroker.frontend.Close()
	defer lbbroker.backend.Close()
	lbbroker.frontend.Bind(Registry.Fend)
	lbbroker.backend.Bind(Registry.Bend)

	for worker_nbr := 0; worker_nbr < Registry.Workers; worker_nbr++ {
		go worker_task()
	}

	//  Queue of available workers
	lbbroker.workers = make([]string, 0, 10)

	//  Prepare reactor and fire it up
	lbbroker.reactor = zmq.NewReactor()
	lbbroker.reactor.AddSocket(lbbroker.backend, zmq.POLLIN,
		func(e zmq.State) error { gtkLoop(); return handle_backend(lbbroker) })
	//bbroker.reactor.AddChannelTime(time.Tick(time.Second), 1, func(a interface{}) error { gtk.MainIterationDo(true); return nil })
	lbbroker.reactor.Run(time.Second)
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
	if msg[0] != Registry.WorkerReady {
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
