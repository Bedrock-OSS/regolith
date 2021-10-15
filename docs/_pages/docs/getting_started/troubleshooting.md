---
permalink: /docs/troubleshooting
layout: single
classes: wide
title: Troubleshooting
sidebar:
  nav: "sidebar"
---

Regolith is a useful tool, but its somewhat complex compilation flow leaves room for user error. This page will explain solutions to common mistakes, as well as guide you through more complex debugging strategies. 

# General Debugging Tips

## Reading the Console

Regolith is a console application, which means that you will need to interact with it via the terminal. When Regolith runs, it will print information into the same log. This information is very useful in debugging, as Regolith will print as much useful information as it can during failure states.

Please get comfortable reading the console output, and try to become familiar with the syntax. Warnings and errors will be printed clearly.

## Check your Version

Regolith is a living, breathing application, which is receiving numerous updates. You can directly install the latest version of Regolith, or watch out for the "A new Version is Available" messages in the console output.


# Common Issues

## Regolith is not Recognized

When first installing Regolith, you may get an error message like this:

```
regolith : The term 'regolith' is not recognized as the name of a cmdlet, function, script file, or operable program. 

Check the spelling of the name, or if a path was included, verify that the path is correct and try again.
```

The most common cause of this issue is incorrect installation. Here are some troubleshooting tips:

 - 1: First, try closing your shell, and opening a new one. Then rerun the `regolith` command.
 - 2: Try a different shell. For example `gitbash` or `vscode` instead of `powershell`.
 - 3: Try reinstalling Regolith
 - 4: If you cannot get Regolith installed, you may download the stand-alone .exe, and place this in your project

## Crash when Running

The most common reason Regolith will crash is from a broken filter. The first step in debugging, is identifying which filter is failing. You can do so by navigating to the Regolith output log, and finding which filter caused the crash. 

Filter errors will be printed like `[filter][error] ... `.

