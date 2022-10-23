---
permalink: /docs/safety
layout: single
classes: wide
title: Safety
sidebar:
  nav: "sidebar"
---

Please be aware that Regolith is only intended to be used by users experienced with working inside command prompt. Due to the extreme power and flexibility that Regolith offers, there is an increased chance of harm to your system, if used improperly. We cannot accept any liability. You are fully responsible for the safety of your system.

## Why is Regolith Unsafe?

Regolith has the ability to run arbitrary code. This code is not sandboxed, and could damage your system. When writing your own filters, you are responsible to write safe code!

Regolith also comes with the ability to download third-party filters from the internet. Regolith does not check these filters for safety. Only download and run internet filters if you are absolutely positive the author is trustworthy.

A compromised filter is able to completely destroy your system.

## Why isn't Regolith Sandboxed?

Software sandboxing is extremely difficult, especially since Regolith offers run targets in multiple languages, as well as a native shell integration.

Sandboxing would also limit the things our users can do. Currently, anything possible with programming can be integrated with Regolith! Sandboxing would limit this.

Additionally, we believe sandboxing may give our users a false sense of security. Since no sandbox is foolproof, we prefer our users to operate with full caution, rather than trust an imperfect solution to guard them.
