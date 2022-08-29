---
permalink: /docs/dotnet-filters
layout: single
classes: wide
title: .NET Filters
sidebar:
  nav: "sidebar"
---

.NET filters run programs written in the .NET framework using `dotnet` command.

## Installing .NET

Before you can run .NET filters, you will need to install
[.NET Runtime](https://dotnet.microsoft.com/download).

## Running .NET applications as Filter

The syntax for running a .NET filter is this:

```json
{
  "runWith": "dotnet",
  "path": "./filters/example.dll"
}
```
