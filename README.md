GoBot
=====
Irc Bot written in Golang, with support for sub-bots and a central registry for them.

:ok_woman: its something   
:wavy_dash:


TODO: Implement ping/pong using 0mq as the connection borks when the laptop is 
put to sleep.

Implement re-connect logic in the event of network failure. . 

irc.recv() EOF line 224


Whats it for?
-------------

The original GoBots intention was to log Irc conversations and provide a 
"What did I miss" feature.

**"What did I miss feature"**
By recording a users lastseen timestamp (unix time) relative
to the channels timestamp as well as the IRC channel conversation its assigned 
to GoBot records messages using a Redis Sorted Set whose message score is the timedelta
of the current time minus the start of the logging in the channel (uptime).

Redis messages *should* be set to expire 72 hours after they are logged so as not to eat up RAM/disk.

Commands
--------

GoBot help help

Meet the Bots
-------------

checkout the GoBots repo at http://www.github.com/mgmtech/gobots

:japanese_ogre:
..... more to come.. Suggest your own bots!!


Wish List
---------

Support for sub-bots and a central registry for them

Encryption and a backdoor for the NSA :smirk:

Ninja bots :ninja: (bots loaded at runtime via Discovery, but otherwise not statically listed in the source)

Interface type definitions for ServerStart and ClientStart for the bots.

A local bot config file to configure the initial setup for bots per host.
