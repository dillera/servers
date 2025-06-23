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

**Debug Mode:**
To enable detailed request and response logging for debugging client programs, run the server with the `-debug` flag:

```bash
./poker_server -debug
```

When debug mode is enabled, the server will print the full JSON of incoming requests and outgoing responses to the console, which is invaluable for developing and troubleshooting new client applications.

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

## Deep Dive: Game Logic and Texas Hold'em Flow

The `gameLogic.go` file is the heart of the Texas Hold'em server, managing the game state, player actions, and round progression. The game follows standard Texas Hold'em rules, progressing through distinct betting rounds.

### Game State (`GameState` Struct)

The `GameState` struct holds all the critical information about the current game:

*   **`GamesPlayed`**: Total number of games completed.
*   **`Round`**: The current betting round (0: Lobby, 1: Pre-flop, 2: Flop, 3: Turn, 4: River).
*   **`RoundName`**: A descriptive name for the current round (e.g., "Pre-flop", "Flop").
*   **`Pot`**: The total amount of chips in the pot.
*   **`CurrentBet`**: The current highest bet placed in the active betting round.
*   **`ActivePlayer`**: The index of the player whose turn it is.
*   **`Players`**: An array of `Player` structs, each containing details like name, status, purse, hand, and `BotProfile`.
*   **`CommunityCards`**: An array of `card` structs representing the community cards dealt on the board.
*   **`deck`**: The shuffled deck of cards for the current hand.
*   **`Winner`**: The name of the winner of the last hand.

### Game Flow and Round Progression

The game progresses through a series of rounds, managed primarily by the `RunGameLogic` and `newRound` functions.

1.  **Initialization (`createGameState`)**:
    *   When a new table is created, `createGameState` initializes the `GameState` struct.
    *   It creates a standard 52-card deck.
    *   It pre-populates the player pool with BOTs, assigning them their unique `BotProfile`s.
    *   The game starts in `Round 0` (Lobby/Waiting state).

2.  **Starting a New Hand (`newRound` - Round 1)**:
    *   `newRound` is called to start a new hand or advance to the next betting round.
    *   If `Round` is 0 (start of a new game):
        *   `GamesPlayed` is incremented.
        *   Player statuses are reset to `STATUS_PLAYING`.
        *   Player hands are cleared, and purses are maintained (or reset for bots with low chips).
        *   The deck is shuffled 7 times.
        *   **Hole Cards Deal**: Each player is dealt 2 private hole cards.
        *   **Blinds**: Small Blind (SB) and Big Blind (BB) are posted by designated players (currently simplified, but designed for rotation).
        *   `CurrentBet` is set to the BB amount.
        *   `ActivePlayer` is set to the player to the left of the Big Blind.
        *   `Round` is set to 1, and `RoundName` to "Pre-flop".

3.  **Betting Rounds (`RunGameLogic` and `performMove`)**:
    *   The `RunGameLogic` function continuously checks the game state and manages player turns.
    *   For the `ActivePlayer`:
        *   If it's a human player, the server waits for a move via the `/move` API endpoint.
        *   If it's a BOT, `getBotMove` is called to determine the BOT's action.
    *   **`getValidMoves`**: This function dynamically determines the available moves (Fold, Check, Call, Bet, Raise, All-in) based on the `CurrentBet`, player's `Purse`, and previous actions.
    *   **`performMove`**: This function executes the chosen move:
        *   Updates the player's `Bet` and `Purse`.
        *   Adjusts the `Pot`.
        *   Updates `CurrentBet` if a new high bet is made.
        *   Handles player status changes (e.g., `STATUS_FOLDED`, `STATUS_ALL_IN`).
        *   Advances `ActivePlayer` to the next valid player.
    *   A betting round ends when all active players have either folded, gone all-in, or matched the `CurrentBet`.

4.  **Community Card Dealing (`dealCommunityCards`)**:
    *   After the Pre-flop betting round concludes, `dealCommunityCards` is called:
        *   **Flop (Round 2)**: 3 community cards are dealt. `RoundName` becomes "Flop".
        *   **Turn (Round 3)**: 1 more community card is dealt. `RoundName` becomes "Turn".
        *   **River (Round 4)**: The final (5th) community card is dealt. `RoundName` becomes "River".
    *   After each community card deal, a new betting round begins, resetting `CurrentBet` to 0 and allowing players to make new actions.

5.  **Hand Evaluation and Game End (`endGame`)**:
    *   After the River betting round, or if all but one player folds, `endGame` is called.
    *   **`getRank`**: This function (using the `cardrank` library) evaluates the best 5-card hand for each active player using their hole cards and the community cards.
    *   The winner(s) are determined, and the `Pot` is awarded.
    *   `Winner` and `LastResult` fields in `GameState` are updated.
    *   The game returns to `Round 0` to await a new hand.

### BOT AI (`getBotMove` and `getStartingHandStrength`)

This section details the enhanced BOT AI logic:

*   **`getBotMove()`**: This is the core AI decision-making function, called for BOT players.
    *   It retrieves the BOT's `BotProfile` (VPIP, PFR, BluffFrequency).
    *   **Pre-flop Strategy (Round 1)**:
        *   **`getStartingHandStrength()`**: A new helper function evaluates the BOT's two hole cards and assigns a strength score (0-10).
        *   **VPIP Logic**: The BOT decides whether to play the hand based on its `VPIP` and the `handStrength`. Tighter BOTs (lower VPIP) require stronger hands.
        *   **PFR Logic**: If the BOT decides to play and has a strong hand, it will consider raising based on its `PFR` (aggressiveness).
        *   **Bluffing**: The BOT might attempt a bluff based on its `BluffFrequency`, even with a weaker hand.
    *   **Post-flop Strategy (Rounds 2, 3, 4)**:
        *   The BOT evaluates its hand using `getRank` (combining hole cards with community cards).
        *   If the hand is strong (e.g., Two Pair or better), the BOT will play more aggressively (bet/raise).
        *   Currently, the post-flop logic is simpler than pre-flop, defaulting to Check, Call, or Fold if no strong hand or bluff opportunity exists.
    *   **Default Actions**: If no specific strategy is triggered, the BOT defaults to `CHECK` (if possible), then `CALL` (if possible), otherwise `FOLD`.

This detailed flow ensures that the Texas Hold'em game progresses correctly, with intelligent BOTs adapting their play based on their defined personalities and the evolving game state.

A Python-based test client is provided in the `web-test2` directory. This client connects to the server, displays the game state in the console, and allows you to observe the BOTs in action.

### Setting Up and Running the Client

## Server Architecture: Concurrency and Multi-Game Support

The Texas Hold'em server is designed to handle multiple independent game tables and numerous client connections concurrently. This is achieved through a combination of Go's concurrency features (goroutines and channels) and careful state management.

### Key Components:

1.  **`main.go` (Server Setup & Routing):**
    *   Initializes the Gin web framework and defines API routes (e.g., `/ws`, `/state`, `/move`).
    *   The `/ws` endpoint is the entry point for WebSocket connections, where clients connect to a specific game table.
    *   The server maintains a `stateMap` (a `sync.Map`) that stores independent `GameState` instances for each active game table, enabling multiple games to run in parallel.

2.  **`websocket_handler.go` (WebSocket Management):**
    *   **`Hub` Struct**: This central component manages all active WebSocket clients connected to a particular game table. It includes:
        *   `clients`: A map of active `Client` connections.
        *   `register`, `unregister`: Channels for adding and removing clients.
        *   `broadcast`: A channel for sending messages to all connected clients.
    *   **`Hub.run(state *GameState)`**: A dedicated goroutine runs for each game table's `GameState`. This goroutine:
        *   Periodically (every 2 seconds) executes `state.RunGameLogic()`, advancing the game state for that specific table.
        *   Uses a `KeyedMutex` (`tableMutex`) to ensure thread-safe access and modification of the `GameState`, preventing race conditions.
        *   Broadcasts the updated `GameState` to all clients connected to that table, providing real-time game updates.
    *   **`serveWs` Function**: Handles the initial WebSocket handshake for new connections, creates a `Client` instance, and registers it with the appropriate `Hub`.
    *   **`readPump` and `writePump` Goroutines**: For each client, these goroutines manage the flow of messages to and from the WebSocket connection.

3.  **`gameLogic.go` (Game State Management):**
    *   This file defines the `GameState` struct, which encapsulates all the data for a single Texas Hold'em game (players, cards, pot, current round, etc.).
    *   Each `GameState` instance operates independently, ensuring that actions in one game do not affect others.

### How Multi-Game Concurrency is Achieved:

*   **Goroutines per Client**: Each WebSocket connection from a client is handled by its own goroutine, allowing the server to manage many simultaneous connections efficiently.
*   **Independent `GameState` Instances**: The server creates and manages separate `GameState` objects for each game table. This means that distinct games run in isolation from each other.
*   **Keyed Mutex for Table-Specific Locking**: The `tableMutex` ensures that while multiple games can run concurrently, access to a *single* game's `GameState` is synchronized. This prevents data corruption when multiple goroutines might try to modify the same game state (e.g., an API call and the periodic game logic update).
*   **Real-time Updates**: The `Hub.run` goroutine for each table periodically broadcasts the updated game state to its connected clients, ensuring they always see the current game situation.

This architecture allows the server to scale and host multiple Texas Hold'em games simultaneously, providing a robust and responsive experience for players.

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
