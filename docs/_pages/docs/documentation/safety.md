---
permalink: /docs/safety
layout: single
classes: wide
title: Safety
sidebar:
  nav: "sidebar"
---

Please be aware that Regolith is only intended to be used by competent users. Due to the extreme power and flexibility that Regolith offers, there is an increased chance of harm to your system, if used improperly.

Please use caution when using Regolith. If you are not a developer, we highly recommend that you leave your Regolith installation in "Safe Mode".

## What is Safe Mode?

By default, Regolith will begin in safe mode: We encourage you to keep it this way!

In Safe Mode, Regolith will only run standard filters. These filters are written by the maintainers of Regolith, and we consider them extremely safe. 

Please be aware however that we cannot accept any liability. Even when running in Safe Mode, you are fully responsible for the safety of your system.

## What is Unsafe mode?

Unsafe Mode unlocks the true potential of Regolith by enabling custom filters. These filters can be written locally, and stored in your project file, or pulled down from the internet by URL, paired with the `regolith install` command.


## How do I turn off Safe Mode?

Safe Mode can be disabled by running `regolith unlock`. You will be required to accept a short liability notice before continuing.

This action will create a certificate file in your regolith folder, which is signed to your machine. It is impossible to commit this file into version control: Every system must individually accept the terms.

## Why is Regolith Unsafe?

Regolith has the ability to run arbitrary code. This code is not sandboxed, and could damage your system. When writing your own filters, you are responsible to write safe code!

Regolith also comes with the ability to download third-party filters from the internet. Regolith does not check these filters for safety. Only download and run internet filters if you are absolutely positive the author is trustworthy.

A compromised filter is able to completely destroy your system.

## Why isn't Regolith Sandboxed?

Software sandboxing is extremely difficult, especially since Regolith offers run targets in three languages, as well as a native shell integration.

Sandboxing would also limit the things our users can do. Currently, anything possible with programming can be integrated with Regolith! Sandboxing would limit this.

Additionally, we believe sandboxing may give our users a false sense of security. Since no sandbox is foolproof, we prefer our users to operate with full caution, rather than trust an imperfect solution to guard them.