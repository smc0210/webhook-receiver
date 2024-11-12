# Webhook Receiver

## Overview

This application provides a simple server to receive webhook events and displays them in real-time on a web interface. It also integrates with ngrok to provide a public URL, making it easy to test webhooks locally.

## Features

- Receives webhook events on a specified port
- Displays events in real-time on the web interface
- Exposes a public URL for the webhook receiver via ngrok
- Automatically copies the public URL to the clipboard

## Usage

```bash
cd webhook
./webhook-receiver
```

Use the `Forwarding` address obtained to send webhooks.

#### Build

To build the binary, execute the following command:

```bash
go build -o webhook-receiver
```


#### TODO

- [ ] ngrok static domain guide
- [ ] Other case ( 5xx-200, 5xx only)
- [ ] Support x-www-form-urlencoded

---
*webhook-receiver* is primarily distributed under the terms of both the [MIT license]
and the [Apache License (Version 2.0)]. See [COPYRIGHT] for details.

[MIT license]: LICENSE-MIT
[Apache License (Version 2.0)]: LICENSE-APACHE
[COPYRIGHT]: COPYRIGHT