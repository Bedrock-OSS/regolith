---
permalink: /docs/node-filters
layout: single
classes: wide
title: NodeJS Filters
sidebar:
  nav: "sidebar"
---

As an asynchronous event-driven JavaScript runtime, Node.js is designed to build scalable network applications.

## Installing NodeJS

Before you can run Node filters, you will need to [install NodeJS](https://nodejs.org/en/download/).

## Running NodeJS code as Filter

The syntax for running a nodejs filter is this:

```json
{
  "runWith": "nodejs",
  "script": "./filters/example.js"
}
```

## Requirements and Dependencies

When installing, regolith will check for a `package.json` file at the top level of the filter folder.

When developing a Node filter with dependencies, you must create this file. You can create a `package.json` file yourself by using `npm init`.