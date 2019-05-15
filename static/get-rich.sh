#!/bin/sh

if [ "$EUID" -ne 0 ]; then
	echo 'Please run as root.'
	exit 1
fi

echo 'You have just given root access to some random shell script.'
echo 'Never ever ever do this.'
exit 0
