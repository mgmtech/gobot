frontend = "tcp://localhost:5555"

sites_test = [
    'http://www.google.com/',
    'http://www.flashnotes.com',
    'http://www.yahoo.com',
    'http://www.fsf.org',
]


import zmq
context = zmq.Context()
socket = context.socket(zmq.REQ)
identity = "Client-1"
socket.setsockopt(zmq.IDENTITY, identity) #Set client identity. Makes tracing easier
socket.connect(frontend)

for site in sites_test:
    socket.send("%s cheeseburger" % site)
    print socket.recv()

