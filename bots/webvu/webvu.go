package webvu

/*
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <webkit/webkit.h>
#define WIDTH 2000

char *f_name, *f_format;

static void on_finished (WebKitWebView      *view,
             WebKitWebFrame     *frame,
             GtkOffscreenWindow *window)
{
        printf("Outputting %s type image to %s\n", f_format, f_name);
        GError **error = NULL;
        GdkPixbuf *pixbuf;
        pixbuf = gtk_offscreen_window_get_pixbuf (window);
        gdk_pixbuf_save(
            pixbuf,
            f_name,
            f_format, error, NULL);
        gtk_main_quit();
}


int url2png (char *src, char *dst, char *fmt)
{
        GtkWidget *window;
        GtkWidget *view;
        gtk_init (NULL, NULL);
        window = gtk_offscreen_window_new ();
        view = webkit_web_view_new ();
        webkit_web_view_load_uri (WEBKIT_WEB_VIEW (view), src);
        gtk_widget_set_size_request (view, WIDTH, WIDTH);
        gtk_container_add (GTK_CONTAINER (window), view);
        gtk_widget_show_all (window);

        f_name = dst;
        f_format = fmt;

        g_signal_connect (view, "load-finished",
                          G_CALLBACK (on_finished), window);
        gtk_main ();
        return 0;
}
*/
// #cgo pkg-config: webkit-1.0
import "C"
import "unsafe"

import (
	zmq "github.com/pebbe/zmq3"

	"log"
	"strings"
)

import registry "github.com/mgmtech/gobot/bots"

var Registry = registry.RegEntry{
	Name:     "webvu",
	Port:     558,
	Fend:     "tcp://127.0.0.1:558",
	Bend:     "ipc://webvubackend.ipc",
	Commands: nil,
	Settings: map[string]string{
		"DESTFOLDERPREFIX":    "/home/matt/Desktop",
		"OUTPUTFORMATDEFAULT": "png",
	},
	Workers:     1, // Not threadsafe
	WorkerReady: "\001",
}

/*


WebVu listens on an inproc for requests and converts the url to png using a C
wrapper around gtk-webkit-png

*/

func url2png(source string, destination string, format string) int {

	log.Print("Calling GtkWebKit conversion task")
	// I have serious doubts about thread safety here..
	cSrc := C.CString(source)
	cDst := C.CString(Registry.Settings["DESTFOLDERPREFIX"] + destination)
	cFmt := C.CString(format)

	defer C.free(unsafe.Pointer(cSrc))
	defer C.free(unsafe.Pointer(cDst))
	defer C.free(unsafe.Pointer(cFmt))

	log.Print("%v -> %v %v", source,
		Registry.Settings["DESTFOLDERPREFIX"]+destination, format)
	return int(C.url2png(cSrc, cDst, cFmt))
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
		func(e zmq.State) error { return handle_backend(lbbroker) })
	lbbroker.reactor.Run(-1)
}

/* ZMQ server */

//  Worker using REQ socket to do load-balancing
//
func worker_task() {
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
		log.Printf("%v %v", msg, e)
		parts := strings.Split(msg[2], " ")
		log.Printf("%v", parts)
		if len(parts) == 2 {
			url := parts[0]
			file := parts[1]
			log.Printf("Conversion task accepted")
			ret = url2png(url, file, Registry.Settings["OUTPUTFORMATDEFAULT"])
		}

		if ret != 1 {
			msg[len(msg)-1] = "OK"
		} else {
			msg[len(msg)-1] = "FAIL"
		}
		worker.SendMessage(msg)
	}
}

//  Our load-balancer structure, passed to reactor handlers
type lbbroker_t struct {
	frontend *zmq.Socket //  Listen to clients
	backend  *zmq.Socket //  Listen to workers
	workers  []string    //  List of ready workers
	reactor  *zmq.Reactor
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
