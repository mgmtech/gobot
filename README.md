GoBot
=====
IRC Logging Bot written in Go.


Whats it for?
-------------
GoBot is basically a mashup of the awesome goirc libary (github.com/fluffle/goirc/)
and redis client.. 

Its primary function is to record a users lastseen timestamp (unix time) relative
to the channels timestamp as well as the IRC channel conversation its assigned to.

GoBot records messages using a Redis Sorted Set whose message score is the timedelta
of the current time minus the start of the logging in the channel (uptime).

Messages are should be set to expire 72 hours after they are logged so as not to eat up RAM/disk.

Eventually a logfile archive might be implemented, along with librarian functions
to retrieve older logs.

Commands
--------

GoBot help help




GitHub Post-Publishing hooks
---------------------------
Slammed in last minute to publish diff urls for tracked repos



Wish List
--------
