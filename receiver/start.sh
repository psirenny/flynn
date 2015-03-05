#!/bin/sh

exec /bin/gitreceived /bin/flynn-key-check /bin/flynn-receiver --cache-key-hook /bin/flynn-push-hook
