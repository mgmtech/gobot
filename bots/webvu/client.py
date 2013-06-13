frontend = "ipc://frontend.ipc"

sites_test = [
    ('http://www.google.com/', 'test1.png'),
    ('http://www.flashnotes.com','test2.png'),
    ('http://www.yahoo.com','test3.png'),
    ('http://www.fsf.org','test4.png'),
]


import zmq
context = zmq.Context()
socket = context.socket(zmq.REQ)
identity = "Client-1"
socket.setsockopt(zmq.IDENTITY, identity) #Set client identity. Makes tracing easier
socket.connect(frontend)

for k, v in sites_test:
    print k
    print v
    socket.send("%s -> %s" % (k,v))
    print socket.recv()

