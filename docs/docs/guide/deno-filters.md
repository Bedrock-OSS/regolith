---
title: Deno Filters
---

# Deno Filters

Deno is a new age javascript/typescript runtime with first class typescript support.

## Installing Deno

Before you can run deno filters, you will need to [install Deno](https://deno.land/).

## Running Deno code as Filter

The syntax for running a deno filter is this:

```json
{
  "runWith": "deno",
  "script": "./filters/example.ts"
}
```

## Requirements and Dependencies

Deno manages and installs dependencies on runtime. So no additional setup required.
