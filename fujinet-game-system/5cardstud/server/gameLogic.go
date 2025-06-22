package main

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cardrank/cardrank"
	hashstructure "github.com/mitchellh/hashstructure/v2"
	"golang.org/x/exp/slices"
)

/*
5 Card Stud Rules below to serve as guideline.

The logic to support below is not all implemented, and will be done as time allows.

Rules -  Assume Limit betting: Anti 1, Bringin 2,  Low 5, High 10
Suit Rank (for comparing first to act): S,H,D,C

Winning hands - tied hands split the pot, remainder is discarded

1. All players anti (e.g.) 1
2. First round
  - Player with lowest card goes first, with a mandatory bring in of 2. Option to make full bet (5)
	- Play moves Clockwise
	- Subsequent player can call 2 (assuming no full bet yet) or full bet 5
	- Subsequent Raises are inrecements of the highest bet (5 first round, or of the highest bet in later rounds)
	- Raises capped at 3 (e.g. max 20 = 5 + 3*5 round 1)
3. Remaining rounds
	- Player with highest ranked visible hand goes first
	- 3rd Street - 5, or if a pair is showing: 10, so max is 5*4 20 or 10*4 40
	- 4th street+ - 10
*/

const (
	ANTE        = 1
	BRINGIN     = 2 // Not used in Texas Hold'em
	LOW         = 5 // Not used in Texas Hold'em
	HIGH        = 10 // Not used in Texas Hold'em
	SB          = 5 // Small Blind
	BB          = 10 // Big Blind
	STARTING_PURSE = 1000
)
const MOVE_TIME_GRACE_SECONDS = 4
const BOT_TIME_LIMIT = time.Second * time.Duration(3)
const PLAYER_TIME_LIMIT = time.Second * time.Duration(39)
const ENDGAME_TIME_LIMIT = time.Second * time.Duration(12)
const NEW_ROUND_FIRST_PLAYER_BUFFER = time.Second * time.Duration(1)

// Drop players who do not make a move in 5 minutes
const PLAYER_PING_TIMEOUT = time.Minute * time.Duration(-5)

const WAITING_MESSAGE = "Waiting for more players"

var suitLookup = []string{"C", "D", "H", "S"}
var valueLookup = []string{"", "", "2", "3", "4", "5", "6", "7", "8", "9", "T", "J", "Q", "K", "A"}
var moveLookup = map[string]string{
	"FO": "FOLD",
	"CH": "CHECK",
	"BB": "POST",
	"BL": "BET", // BET LOW (e.g. 5 of 5/10, or 2 of 2/5 first round)
	"BH": "BET", // BET HIGH (e.g. 10)
	"CA": "CALL",
	"RA": "RAISE",
	"ALLIN": "ALLIN",
}

var botNames = []string{"Clyd", "Jim", "Kirk", "Hulk", "Fry", "Meg", "Grif", "GPT"}

// For simplicity on the 8bit side (using switch statement), using a single character for each key.
// DOUBLE CHECK that letter isn't already in use on the object!
// Double characters are used for the list objects (validMoves and players)

type validMove struct {
	Move string `json:"m"`
	Name string `json:"n"`
}

type card struct {
	Rank int `json:"Rank"`
	Suit int `json:"Suit"`
}

type Status int64

const (
	STATUS_WAITING Status = 0
	STATUS_PLAYING Status = 1
	STATUS_FOLDED  Status = 2
	STATUS_LEFT    Status = 3
)

type Player struct {
	Name   string `json:"Name"`
	Status Status `json:"Status"`
	Bet    int    `json:"Bet"`
	Move   string `json:"Move"`
	Purse  int    `json:"Purse"`
	Hand   []card `json:"Hand"`
	IsHuman bool   `json:"IsHuman"`

	// Internal
	isBot    bool
	lastPing time.Time
}

type GameState struct {
	GamesPlayed  int    `json:"gamesPlayed"`
	TableId      string `json:"tableId"`
	// External (JSON)
	LastResult   string      `json:"LastResult"`
	Winner       string      `json:"Winner"`
	Round        int         `json:"Round"`
	RoundName    string      `json:"roundName"`
	Pot          int         `json:"Pot"`
	ActivePlayer int         `json:"ActivePlayer"`
	MoveTime     int         `json:"MoveTime"`
	Viewing      int         `json:"Viewing"`
	ValidMoves   []validMove `json:"ValidMoves"`
	Players      []Player    `json:"Players"`
	CommunityCards []card `json:"CommunityCards"`
	CurrentBet int `json:"currentBet"`

	// Internal
	deck          []card
	deckIndex     int
	currentBet    int
	gameOver      bool
	clientPlayer  int
	wonByFolds    bool
	moveExpires   time.Time
	serverName    string
	raiseCount    int
	raiseAmount   int
	registerLobby bool
	hash          string //   `json:"z"` // external later
}

// Used to send a list of available tables
type GameTable struct {
	Table      string `json:"t"`
	Name       string `json:"n"`
	CurPlayers int    `json:"p"`
	MaxPlayers int    `json:"m"`
}

func initializeGameServer() {

	// Append BOT to botNames array
	for i := 0; i < len(botNames); i++ {
		botNames[i] = botNames[i] + " BOT"
	}
}

func createGameState(playerCount int, registerLobby bool) *GameState {

	deck := []card{}

	// Create deck of 52 cards
	for suit := 0; suit < 4; suit++ {
		for value := 2; value < 15; value++ {
			card := card{Rank: value, Suit: suit}
			deck = append(deck, card)
		}
	}

	state := GameState{}
	state.deck = deck
	state.Round = 0
	state.ActivePlayer = -1
	state.registerLobby = registerLobby
	state.CommunityCards = []card{}

	// Pre-populate player pool with bots
	for i := 0; i < playerCount; i++ {
		state.addPlayer(botNames[i], true)
	}

	if playerCount < 2 {
		state.LastResult = WAITING_MESSAGE
	}

	return &state
}

func (state *GameState) newRound() {

	// Drop any players that left last round
	state.dropInactivePlayers(true, false)

	// Check if multiple players are still playing
	if state.Round == 0 {
		state.GamesPlayed++
	} else if state.Round > 0 {
		playersLeft := 0
		for _, player := range state.Players {
			if player.Status == STATUS_PLAYING {
				playersLeft++
			}
		}

		if playersLeft < 2 {
			state.endGame(false)
			return
		}
	} else {
		if len(state.Players) < 2 {
			return
		}
	}

	state.Round++

	// Clear community cards and pot at start of a new hand
	if state.Round == 1 {
		state.Winner = ""
		state.RoundName = "Pre-flop"
		state.Pot = 0
		state.gameOver = false
		state.CommunityCards = []card{}
	}

	// Reset players for this round
	for i := 0; i < len(state.Players); i++ {

		// Get pointer to player
		player := &state.Players[i]

		if state.Round > 1 {
			// If not the first round, add any bets into the pot
			state.Pot += player.Bet
		} else {

			// First round of a new game

			// A bot will leave if it has under 25 chips, another will take their place
			if player.isBot && player.Purse < 25 {
				player.Purse = STARTING_PURSE
				for j := 0; j < len(botNames); j++ {
					botNameUsed := false
					for k := 0; k < len(state.Players); k++ {
						if strings.EqualFold(botNames[j], state.Players[k].Name) {
							botNameUsed = true
							break
						}
					}
					if !botNameUsed {
						player.Name = botNames[j]
						break
					}
				}
			}

			// Reset player status
			player.Status = STATUS_PLAYING
			player.Hand = []card{} // Clear cards for new hand
			player.Bet = 0 // Clear bets for new hand
		}

		// Reset player's last move/bet for this round
		player.Move = ""
		player.Bet = 0
	}

	state.currentBet = 0
	state.raiseCount = 0
	state.raiseAmount = 0

	// First round of a new game? Shuffle the cards and deal an extra card
	if state.Round == 1 {
			// Shuffle the deck 7 times :)
		for shuffle := 0; shuffle < 7; shuffle++ {
			rand.Shuffle(len(state.deck), func(i, j int) { state.deck[i], state.deck[j] = state.deck[j], state.deck[i] })
		}
		state.deckIndex = 0
		if state.LastResult == WAITING_MESSAGE {
			state.LastResult = ""
		}

		// Deal 2 hole cards to each player
		for cardNum := 0; cardNum < 2; cardNum++ {
			for i := 0; i < len(state.Players); i++ {
				player := &state.Players[i]
				if player.Status == STATUS_PLAYING {
					state.Players[i].Hand = []card{state.deck[state.deckIndex], state.deck[state.deckIndex+1]}
					state.deckIndex += 2
				}
			}
		}

		// Determine dealer, small blind, big blind
		// For simplicity, let's assume the first active player is the dealer for now.
		// This will need more robust logic later for button rotation.
		dealerIndex := -1
		for i, p := range state.Players {
			if p.Status == STATUS_PLAYING {
				dealerIndex = i
				break
			}
		}

		if dealerIndex != -1 {
			smallBlindIndex := (dealerIndex + 1) % len(state.Players)
			bigBlindIndex := (dealerIndex + 2) % len(state.Players)

			// Post Small Blind
			state.Players[smallBlindIndex].Purse -= SB
			state.Players[smallBlindIndex].Bet += SB
			state.Pot += SB
			state.Players[smallBlindIndex].Move = fmt.Sprintf("POST %d", SB)

			// Post Big Blind
			state.Players[bigBlindIndex].Purse -= BB
			state.Players[bigBlindIndex].Bet += BB
			state.Pot += BB
			state.Players[bigBlindIndex].Move = fmt.Sprintf("POST %d", BB)
			state.currentBet = BB

			// First player to act is left of Big Blind
			state.ActivePlayer = (bigBlindIndex + 1) % len(state.Players)
		} else {
			state.ActivePlayer = 0 // Fallback if no active players
		}
	}
	state.resetPlayerTimer(true)
}

func (state *GameState) getPlayerWithBestVisibleHand(highHand bool) int {

	ranks := [][]int{}

	for i := 0; i < len(state.Players); i++ {
		player := &state.Players[i]
		if player.Status == STATUS_PLAYING {
			rank := getRank(player.Hand, state.CommunityCards)

			// Add player number to start of rank to hold on to when sorting
			rank = append([]int{i}, rank...)
			ranks = append(ranks, rank)
		}
	}

	// Sort the ranks by value first, the breaking tie by suit
	sort.SliceStable(ranks, func(i, j int) bool {
		for k := 1; k < 9; k++ {
			if ranks[i][k] != ranks[j][k] {
				return ranks[i][k] < ranks[j][k]
			}
		}
		return false
	})

	// Return player with highest (or lowest) hand
	result := 0
	if highHand {
		result = ranks[0][0]
	} else {
		result = ranks[len(ranks)-1][0]
	}

	// If something goes amiss, just select the first player
	if result < 0 {
		result = 0
	}
	return result
}

func (state *GameState) dealCommunityCards(count int) {
	for i := 0; i < count; i++ {
		state.CommunityCards = append(state.CommunityCards, state.deck[state.deckIndex])
		state.deckIndex++
	}
}

func (state *GameState) resetPlayersForNewBettingRound() {
	state.currentBet = 0
	state.raiseCount = 0
	state.raiseAmount = 0
	for i := 0; i < len(state.Players); i++ {
		player := &state.Players[i]
		// Only reset status for players who are still in the hand (not folded or left)
		if player.Status != STATUS_FOLDED && player.Status != STATUS_LEFT {
			player.Status = STATUS_PLAYING // Ensure player is set to playing for the new round
			player.Bet = 0
			player.Move = ""
		}
	}
	state.nextValidPlayer() // Set the active player for the new round
}

func (state *GameState) addPlayer(playerName string, isBot bool) {

	newPlayer := Player{
		Name:   playerName,
		Status: 0,
		Purse:  STARTING_PURSE,
		Hand:   []card{},
		isBot:  isBot,
	}

	state.Players = append(state.Players, newPlayer)
}

func (state *GameState) setClientPlayerByName(playerName string) {
	// If no player name was passed, simply return. This is an anonymous viewer.
	if len(playerName) == 0 {
		state.clientPlayer = -1
		return
	}
	state.clientPlayer = slices.IndexFunc(state.Players, func(p Player) bool { return strings.EqualFold(p.Name, playerName) })

	// If a new player is joining, remove any old players that timed out to make space
	if state.clientPlayer < 0 {
		// Drop any players that left to make space
		state.dropInactivePlayers(false, true)
	}

	// Add new player if there is room
	if state.clientPlayer < 0 && len(state.Players) < 8 {
		state.addPlayer(playerName, false)
		state.clientPlayer = len(state.Players) - 1

		// Set the ping for this player so they are counted as active when updating the lobby
		state.playerPing()

		// Update the lobby with the new state (new player joined)
		state.updateLobby()
	}

	// Extra logic if a player is requesting
	if state.clientPlayer > 0 {

		// In case a player returns while they are still in the "LEFT" status (before the current game ended), add them back in as waiting
		if state.Players[state.clientPlayer].Status == STATUS_LEFT {
			state.Players[state.clientPlayer].Status = STATUS_WAITING
		}
	}
}

func (state *GameState) endGame(abortGame bool) {
	// The next request for /state will start a new game

	// Hand rank details
	// Rank: SF, 4K, FH, F, S, 3K, 2P, 1P, HC

	state.gameOver = true
	state.ActivePlayer = -1
	state.Round = 5

	remainingPlayers := []int{}
	pockets := [][]cardrank.Card{}

	for index, player := range state.Players {
		state.Pot += player.Bet
		if !abortGame && player.Status == STATUS_PLAYING {
			remainingPlayers = append(remainingPlayers, index)
			hand := ""
			// Loop through and build hand string
			for _, card := range player.Hand {
				hand += valueLookup[card.Rank] + suitLookup[card.Suit]
			}
			pockets = append(pockets, cardrank.Must(hand))
		}
	}

	evs := cardrank.StudFive.EvalPockets(pockets, nil)
	order, pivot := cardrank.Order(evs, false)

	if pivot == 0 {
		// If nobody won, the game was aborted. Display the waiting message if this
		// server does not contains bots.
		humanAvailSlots, _ := state.getHumanPlayerCountInfo()
		if humanAvailSlots == 8 {
			state.LastResult = WAITING_MESSAGE
			state.moveExpires = time.Now().Add(ENDGAME_TIME_LIMIT)
		} else {
			state.moveExpires = time.Now()
		}
		return
	}

	// Int divide, so "house" takes remainder
	perPlayerWinnings := state.Pot / pivot

	result := ""

	for i := 0; i < pivot; i++ {
		player := &state.Players[remainingPlayers[order[i]]]

		// Award winnings to player's purse
		player.Purse += int(perPlayerWinnings)

		// Add player's name to result
		if result != "" {
			result += " and "
		}
		result += player.Name
	}

	if len(remainingPlayers) > 1 {
		state.wonByFolds = false
		result += strings.Join(strings.Split(strings.Split(fmt.Sprintf(" won with %s", evs[order[0]]), " [")[0], ",")[0:2], ",")
		result = strings.ReplaceAll(result, "kickers", "kicker")
	} else {
		state.wonByFolds = true
		result += " won by default"
	}
	state.Winner = result
	state.LastResult = result

	state.moveExpires = time.Now().Add(ENDGAME_TIME_LIMIT)

	log.Println(result)
}

// Emulates simplified player/logic for 5 card stud
func (state *GameState) RunGameLogic() {
	log.Printf("DEBUG: RunGameLogic called. ActivePlayer: %d, Round: %d\n", state.ActivePlayer, state.Round)

	// We can't play a game until there are at least 2 players
	if len(state.Players) < 2 {
		// Reset the round to 0 so the client knows there is no active game being run
		state.Round = 0
		state.Pot = 0
		state.ActivePlayer = -1
		return
	}

	// Very first call of state? Initialize first round but do not play for any BOTs
	if state.Round == 0 {
		state.newRound()
		return
	}

	//isHumanPlayer := state.ActivePlayer == state.clientPlayer

	if state.gameOver {

		// Create a new game if the end game delay is past
		if int(time.Until(state.moveExpires).Seconds()) < 0 {
			state.dropInactivePlayers(false, false)
			state.Round = 0
			state.Pot = 0
			state.gameOver = false
			state.newRound()
		}
		return
	}

	// Check if only one player is left
	playersLeft := 0
	for _, player := range state.Players {
		if player.Status == STATUS_PLAYING {
			playersLeft++
		}
	}

	// If only one player is left, just end the game now
	if playersLeft == 1 {
		state.endGame(false)
		return
	}

	// Check if we should start the next round. One of the following must be true
	// 1. We got back to the player who made the most recent bet/raise
	// 2. There were checks/folds around the table


	// Return if the move timer has not expired
	// Check timer if no active player, or the active player hasn't already left
	if state.ActivePlayer == -1 || state.Players[state.ActivePlayer].Status != STATUS_LEFT {
		moveTimeRemaining := int(time.Until(state.moveExpires).Seconds())
		if moveTimeRemaining > 0 {
			return
		}
	}

	// If there is no active player, we are done
	if state.ActivePlayer < 0 {
		return
	}

	// Edge cases
	// - player leaves when it is their move - skip over them
	// - player's turn but they are waiting (out of this hand)
	if state.Players[state.ActivePlayer].Status == STATUS_LEFT ||
		state.Players[state.ActivePlayer].Status == STATUS_WAITING {
		state.nextValidPlayer()
		return
	}

	// Force a move for this player or BOT if they are in the game and have not folded
	log.Printf("DEBUG: Player %d Status: %d\n", state.ActivePlayer, state.Players[state.ActivePlayer].Status)
	if state.Players[state.ActivePlayer].Status == STATUS_PLAYING {
		log.Printf("DEBUG: Inside STATUS_PLAYING block for Player %d\n", state.ActivePlayer)
		cards := state.Players[state.ActivePlayer].Hand
		moves := state.getValidMoves()

		// Default to FOLD
		choice := 0

		// Never fold if CHECK is an option. This applies to forced player moves as well as bots
		if len(moves) > 1 && moves[1].Move == "CH" {
			choice = 1
		}

		// If this is a bot, pick the best move using some simple logic (sometimes random)
		if state.Players[state.ActivePlayer].isBot {
			// rank := getRank(cards, state.CommunityCards)
			// bestHandRank := rank[0] // Assuming getRank returns a single integer rank from cardrank

			// Simple AI logic for Texas Hold'em
			// This is a very basic AI and can be greatly improved.

			// Pre-flop strategy (Round 1)
			if state.Round == 1 {
				// Check for strong starting hands (e.g., high pairs, suited connectors, high cards)
				// This part needs more detailed logic based on hole cards only.
				// For now, let's just make it play somewhat aggressively with good cards.
				if (cards[0].Rank == cards[1].Rank && cards[0].Rank >= 8) || // Pairs 8s+
					(cards[0].Rank >= 10 && cards[1].Rank >= 10) || // Two high cards (TJ+)
					(cards[0].Suit == cards[1].Suit && cards[0].Rank >= 7 && cards[1].Rank >= 7) { // Suited connectors 7+
					// Try to raise or call
					if len(moves) > 2 && rand.Intn(2) == 0 { // 50% chance to raise if possible
						choice = len(moves) - 1 // Take the highest available bet/raise
					} else if len(moves) > 1 { // Otherwise call or check
						choice = 1
					}
				} else if state.currentBet == 0 && slices.ContainsFunc(moves, func(m validMove) bool { return m.Move == "CH" }) { // If no bet, check
					choice = slices.IndexFunc(moves, func(m validMove) bool { return m.Move == "CH" })
				} else {
					choice = 0 // Fold
				}
			} else { // Rounds 2, 3, 4 (Flop, Turn, River)
				// Simple post-flop strategy: Check if possible, otherwise call if affordable, else fold.
				if state.currentBet == 0 && slices.ContainsFunc(moves, func(m validMove) bool { return m.Move == "CH" }) {
					choice = slices.IndexFunc(moves, func(m validMove) bool { return m.Move == "CH" })
				} else if state.currentBet > 0 && len(moves) > 1 && slices.ContainsFunc(moves, func(m validMove) bool { return strings.HasPrefix(m.Move, "CALL") }) { // Check if CALL is an option
					choice = slices.IndexFunc(moves, func(m validMove) bool { return strings.HasPrefix(m.Move, "CALL") }) // Find the index of CALL
				} else {
					choice = 0 // Fold
				}
			}
		}
		// Apply the chosen move
		log.Printf("DEBUG: Player %d chose move: %s (choice index: %d)\n", state.ActivePlayer, moves[choice].Move, choice)
		state.performMove(moves[choice].Move, true)

		// Check for round advancement after a player has made a move
		allPlayersMovedAndMatchedBet := true
		for _, player := range state.Players {
			if player.Status == STATUS_PLAYING {
				if player.Move == "" {
					allPlayersMovedAndMatchedBet = false
					break
				}
				if player.Bet < state.currentBet {
					allPlayersMovedAndMatchedBet = false
					break
				}
			}
		}

		if allPlayersMovedAndMatchedBet || state.wonByFolds {
			log.Printf("DEBUG: All players moved and matched bet or won by folds. Advancing round.\n")
			if state.Round == 1 && !state.wonByFolds { // Pre-flop -> Flop
				state.dealCommunityCards(3)
				state.Round++
				state.RoundName = "Flop"
				state.resetPlayersForNewBettingRound()
			} else if state.Round == 2 && !state.wonByFolds { // Flop -> Turn
				state.dealCommunityCards(1)
				state.Round++
				state.RoundName = "Turn"
				state.resetPlayersForNewBettingRound()
			} else if state.Round == 3 && !state.wonByFolds { // Turn -> River
				state.dealCommunityCards(1)
				state.Round++
				state.RoundName = "River"
				state.resetPlayersForNewBettingRound()
			} else { // River -> Showdown/End Game
				state.Round++
				state.RoundName = "Showdown"
				state.endGame(false)
			}
			return // Return after advancing round
		}
	}
}

	// Drop players that left or have not pinged within the expected timeout
func (state *GameState) dropInactivePlayers(inMiddleOfGame bool, dropForNewPlayer bool) {
	cutoff := time.Now().Add(PLAYER_PING_TIMEOUT)
	players := []Player{}
	currentPlayerName := ""
	if state.clientPlayer > -1 {
		currentPlayerName = state.Players[state.clientPlayer].Name
	}

	for _, player := range state.Players {
		if len(state.Players) > 0 && player.Status != STATUS_LEFT && (inMiddleOfGame || player.isBot || player.lastPing.Compare(cutoff) > 0) {
			players = append(players, player)
		}
	}

	// If one player is left, don't drop them within the round, let the normal game end take care of it
	if inMiddleOfGame && len(players) == 1 {
		return
	}

	// Store if players were dropped, before updating the state player array
	playersWereDropped := len(state.Players) != len(players)

	if playersWereDropped {
		state.Players = players
	}

	// If a new player is joining, don't bother updating anything else
	if dropForNewPlayer {
		return
	}

	// Update the client player index in case it changed due to players being dropped
	if len(players) > 0 {
		state.clientPlayer = slices.IndexFunc(players, func(p Player) bool { return strings.EqualFold(p.Name, currentPlayerName) })
	}

	// If only one player is left, we are waiting for more
	if len(state.Players) < 2 {
		state.LastResult = WAITING_MESSAGE
	}

	// If any player state changed, update the lobby
	if playersWereDropped {
		state.updateLobby()
	}

}

func (state *GameState) clientLeave() {
	if state.clientPlayer < 0 {
		return
	}
	player := &state.Players[state.clientPlayer]

	player.Status = STATUS_LEFT
	player.Move = "LEFT"

	// Check if no human players are playing. If so, end the game
	playersLeft := 0
	for _, player := range state.Players {
		if player.Status == STATUS_PLAYING && !player.isBot {
			playersLeft++
		}
	}

	// If the last player dropped, stop the game and update the lobby
	if playersLeft == 0 {
		state.endGame(true)
		state.dropInactivePlayers(false, false)
		return
	}
}

// Update player's ping timestamp. If a player doesn't ping in a certain amount of time, they will be dropped from the server.
func (state *GameState) playerPing() {
	if state.clientPlayer < 0 || state.clientPlayer >= len(state.Players) {
		return // Not in a valid player context to ping
	}
	state.Players[state.clientPlayer].lastPing = time.Now()
}

// Performs the requested move for the active player, and returns true if successful
func (state *GameState) performMove(move string, internalCall ...bool) bool {
	log.Printf("DEBUG: performMove called with move: %s for player %d\n", move, state.ActivePlayer)

	if len(internalCall) == 0 || !internalCall[0] {
		state.playerPing()
	}

	// Get pointer to player
	player := &state.Players[state.ActivePlayer]

	// Sanity check if player is still in the game. Unless there is a bug, they should never be active if their status is != PLAYING
	if player.Status != STATUS_PLAYING {
		return false
	}

	// Only perform move if it is a valid move for this player
	if !slices.ContainsFunc(state.getValidMoves(), func(m validMove) bool { return m.Move == move }) {
		return false
	}

	if move == "FO" { // FOLD
		player.Status = STATUS_FOLDED
		player.Move = moveLookup[move] // Set player.Move for FOLD
	} else if move == "CH" { // CHECK
		player.Move = moveLookup[move] // Set player.Move for CHECK
	} else if move == "ALLIN" { // ALLIN
		betAmount := player.Purse
		player.Bet += betAmount
		player.Purse = 0
		state.Pot += betAmount
		if player.Bet > state.currentBet {
			state.currentBet = player.Bet
		}
		player.Move = moveLookup[move] // Set player.Move for ALLIN
	} else { // BET, CALL, RAISE
		betAmount := 0
		// Assuming the move string contains the amount for BET/RAISE/CALL for simplicity for now
		// In a real implementation, this would be derived from the move type and current bet
		// For now, let's assume 'move' is like 'BET 100' or 'CALL 50'
		parts := strings.Split(move, " ")
		if len(parts) > 1 {
			betAmount, _ = strconv.Atoi(parts[1])
		}

		if betAmount > player.Purse {
			betAmount = player.Purse // Cannot bet more than available purse
		}
		player.Move = move // For BET/CALL/RAISE, the move string itself is the full move

		player.Purse -= betAmount
		state.Pot += betAmount
		player.Bet += betAmount

		if player.Bet > state.currentBet {
			state.currentBet = player.Bet
		}

		// For RAISE, update raiseCount and raiseAmount if needed
		if strings.Contains(move, "RAISE") {
			state.raiseCount++
			// This needs more sophisticated logic for actual raise amounts in NLHE
			// For now, we'll just track that a raise happened.
		}
	}

	// Assign the move string directly, or use lookup for simple moves
	if move == "FO" || move == "CH" || move == "ALLIN" {
		player.Move = moveLookup[move]
	} else {
		player.Move = move
	}
	state.nextValidPlayer()

	return true
}

func (state *GameState) resetPlayerTimer(newRound bool) {
	timeLimit := PLAYER_TIME_LIMIT

	if state.Players[state.ActivePlayer].isBot {
		timeLimit = BOT_TIME_LIMIT
	}

	if newRound {
		timeLimit += NEW_ROUND_FIRST_PLAYER_BUFFER
	}

	state.moveExpires = time.Now().Add(timeLimit)
}

func (state *GameState) nextValidPlayer() {
	// Move to next player
	state.ActivePlayer = (state.ActivePlayer + 1) % len(state.Players)

	// Skip over player if not in this game (joined late / folded)
	for state.Players[state.ActivePlayer].Status != STATUS_PLAYING {
		state.ActivePlayer = (state.ActivePlayer + 1) % len(state.Players)
	}
	state.resetPlayerTimer(false)
}

func (state *GameState) getValidMoves() []validMove {
	moves := []validMove{}

	player := state.Players[state.ActivePlayer]

	// Always allow fold
	moves = append(moves, validMove{Move: "FO", Name: "Fold"})

	// Calculate amount needed to call
	callAmount := state.currentBet - player.Bet

	// If currentBet is 0, player can CHECK or BET
	if state.currentBet == 0 {
		moves = append(moves, validMove{Move: "CH", Name: "Check"})
		// Player can bet any amount from BB to their purse
		if player.Purse >= BB {
			moves = append(moves, validMove{Move: fmt.Sprintf("BET %d", BB), Name: fmt.Sprintf("Bet %d", BB)}) // Min bet is BB
			// For simplicity, let's also add a common bet size like 2*BB or half pot
			if player.Purse >= 2*BB {
				moves = append(moves, validMove{Move: fmt.Sprintf("BET %d", 2*BB), Name: fmt.Sprintf("Bet %d", 2*BB)})
			}
		}
	} else { // A bet has been made
		// Player can CALL
		if player.Purse >= callAmount {
			moves = append(moves, validMove{Move: fmt.Sprintf("CALL %d", callAmount), Name: fmt.Sprintf("Call %d", callAmount)})
		}

		// Player can RAISE
		// Minimum raise is the size of the last bet/raise, or BB if no previous raise
		minRaiseAmount := BB
		// This needs to be more robust, tracking the actual last raise amount
		// For now, let's assume min raise is currentBet if it's not 0, otherwise BB
		if state.currentBet > 0 {
			minRaiseAmount = state.currentBet
		}

		raisableAmount := callAmount + minRaiseAmount
		if player.Purse >= raisableAmount {
			moves = append(moves, validMove{Move: fmt.Sprintf("RAISE %d", raisableAmount), Name: fmt.Sprintf("Raise %d", raisableAmount)})
			// Add another raise option, e.g., 2*minRaiseAmount
			if player.Purse >= callAmount + 2*minRaiseAmount {
				moves = append(moves, validMove{Move: fmt.Sprintf("RAISE %d", callAmount + 2*minRaiseAmount), Name: fmt.Sprintf("Raise %d", callAmount + 2*minRaiseAmount)})
			}
		}
	}

	// Always allow ALLIN if player has chips
	if player.Purse > 0 {
		moves = append(moves, validMove{Move: "ALLIN", Name: fmt.Sprintf("All-in %d", player.Purse)})
	}

	return moves
}

// Creates a copy of the state and modifies it to be from the
// perspective of this client (e.g. player array, visible cards)
func (state *GameState) createClientState() *GameState {

	stateCopy := *state

	setActivePlayer := false

	// Check if:
	// 1. The game is over,
	// 2. Only one player is left (waiting for another player to join)
	// 3. We are at the end of a round, where the active player has moved
	// This lets the client perform end of round/game tasks/animation
	if state.gameOver ||
		len(stateCopy.Players) < 2 ||
		(stateCopy.ActivePlayer > -1 && ((state.currentBet > 0 && state.Players[state.ActivePlayer].Bet == state.currentBet) ||
			(state.currentBet == 0 && state.Players[state.ActivePlayer].Move != ""))) {
		stateCopy.ActivePlayer = -1
		setActivePlayer = true
	}

	// Now, store a copy of state players, then loop
	// through and add to the state copy, starting
	// with this player first

	statePlayers := stateCopy.Players
	stateCopy.Players = []Player{}

	// When on observer is viewing the game, the clientPlayer will be -1, so just start at 0
	// Also, set flag to let client know they are not actively part of the game
	start := state.clientPlayer
	if start < 0 {
		start = 0
		stateCopy.Viewing = 1
	} else {
		stateCopy.Viewing = 0
	}

	// Loop through each player and create the hand, starting at this player, so all clients see the same order regardless of starting player
	for i := start; i < start+len(statePlayers); i++ {

		// Wrap around to beginning of playar array when needed
		playerIndex := i % len(statePlayers)

		// Update the ActivePlayer to be client relative
		if !setActivePlayer && playerIndex == stateCopy.ActivePlayer {
			setActivePlayer = true
			stateCopy.ActivePlayer = i - start
		}

		player := statePlayers[playerIndex]

		// If the client is a player (not an observer) and this is not the client's player, mask the hand.
		if state.clientPlayer != -1 && playerIndex != state.clientPlayer {
			if len(player.Hand) > 0 {
				player.Hand = make([]card, len(player.Hand))
			}
		}

		// Add this player to the copy of the state going out
		stateCopy.Players = append(stateCopy.Players, player)
	}

	// Add community cards to the client state
	stateCopy.CommunityCards = state.CommunityCards

	// Determine valid moves for this player (if their turn)
	if stateCopy.ActivePlayer == 0 {
		stateCopy.ValidMoves = state.getValidMoves()
	}

	// Determine the move time left. Reduce the number by the grace period, to allow for plenty of time for a response to be sent back and accepted
	stateCopy.MoveTime = int(time.Until(stateCopy.moveExpires).Seconds())

	if stateCopy.ActivePlayer > -1 {
		stateCopy.MoveTime -= MOVE_TIME_GRACE_SECONDS
	}

	// No need to send move time if the calling player isn't the active player
	if stateCopy.MoveTime < 0 || stateCopy.ActivePlayer != 0 {
		stateCopy.MoveTime = 0
	}

	// Compute hash - this will be compared with an incoming hash. If the same, the entire state does not
	// need to be sent back. This speeds up checks for change in state
	stateCopy.hash = "0"
	hash, _ := hashstructure.Hash(stateCopy, hashstructure.FormatV2, nil)
	stateCopy.hash = fmt.Sprintf("%d", hash)

	return &stateCopy
}

func (state *GameState) updateLobby() {
	if !state.registerLobby {
		return
	}

	humanPlayerSlots, humanPlayerCount := state.getHumanPlayerCountInfo()

	// Send the total human slots / players to the Lobby
	sendStateToLobby(humanPlayerSlots, humanPlayerCount, true, state.serverName, "?table="+state.TableId)
}

// Return number of active human players in the table, for the lobby
func (state *GameState) getHumanPlayerCountInfo() (int, int) {
	humanAvailSlots := 8
	humanPlayerCount := 0
	cutoff := time.Now().Add(PLAYER_PING_TIMEOUT)

	for _, player := range state.Players {
		if player.isBot {
			humanAvailSlots--
		} else if player.Status != STATUS_LEFT && player.lastPing.Compare(cutoff) > 0 {
			humanPlayerCount++
		}
	}
	return humanAvailSlots, humanPlayerCount
}

// Ranks hand as an array of large to small values representing sets of 4 or less. Intended for 4 visible cards or simple AI
var twoPlusTwoEval = cardrank.NewTwoPlusTwoEval()

func toCardrankSuit(suit int) cardrank.Suit {
	switch suit {
	case 0:
		return cardrank.Spade
	case 1:
		return cardrank.Heart
	case 2:
		return cardrank.Diamond
	case 3:
		return cardrank.Club
	}
	return cardrank.InvalidSuit // Should not happen
}

func getRank(holeCards []card, communityCards []card) []int {
	allCards := append(holeCards, communityCards...)

	// Convert custom card struct to cardrank.Card
	crCards := make([]cardrank.Card, len(allCards))
	for i, c := range allCards {
		crCards[i] = cardrank.New(cardrank.Rank(c.Rank), toCardrankSuit(c.Suit))
	}

	// Evaluate the hand using cardrank's TwoPlusTwo evaluator
	rank := twoPlusTwoEval(crCards)

	// The cardrank library returns an EvalRank. We need to convert this to an int slice.
	// For now, we'll just return the integer value of the rank.
	return []int{int(rank)}
}
