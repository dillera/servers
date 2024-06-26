# FujiNet Servers
Source to servers and clients for games and apps that work with FujiNet.

## What is a Server?
A server here in this repo is a body of code, maintained here, that will run on a computer system and respond via API calls. There are typically two types:

- Interface Servers - these connect to FujiNet clients on one end and also other Internet 'servers' and proxy and modify requests and responses in order to allow the clients (on 8bit systems) to better parse and do something with the data from the Internet.
  - Examples include News, Mastodon, Yail, Apod
  - The clients won't function without a running instance of the server hosted somewhere

- Game Servers - the game servers also connect to FujiNet clients (games) on one end and allow multiple clients to play a shared game.
  - Examples include Five Card Stud, Reversi, Lobby
  - The clients won't function without a running instance of the server hosted somewhere

# FujiNet Game System
This folder holds both server and client code for games that use FujiNet to allow multiplayer interactions.
They may or may not use the Lobby (client and server) which can advertise games and allow discovery for clients.

### 5CS
- "5cardstud" - A Multi-player/Multi-Platform Poker Server and Clients that impliment 5 Card Stud poker game. This is very much a work in progress.
  - Clients
    - "client/pc/python" - PC client, written in Python.
  - Servers
    - "dummy-server/pc/Python" - Json server written in Python, serves random hands for client testing.
    - "[server/mock-server](5cardstud/server/mock-server)" - Json Api server written in Go. It started as a mock server for the purpose of writing 5 Card Stud clients and migrated into a full server supporting multiple clients, with bots. It still supports mock tables to assist in writing/testing new clients.


# Servers

### APOD
- "apod" - Astronomy Picture of the Day fetcher. (Interface Server)
  - Fetch [NASA's Astronomy Picture of the Day (APOD)](https://apod.nasa.gov/apod/),
    convert it to a format suitable for quickly loading on an Atari (e.g.,
    80x192 16 grey shade `GRAPHICS 9`), and make it available via HTTP for
    an Atari with a #FujiNet and its `N:` device.

### Cherrysrv
- "cherrysrv" - A simple chat multi-channel server that works over TCP.

### Kaplow
- "kaplow" - A server to play a scorched earth like game.

### Networds
- "networds" - A server for a two-player word game played via mostly-RESTful HTTP requests.

