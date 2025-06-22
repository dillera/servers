# Texas Hold'em Poker Server

## Project Overview

This project implements a Texas Hold'em poker game server in Go, featuring a sophisticated BOT AI system with distinct playing styles. A Python-based test client is provided to interact with the server and observe gameplay.

## Server Details

The server is built with Go and utilizes the Gin web framework for handling HTTP requests and the Gorilla WebSocket library for real-time communication with clients.

### Building and Running the Server

To build and run the server, navigate to the `server` directory and use the following commands:

```bash
# Build the server executable
go build -o poker_server .

# Run the server
./poker_server
```

The server will start on port 8080.

## BOT AI System

The server features a new BOT AI system designed to simulate a variety of playing styles, making the game more dynamic and challenging. Each BOT is assigned a unique profile that dictates its behavior.

### BOT Profiles

Each BOT's playing style is determined by a `BotProfile` struct, which includes the following parameters:

*   **VPIP (Voluntarily Puts In Pot):** How often the BOT chooses to play a hand (a measure of tightness/looseness).
*   **PFR (Pre-Flop Raise):** How often the BOT raises before the flop when they do play (a measure of aggressiveness).
*   **BluffFrequency:** How often the BOT will attempt to bluff.

### Available BOT Personalities

| Name       | Profile            | VPIP  | PFR   | Bluffing |
|------------|--------------------|-------|-------|----------|
| **Clyd BOT**   | Tight-Aggressive   | 0.20  | 0.80  | 0.10     |
| **Jim BOT**    | Loose-Passive      | 0.60  | 0.10  | 0.05     |
| **Kirk BOT**   | The Rock           | 0.10  | 0.50  | 0.01     |
| **Hulk BOT**   | The Maniac         | 0.80  | 0.70  | 0.40     |
| **Fry BOT**    | Calling Station    | 0.70  | 0.05  | 0.02     |
| **Meg BOT**    | Balanced           | 0.35  | 0.60  | 0.15     |
| **Grif BOT**   | The Bluffer        | 0.40  | 0.50  | 0.50     |
| **GPT BOT**    | GTO Pro            | 0.25  | 0.75  | 0.20     |

## Test Client

A Python-based test client is provided in the `web-test2` directory. This client connects to the server, displays the game state in the console, and allows you to observe the BOTs in action.

### Setting Up and Running the Client

To run the client, you will need Python and the `websocket-client` library. It is recommended to use a virtual environment.

```bash
# Navigate to the client directory
cd web-test2

# Create and activate a virtual environment (optional but recommended)
python -m venv venv
source venv/bin/activate  # On Windows, use `venv\Scripts\activate`

# Install the required library
pip install websocket-client

# Run the client
python client.py
```

The client will attempt to connect to the server at `ws://localhost:8080/ws`. A 5-second delay has been added to the client to ensure the server has time to fully initialize before the first connection attempt.
