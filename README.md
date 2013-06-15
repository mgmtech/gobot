GoBot
=====
Irc Bot written in Golang, with support for sub-bots and a central registry for them.


Whats it for?
-------------
GoBot was basically a mashup of the awesome goirc libary (github.com/fluffle/goirc/)
and (alphazero) redis client. Its original intention was to log Irc conversations and 
provide a "What did I miss" feature.

By recording a users lastseen timestamp (unix time) relative
to the channels timestamp as well as the IRC channel conversation its assigned 
to GoBot records messages using a Redis Sorted Set whose message score is the timedelta
of the current time minus the start of the logging in the channel (uptime).

Messages *should* be set to expire 72 hours after they are logged so as not to eat up RAM/disk.

Eventually a logfile archive might be implemented, along with librarian functions
to retrieve older logs.


**Quite a bit has changed** 

GoBot will soon incorporate the ability to speak to sub-bots (colloquially GoBots)
using ZeroMq and a central configuration registry.

Commands
--------

GoBot help help

Bots Report
--------

Parrot  -> Parrot is configured to listen for github post-receive web hooks and report changes and a diff link.
Burt    -> Burt is a Basic URl Transformation bot, in plain English he takes a URL and returns a PNG file.
WebVu   -> Same as burt but uses C implementation of gtk-webkit-png rather than a full GTK+ stack.

:japanese_ogre:
..... more to come.. Suggest your own bots!!

Wish List
--------

A GoBot Army :construction_worker: 

