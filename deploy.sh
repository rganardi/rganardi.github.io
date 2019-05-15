#!/bin/sh

ssh -L9418:localhost:9418 debbie.gcloud 'git daemon --verbose --base-path=$HOME/git/ --reuseaddr'
