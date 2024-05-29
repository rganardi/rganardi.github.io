---
title: "Broadcom Meets Linux"
description: "when open source meets proprietary"
date: 2019-05-07T23:40:11+07:00
draft: false
---

This is madness.

Today I did what I thought would be simple linux kernel upgrade to 5.1.
<!--more-->
Somehow, my wifi (BCM4360, broadcom-wl kernel module) stops working after upgrade. First thought, check the pacman logs, I found this line

```
$ cat /var/log/pacman.log
***snip***
[2019-05-07 18:34] [ALPM-SCRIPTLET] ==> dkms install broadcom-wl/6.30.223.271 -k 5.1.0-arch1-3-local
[2019-05-07 18:34] [ALPM-SCRIPTLET] Error! Bad return status for module build on kernel: 5.1.0-arch1-3-local (x86_64)
[2019-05-07 18:34] [ALPM-SCRIPTLET] Consult /var/lib/dkms/broadcom-wl/6.30.223.271/build/make.log for more information.
[2019-05-07 18:34] [ALPM] running '90-linux-local.hook'...
***snip***
```

Alright, the dkms build for my wifi card driver (`broadcom-wl`) failed.
No wonder wifi didn't work.
Let's see what this `/var/lib/dkms/broadcom-wl/6.30.223.271/build/make.log` has to say.

```
$ cat /var/lib/dkms/broadcom-wl/6.30.223.271/build/make.log
***snip***
/var/lib/dkms/broadcom-wl/6.30.223.271/build/src/wl/sys/wl_cfg80211_hybrid.c: In function ‘wl_dev_ioctl’:
/var/lib/dkms/broadcom-wl/6.30.223.271/build/src/wl/sys/wl_cfg80211_hybrid.c:461:9: error: implicit declaration of function ‘get_ds’; did you mean ‘get_fs’? [-Werror=implicit-function-declaration]
  set_fs(get_ds());
         ^~~~~~
         get_fs
/var/lib/dkms/broadcom-wl/6.30.223.271/build/src/wl/sys/wl_cfg80211_hybrid.c:461:9: error: incompatible type for argument 1 of ‘set_fs’
  set_fs(get_ds());
         ^~~~~~~~
In file included from ./include/linux/uaccess.h:11,
                 from ./include/linux/crypto.h:26,
                 from ./include/crypto/hash.h:16,
                 from ./include/linux/uio.h:14,
                 from ./include/linux/socket.h:8,
                 from ./include/linux/compat.h:15,
                 from ./include/linux/ethtool.h:17,
                 from ./include/linux/netdevice.h:41,
                 from /var/lib/dkms/broadcom-wl/6.30.223.271/build/src/include/linuxver.h:69,
                 from /var/lib/dkms/broadcom-wl/6.30.223.271/build/src/wl/sys/wl_cfg80211_hybrid.c:26:
***snip***
```

Huh.
That's funny.
If you boot with the standard kernel package in arch (v5.0.13.arch1-1), the driver works just fine.
Somehow the problem is between the kernel version (v5.0.13) and (v5.1).

Alright, let's checkout what this `get_ds()` function do in the kernel.

```
$ cd ~/git/linux/src/
$ make tags
$ vim -t get_ds
```

Huh.
Tag not found.
Let's see what it does in (v5.0.13)

```
$ git checkout v5.0.13
$ make tags
$ vim -t get_ds
```

Alright, so it's just some macro that evaluates to another macro.
But it got deleted on v5.1?

Let's check this in the logs

```
$ git log v5.0.13...v5.1 arch/x86/include/asm/uaccess.h
commit 736706bee3298208343a76096370e4f6a5c55915
Author: Linus Torvalds <torvalds@linux-foundation.org>
Date:   Mon Mar 4 10:39:05 2019 -0800

    get rid of legacy 'get_ds()' function

    Every in-kernel use of this function defined it to KERNEL_DS (either as
    an actual define, or as an inline function).  It's an entirely
    historical artifact, and long long long ago used to actually read the
    segment selector valueof '%ds' on x86.

    Which in the kernel is always KERNEL_DS.
***snip***

```

Aha!
I think we've found it!

Linus decided to get rid of `get_ds()`, thus breaking our sad little driver.
Let's see if the fix is easy as suggested.

```
$ sed -i -e 's/get_ds()/KERNEL_DS/g' /usr/src/broadcom-wl-6.30.223.271/src/wl/sys/wl_cfg80211_hybrid.c
$ dkms build broadcom-wl/6.30.223.271 -k 5.1.0-arch1-3-local

$ modprobe wl
$ #SUCCESS!!
```

Alright!
So we've found the problem.

Now it's time to file appropriate bug reports.

```
$ pacman -Qi broadcom-wl-dkms | grep URL
URL             : https://www.broadcom.com/support/download-search/?pf=Wireless+LAN+Infrastructure
```

However, going to the URL, it's just a plain download site.
No support link, no contact us, no way to file a bug report.

I guess I'll have to live with writing it on a blog post then.

---

UPDATE 20190515: The `broadcom-wl-dkms` package in arch linux has been patched, so this is not an issue anymore.
