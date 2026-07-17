# Texas Hold'em Poker Server

A comprehensive Texas Hold'em poker server implementation in Go with sophisticated AI bots, real-time WebSocket communication, and extensive logging for debugging and game monitoring.

## 🎯 Features

- **Complete Texas Hold'em Implementation**: Full game logic following official poker rules
- **Comprehensive Phase Logging**: Detailed server logs for every game phase and action
- **Sophisticated AI Bots**: 8 unique bot personalities with different playing styles
- **Real-time WebSocket Communication**: Live game updates for all connected clients
- **Multi-table Support**: Concurrent games with independent state management
- **Extensive Unit Testing**: Comprehensive test coverage for all game phases
- **Professional Hand Evaluation**: Uses the `cardrank` library for accurate poker hand rankings

## 🚀 Quick Start

### Prerequisites

- Go 1.19 or later
- Git (for version control)

### Building and Running the Server

```bash
# Clone the repository
git clone <repository-url>
cd fujinet-game-system/texasholdem

# Navigate to server directory
cd server

# Install dependencies
go mod tidy

# Build the server
go build -o poker_server .

# Run the server
./poker_server
```

The server will start on **port 8080** and be ready to accept WebSocket connections.

### Debug Mode

For detailed request/response logging and game state debugging:

```bash
./poker_server -debug
```

Debug mode provides extensive logging including:
- Full JSON request/response data
- Detailed game state transitions
- Player action tracking
- Hand evaluation details

## 🎮 Game Flow & Logging

Our server provides comprehensive logging that tracks every aspect of Texas Hold'em gameplay:

### Game Initialization
```
=== TEXAS HOLD'EM: Starting new hand (Game #1) ===
PHASE: Pre-flop - Dealing hole cards to 4 players
CARDS: Dealt 2 hole cards to each player
```

### Blind Posting
```
BLINDS: Player2 posts small blind $5
BLINDS: Player3 posts big blind $10
BETTING: Pre-flop betting begins with Player4 (UTG)
```

### Phase Transitions
```
=== PHASE TRANSITION: Pre-flop betting complete ===
PHASE: Flop - Dealt 3 community cards [3 total]
BETTING: Flop betting begins with Player1

=== PHASE TRANSITION: Flop betting complete ===
PHASE: Turn - Dealt 1 community card [4 total]
BETTING: Turn betting begins with Player1

=== PHASE TRANSITION: Turn betting complete ===
PHASE: River - Dealt 1 community card [5 total]
BETTING: River betting begins with Player1
```

### Showdown & Winner Determination
```
=== PHASE TRANSITION: River betting complete ===
PHASE: Showdown - Determining winner with 5 community cards
POT: Final pot size is $120
SHOWDOWN: 2 players remain for showdown
WINNER: Player1 wins with Royal Flush - pot: $60 each
```

## 🤖 AI Bot System

The server features 8 unique bot personalities, each with distinct playing characteristics:

| Bot Name | Profile | VPIP | PFR | Bluff % | Description |
|----------|---------|------|-----|---------|-------------|
| **Clyd BOT** | Tight-Aggressive | 0.20 | 0.80 | 0.10 | Plays few hands but plays them aggressively |
| **Jim BOT** | Loose-Passive | 0.60 | 0.10 | 0.05 | Plays many hands but rarely raises |
| **Kirk BOT** | The Rock | 0.10 | 0.50 | 0.01 | Ultra-tight, only plays premium hands |
| **Hulk BOT** | The Maniac | 0.80 | 0.70 | 0.40 | Plays almost every hand aggressively |
| **Fry BOT** | Calling Station | 0.70 | 0.05 | 0.02 | Calls frequently, rarely folds or raises |
| **Meg BOT** | Balanced | 0.35 | 0.60 | 0.15 | Well-rounded, balanced approach |
| **Grif BOT** | The Bluffer | 0.40 | 0.50 | 0.50 | Frequent bluffer, unpredictable |
| **GPT BOT** | GTO Pro | 0.25 | 0.75 | 0.20 | Game theory optimal approach |

### Bot AI Features

- **Pre-flop Strategy**: Starting hand evaluation with position awareness
- **Post-flop Adaptation**: Dynamic strategy based on board texture and hand strength
- **Bluffing Logic**: Sophisticated bluffing based on personality and game state
- **Bankroll Management**: Smart all-in decisions and bet sizing

## 🧪 Testing & Validation

### Running the Test Suite

```bash
cd server

# Run all tests (unit, end-to-end engine, and HTTP multi-client integration)
go test -v

# Recommended full validation (as used in CI/verification)
go build ./... && go vet ./...
go test -race -count=1 ./...

# Run specific test suites
go test -v -run TestGetRank          # Hand evaluation (2..14 rank convention)
go test -v -run TestBotAI            # Bot AI (pre-flop and post-flop)
go test -v -run TestTexasHoldem      # Setup/blinds/positions driven by the engine
go test -v -run 'TestFullHand|TestChipConservation|TestAllIn'  # End-to-end engine tests
go test -v -run TestHTTP             # Multi-client HTTP integration tests
```

### Test Coverage

**Unit / engine tests** (drive `RunGameLogic` directly with fast timers, seeded RNG,
and rigged decks):

- ✅ Hand evaluation for all rankings, wheel straights, and K/A handling
- ✅ Full hand deal → showdown with correct phase order (Pre-flop → Flop → Turn → River → Showdown)
- ✅ Chip conservation across many consecutive hands (pot never mints or leaks chips)
- ✅ Dealer button rotation and blind positions, including heads-up rules
- ✅ Betting round termination: BB option after limps, raises re-opening action
- ✅ All-in handling with layered main/side pot distribution and board run-out
- ✅ Fold-out wins, human move-timeout auto-fold, bot AI validity fuzzing

**Multi-client HTTP integration tests** (`integration_test.go`): a small framework
(`simClient`) simulates real FujiNet clients against the actual Gin router via
`httptest`, speaking the compact 8-bit wire protocol (single-character JSON keys,
2-character move codes). Each client polls `GET /state?table=T&player=P` and
submits moves via `GET /move/<code>?table=T&player=P` according to a pluggable
policy (call-any, fold-unless-free, all-in). Covered:

- ✅ A single client joining and playing a complete hand over HTTP
- ✅ **Three concurrent clients** with mixed strategies playing multiple complete hands
- ✅ Wire contract: client always at `pl[0]`, opponent hole cards masked as `??`
  until showdown, `vm` moves only on your turn, `?hash=`/`z` change short-circuit
- ✅ All-in hands, leaving mid-hand, and running with the websocket hub ticker
  enabled (validated race-free with `go test -race`)

## 💻 Client Applications

### Go Test Client (`server/test-go-client`)

A fully playable Bubbletea TUI client that speaks the same HTTP protocol the
8-bit (Atari / Apple II) clients use - table picker, live table view, move
selection, spectator mode, and hash-based polling:

```bash
cd server/test-go-client
go run .                                   # interactive, table picker
go run . -table dev3 -name Andy            # join a dev table directly
go run . -auto                             # auto-play (check/call) mode
go run . -headless -table dev3 -hands 2    # no UI: scripted end-to-end run
```

### Python CLI Client (`test-cli-client`)

A Python observer client that renders the table state in the console:

```bash
cd test-cli-client
pip install -r requirements.txt
python client.py
```

## 🏗️ Architecture

### Core Components

1. **Game Logic (`gameLogic.go`)**
   - Complete Texas Hold'em rule implementation
   - Game state management
   - Player action processing
   - Hand evaluation using `cardrank` library

2. **WebSocket Handler (`websocket_handler.go`)**
   - Real-time client communication
   - Multi-table support with independent hubs
   - Concurrent connection management

3. **Main Server (`main.go`)**
   - HTTP/WebSocket routing
   - State management across multiple tables
   - API endpoints for game interaction

### Game State Structure

```go
type GameState struct {
    GamesPlayed     int           // Total games completed
    Round          int           // Current betting round (0-5)
    RoundName      string        // Human-readable phase name
    Pot            int           // Total pot size
    CurrentBet     int           // Current highest bet
    ActivePlayer   int           // Index of active player
    Players        []Player      // All players in the game
    CommunityCards []card        // Board cards
    Deck           []card        // Shuffled deck
    Winner         string        // Last hand winner
    // ... additional fields
}
```

### Concurrency & Multi-table Support

- **Goroutines per Client**: Each WebSocket connection handled independently
- **Table-specific Locking**: Keyed mutex ensures thread-safe game state access
- **Independent Game States**: Multiple tables run simultaneously without interference
- **Real-time Broadcasting**: Live updates sent to all connected clients

## 📡 API Endpoints

### WebSocket Connection
- **`/ws`**: One-way broadcast of observer game state every 2 seconds

### HTTP Endpoints (the gameplay API)
- **`GET/POST /state?table=T&player=P`**: Join (implicitly) and retrieve the game
  state from player P's perspective; advances the game. Pass `hash=<z>` from a
  previous response to receive `"1"` when nothing changed.
- **`GET/POST /move/<code>?table=T&player=P`**: Submit a move when it is your
  turn. `<code>` is a 2-character move code from the state's `vm` array
  (`FO`, `CH`, `CA`, `BL`, `BH`, `RL`, `RH`, `AI`); amounts are computed server-side.
  See `server/readme.md` for the full wire protocol specification.
- **`GET /view?table=T`**: Observer snapshot without advancing the game
- **`GET/POST /leave?table=T&player=P`**: Leave the table
- **`GET /tables`** (`?dev=1` for dev tables): List tables

## 🔧 Configuration

### Environment Variables

- `PORT`: Server port (default: 8080)
- `DEBUG`: Enable debug logging (default: false)

### Game Constants

```go
const (
    STARTING_PURSE     = 1000    // Starting chips per player
    SB                = 5        // Small blind amount
    BB                = 10       // Big blind amount
    ENDGAME_TIME_LIMIT = 10s     // Delay between hands
)
```

## 📊 Logging Levels

The server provides multiple logging levels for different purposes:

- **PHASE**: Game phase transitions and major events
- **CARDS**: Card dealing and deck management
- **BLINDS**: Blind posting and amounts
- **BETTING**: Betting rounds and player actions
- **POT**: Pot size changes and management
- **WINNER**: Game results and payouts
- **SHOWDOWN**: Hand comparisons and evaluations
- **DEBUG**: Detailed internal state information

## 🐛 Debugging

### Common Issues

1. **Connection Problems**: Ensure server is running on correct port
2. **Game State Sync**: Check WebSocket connection stability
3. **Bot Behavior**: Review bot profiles and AI logic
4. **Hand Evaluation**: Verify card representations and rankings

### Debug Tools

- Use `-debug` flag for verbose logging
- Monitor WebSocket traffic with browser dev tools
- Run unit tests to validate game logic
- Check server logs for error messages

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add comprehensive tests for new features
4. Ensure all existing tests pass
5. Submit a pull request with detailed description

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🎯 Future Enhancements

- [ ] Tournament mode support
- [ ] Player statistics tracking
- [ ] Advanced bot AI with machine learning
- [ ] Mobile client applications
- [ ] Database persistence for game history
- [ ] Multi-language support
- [ ] Cash game vs tournament modes
- [ ] Player chat functionality

---

**Ready to play Texas Hold'em?** Start the server and connect with your favorite client to experience professional-grade poker gameplay with intelligent AI opponents!
