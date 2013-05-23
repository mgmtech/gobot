GoBot
=====
IRC Logging Bot written in Go.


Whats it for?
-------------
GoBot is basically a mashup of the awesome goirc libary (github.com/fluffle/goirc/)
and redis client.. 

Its primary function is to record a users lastseen timestamp (unix time) relative
to the channels timestamp as well as the IRC channel conversation its assigned to.

Channel messages addressed to GoBot are interpereted as commands and not recorded
as part of the channels logged conversation.

Messages not addressed to GoBot are recorded using a Redis Sorted Set whose message score is the timedelta
of the current time minus the start of the logging in the channel (uptime).

Messages are set to expire 72 hours after they are logged so as not to eat up RAM/disk.

Commands
--------

GoBot command arg1 arg2


HELP
===
Display the help message

