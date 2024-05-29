---
title: "smailx GPG Filter"
date: 2020-05-14T14:20:20+02:00
draft: true
---

I've quite a bit of time on my hands, and I've been feeling unhappy with my
email client setup.

Mutt feels archaic and I don't really know how to even start creating
keybindings that makes sense.

Aerc is better in that respect, but there are a lot of edge cases that is still
annoying (sometimes the imap connection is closed because of a mistake in
computing message length). PGP support is also sketchy. You have to maintain a
separate keyring for `aerc`, which is stupid.


I've decided to switch to using UNIX mail as my email client, however it
doesn't have any PGP support. I thought a simple filter would do the job.

Here's a [go script](/src/mailx-filter.go) decodes an email that is piped in
on-the-fly. It even pipes the message through `w3m` if a `text/html` part is
detected.
