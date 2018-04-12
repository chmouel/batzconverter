# tzconverter
Show times in different timezone with bash

* Author: Chmouel Boudjnah <chmouel@chmouel.com
* License: GPL

## Usage
```bash
% tz
% tz 10h30
% tz 10h30 next week
% tz 11:00 next thursday
```

TZ  will show all different timezone for the timeformat

You can as well add multiple timezones directly on the command line like this :
```bash
% tz +America/Chicago +UTC 10h00 tomorrow
```

By default this script will try to detect your current timezone, if you want
to say something like this, show me the different times tomorrow at 10h00 UTC
you can do :

```bash
% tz +UTC -t UTC 10h00 tomorrow
````

*The order here is important first have the + to add the UTC timezone and set
the base timezone to UTC to calculate the others*

## Install

This needs gnu date, on MacOSX just install gnuutils from brew

It needs bash v4 too, you need to install it from brew as well on MacOSX
