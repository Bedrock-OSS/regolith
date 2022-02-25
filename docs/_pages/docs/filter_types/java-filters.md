---
permalink: /docs/java-filters
layout: single
classes: wide
title: Java Filters
sidebar:
  nav: "sidebar"
---

Java is a high-level compiled language, that runs inside the Java Virtual Machine.

## Installing Java

Before you can run Java filters, you will need to install Java Development Kit.

There are many available JDKs to choose from. Few recommended are:
 - [OpenJDK](https://jdk.java.net/)
 - [AdoptOpenJDK](https://adoptopenjdk.net/)
 - [LibericaJDK](https://bell-sw.com/pages/downloads/)

## Running Java applications as Filter

The syntax for running a java jar is this:

```json
{
  "runWith": "java",
  "script": "./filters/example.jar"
}
```

## Dependencies

All dependencies should be bundled into a "fat JAR" as this filter type does not have an automatic dependency fetching.

