GoBot
=====
Irc Bot written in Golang, with support for sub-bots and a central registry for them.


Whats it for?
-------------
GoBot is now a full-featured bot with modular subbots (GoBots) controlled via 
0mq.

The original GoBot implementation was basically a mashup of the awesome goirc 
libary (github.com/fluffle/goirc/) and (alphazero) redis client. Its original 
intention was to log Irc conversations and provide a "What did I miss" feature.

**"What did I miss feature"**
By recording a users lastseen timestamp (unix time) relative
to the channels timestamp as well as the IRC channel conversation its assigned 
to GoBot records messages using a Redis Sorted Set whose message score is the timedelta
of the current time minus the start of the logging in the channel (uptime).

Messages *should* be set to expire 72 hours after they are logged so as not to eat up RAM/disk.

Commands
--------

GoBot help help

Meet the Bots
-------------

*Parrot*  -> I'm ParrotBot (*sqwuak*) I can report changes to github and provide you with a diff url.  Got a cracker? 

*Burt*    -> Hi my name is Burt, dont call me BurtBot and no my name is no spelled like the Seasame Street character. People say I am temper-mental but I think its that I am not thread safe. I can eat up all of the RAM on your computer, try me sucka. But really I am supposed to take a URL and rasterize it.. if I get around to it.

*WebVu*   -> *beep*  -I am not thread-safe- ... - I will preform the same as burt without all the bullshit - *beep*


:japanese_ogre:
..... more to come.. Suggest your own bots!!


Wish List
---------

Encryption

Ninja bots aka Stealth bots (bots loaded at runtime via Discovery, but otherwise not statically listed in the source)

Interface type definitions for ServerStart and ClientStart for the bots.

A local bot config file to configure the initial setup for bots per host.

Vektor - a side-project in the works to allow remote-loading GoBot IRC servers remotely via SSH.

