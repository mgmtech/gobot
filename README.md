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

Meet the Bots
-------------

*Parrot*  -> Hi I am ParrotBot (*sqwuak*) I can report changes to github and provide you with a diff url.  Got a cracker? 

*Burt*    -> Hi I am Burt, dont call me BurtBot and yes its spelled like the Seasame Street character. People say I am tempermental but I think its that I am not thread safe. I can eat up all of the RAM on your computer, try me sucka. But really I am supposed to take a URL and rasterize it.. yeah right.

*WebVu*   -> Hi I have a non-personal name and I am not thread-safe but I do the same as burt without all the bullshit.

:japanese_ogre:
..... more to come.. Suggest your own bots!!
