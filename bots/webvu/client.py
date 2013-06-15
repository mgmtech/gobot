frontend = "ipc://frontend.ipc"

sites_test = [
    'http://www.google.com/',
    'http://www.flashnotes.com',
    'http://www.msdn.com',
    'http://www.yahoo.com',
    'http://www.google.com/news',
    'http://www.fsf.org',
    'http://www.afitzgraphics.com',
    'http://www.cocacola.com',
    'http://www.elementalstudios.com',
    'http://www.9gag.com',
    'http://www.nasa.gov',
    'http://www.ieee.org',
    'http://www.cnn.com',
    'http://redis.io',
    'http://www.zeromq.org'
]


import zmq
context = zmq.Context()
socket = context.socket(zmq.REQ)
identity = "Client-WebVu-Test"
socket.setsockopt(zmq.IDENTITY, identity) #Set client identity. Makes tracing easier
socket.connect(frontend)

c = 0
for s in  sites_test:
    socket.send("%s test%s" % (s, c))
    print "{0} -> {1}".format(s, socket.recv())
    c += 1

