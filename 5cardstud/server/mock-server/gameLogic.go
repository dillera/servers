package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cardrank/cardrank"
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
	- 4th street - 10
*/

const ANTI = 1
const BRINGIN = 2
const LOW = 5
const HIGH = 10
const STARTING_PURSE = 200

var lock sync.Mutex

var suitLookup = []string{"C", "D", "H", "S"}
var valueLookup = []string{"", "", "2", "3", "4", "5", "6", "7", "8", "9", "T", "J", "Q", "K", "A"}
var moveLookup = map[string]string{
	"FO": "FOLD",
	"CH": "CHECK",
	"BB": "POST",
	"BL": "BET", // BET LOW (e.g. 5 of 5/10, or 2 of 2/5 first round)
	"BH": "BET", // BET HIGH (e.g. 10)
	"CA": "CALL",
	"RL": "RAISE",
	"RH": "RAISE",
}

var playerPool = []player{
	{Name: "Clyde BOT"},
	{Name: "Spock BOT"},
	{Name: "Kirk BOT"},
	{Name: "Hulk BOT"},
	{Name: "Fry BOT"},
	{Name: "Meg BOT"},
	{Name: "GI BOT"},
	{Name: "AI BOT"},
}

type validMove struct {
	Move string `json:"move"`
	Name string `json:"name"`
}

type card struct {
	value int
	suit  int
}

type player struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
	Bet    int    `json:"bet"`
	Move   string `json:"move"`
	Purse  int    `json:"purse"`
	Hand   string `json:"hand"`

	// Internal
	IsBot bool
	cards []card
}

type gameState struct {
	// External (JSON)
	LastResult   string      `json:"lastResult"`
	Round        int         `json:"round"`
	Pot          int         `json:"pot"`
	ActivePlayer int         `json:"activePlayer"`
	ValidMoves   []validMove `json:"validMoves"`
	Players      []player    `json:"players"`
	MoveTime     int         `json:"moveTime"`

	// Internal
	deck         []card
	deckIndex    int
	currentBet   int
	gameOver     bool
	clientPlayer int
	table        string
	wonByFolds   bool
	isMockGame   bool
	moveExpires  time.Time
	serverName   string
}

func createGameState(playerCount int, isMockGame bool) *gameState {
	deck := []card{}

	// Create deck of 52 cards
	for suit := 0; suit < 4; suit++ {
		for value := 2; value < 15; value++ {
			card := card{value: value, suit: suit}
			deck = append(deck, card)
		}
	}

	state := gameState{}
	state.deck = deck
	state.Round = 0
	state.ActivePlayer = -1
	state.isMockGame = isMockGame

	// Force between 2 and 8 players during mock games
	if isMockGame {
		playerCount = int(math.Min(math.Max(2, float64(playerCount)), 8))
	}
	state.Players = playerPool[0:playerCount]
	for i := 0; i < playerCount; i++ {
		state.Players[i].Purse = STARTING_PURSE
		if i > 0 || !isMockGame {
			state.Players[i].IsBot = true
		}
	}

	log.Print("Created GameState")
	return &state
}

func (state *gameState) updatePlayerCount(playerCount int) {
	if playerCount <= len(state.Players) || playerCount > 8 {
		return
	}

	delta := playerCount - len(state.Players)
	for i := 0; i < delta; i++ {
		// Create a player that is waiting to play the next game
		player := playerPool[len(state.Players)]
		player.Status = 0
		player.Purse = STARTING_PURSE
		player.cards = []card{}
		state.Players = append(state.Players, player)
	}
}

func (state *gameState) newRound() {
	state.Round++

	// Clear pot at start so players can anti
	if state.Round == 1 {
		state.Pot = 0
	}

	// Reset players for this round
	for i := 0; i < len(state.Players); i++ {

		// Get pointer to player
		player := &state.Players[i]

		if state.Round > 1 {
			// If not the first round, add any bets into the pot
			state.Pot += player.Bet
		} else {

			// First round of a new game? Reset player status and take the ANTI
			if player.Purse > 2 {
				player.Status = 1
				player.Purse -= ANTI
				state.Pot += ANTI
			} else {
				// Player doesn't have enough money to play
				player.Status = 0
			}
			player.cards = []card{}
		}

		// Reset player's last move/bet for this round
		player.Move = ""
		player.Bet = 0
	}

	state.currentBet = 0

	// First round of a new game? Shuffle the cards and deal an extra card
	if state.Round == 1 {

		// Shuffle the deck 7 times :)
		for shuffle := 0; shuffle < 7; shuffle++ {
			rand.Shuffle(len(state.deck), func(i, j int) { state.deck[i], state.deck[j] = state.deck[j], state.deck[i] })
		}
		state.deckIndex = 0
		state.dealCards()
	}

	state.dealCards()
	state.ActivePlayer = state.getPlayerWithBestVisibleHand(state.Round > 1)
	state.resetPlayerTimer()
}

func (state *gameState) getPlayerWithBestVisibleHand(highHand bool) int {

	ranks := [][]int{}

	for i := 0; i < len(state.Players); i++ {
		player := &state.Players[i]
		if player.Status == 1 {
			rank := getRank(player.cards[1:len(player.cards)])

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
	if highHand {
		return ranks[0][0]
	} else {
		return ranks[len(ranks)-1][0]
	}
}

func (state *gameState) dealCards() {
	for i, player := range state.Players {
		if player.Status == 1 {
			player.cards = append(player.cards, state.deck[state.deckIndex])
			state.Players[i] = player
			state.deckIndex++
		}
	}
}

func (state *gameState) setClientPlayerByName(playerName string) {
	state.clientPlayer = slices.IndexFunc(state.Players, func(p player) bool { return strings.EqualFold(p.Name, playerName) })

	// Add new player if there is room
	if state.clientPlayer < 0 && len(state.Players) < 8 {

		newPlayer := player{
			Name:   playerName,
			Status: 0,
			Purse:  STARTING_PURSE,
			cards:  []card{},
		}

		state.Players = append(state.Players, newPlayer)
		state.clientPlayer = len(state.Players) - 1
	}
}

func (state *gameState) endGame() {
	// The next request for /state will start a new game

	// Hand rank details (for future)
	// Rank: SF, 4K, FH, F, S, 3K, 2P, 1P, HC

	state.gameOver = true
	state.ActivePlayer = -1
	state.Round = 5

	remainingPlayers := []int{}
	pockets := [][]cardrank.Card{}

	for index, player := range state.Players {
		state.Pot += player.Bet
		if player.Status == 1 {
			remainingPlayers = append(remainingPlayers, index)
			hand := ""
			// Loop through and build hand string
			for _, card := range player.cards {
				hand += valueLookup[card.value] + suitLookup[card.suit]
			}
			pockets = append(pockets, cardrank.Must(hand))
		}
	}

	evs := cardrank.StudFive.EvalPockets(pockets, nil)
	order, pivot := cardrank.Order(evs, false)

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
	} else {
		state.wonByFolds = true
		result += " won by default"
	}
	state.LastResult = result
	log.Println(result)
}

// Emulates simplified player/logic for 5 card stud
func (state *gameState) runGameLogic() {

	// Only one thread can execute this at a time, to avoid multiple threads updating the state
	lock.Lock()
	defer lock.Unlock()

	// We can't play a game until there are at least 2 players
	if len(state.Players) < 2 {
		return
	}

	// Very first call of state? Initialize first round but do not play for any BOTs
	if state.Round == 0 {
		state.newRound()
		return
	}

	//isHumanPlayer := state.ActivePlayer == state.clientPlayer

	if state.gameOver {
		state.Round = 0
		state.gameOver = false
		state.newRound()
		return
	}

	// Check if only one player is left
	playersLeft := 0
	for _, player := range state.Players {
		if player.Status == 1 {
			playersLeft++
		}
	}

	if playersLeft == 1 {
		state.endGame()
		return
	}

	// Check if we should start the next round. One of the following must be true
	// 1. We got back to the player who made the most recent bet/raise
	// 2. There were checks/folds around the table
	if state.ActivePlayer > -1 {
		if (state.currentBet > 0 && state.Players[state.ActivePlayer].Bet == state.currentBet) ||
			(state.currentBet == 0 && state.Players[state.ActivePlayer].Move != "") {
			if state.Round == 4 {
				state.endGame()
			} else {
				state.newRound()
			}
			return
		}
	}

	// If a real game, return if the move timer has not expired
	if !state.isMockGame {
		moveTimeRemaining := int(time.Until(state.moveExpires).Seconds())
		if moveTimeRemaining > 0 {
			return
		}
	}

	// Force a move for this player or BOT if they are in the game and have not folded
	if state.Players[state.ActivePlayer].Status == 1 {
		cards := state.Players[state.ActivePlayer].cards
		moves := state.getValidMoves()

		// Default to FOLD
		choice := 0

		// Never fold if CHECK is an option. This applies to forced player moves as well as bots
		if len(moves) > 1 && moves[1].Move == "CH" {
			choice = 1
		}

		// If this is a bot, pick the best move using some simple logic (sometimes random)
		if state.Players[state.ActivePlayer].IsBot {

			// Potential TODO: If on round 5 and check is not an option, fold if there is a visible hand that beats the bot's hand.
			//if len(cards) == 5 && len(moves) > 1 && moves[1].Move == "CH" {}

			// Hardly ever fold early if a BOT has an jack or higher.
			if state.Round < 3 && len(moves) > 1 && rand.Intn(3) > 0 && slices.IndexFunc(cards, func(c card) bool { return c.value > 10 }) > -1 {
				choice = 1
			}

			// Likely Don't fold if BOT has a pair or better
			rank := getRank(cards)
			if rank[0] < 300 && rand.Intn(20) > 0 {
				choice = 1
			}

			// Don't fold if BOT has a 2 pair or better
			if rank[0] < 200 {
				choice = 1
			}

			// Raise the bet if three of a kind or better
			if len(moves) > 2 && rank[0] < 312 && state.currentBet < LOW {
				choice = 2
			} else if len(moves) > 2 && state.getPlayerWithBestVisibleHand(true) == state.ActivePlayer && state.currentBet < HIGH && (rank[0] < 306) {
				choice = len(moves) - 1
			} else {

				// Consider bet/call/raise most of the time
				if len(moves) > 1 && rand.Intn(3) > 0 && (len(cards) > 2 ||
					cards[0].value == cards[1].value ||
					math.Abs(float64(cards[1].value-cards[0].value)) < 3 ||
					cards[0].value > 8 ||
					cards[1].value > 5) {

					// Avoid endless raises
					if state.currentBet >= 20 || rand.Intn(3) > 0 {
						choice = 1
					} else {
						choice = rand.Intn(len(moves)-1) + 1
					}

				}
			}
		}

		move := moves[choice]

		state.performMove(move.Move)
	}

}

// Performs the requested move for the active player, and returns true if successful
func (state *gameState) performMove(move string) bool {

	// Get pointer to player
	player := &state.Players[state.ActivePlayer]

	// Sanity Check 2 - Player status should be 1 to move. In theory they should never be active if their status is != 1
	if player.Status != 1 {
		return false
	}

	// Only continue if this is a valid move for this player
	moveIsValid := false
	for _, validMove := range state.getValidMoves() {
		moveIsValid = moveIsValid || validMove.Move == move
	}

	if !moveIsValid {
		return false
	}

	if move == "FO" { // FOLD
		player.Status = 2
	} else if move != "CH" { // Not Checking

		// Default raise to 0 (effectively a CALL)
		raise := 0

		if move == "BH" || move == "RH" {
			raise = HIGH
		} else if move == "BL" || move == "RL" {
			raise = LOW
			if state.Pot == BRINGIN {
				// If betting LOW the very first time and the pot is BRINGIN
				// just make their bet enough to make the total bet LOW
				raise = -BRINGIN
			}
		} else if move == "BB" {
			raise = BRINGIN
		}

		// Place the bet
		delta := state.currentBet + raise - player.Bet
		state.currentBet += raise
		player.Bet += delta
		player.Purse -= delta
	}

	player.Move = moveLookup[move]
	state.nextValidPlayer()

	return true
}

func (state *gameState) resetPlayerTimer() {
	seconds := 30
	if state.Players[state.ActivePlayer].IsBot {
		seconds = 1
	}
	state.moveExpires = time.Now().Add(time.Second * time.Duration(seconds))
}

func (state *gameState) nextValidPlayer() {
	// Move to next player
	state.ActivePlayer = (state.ActivePlayer + 1) % len(state.Players)

	// Skip over player if not in this game (joined late / folded)
	for state.Players[state.ActivePlayer].Status != 1 {
		state.ActivePlayer = (state.ActivePlayer + 1) % len(state.Players)
	}
	state.resetPlayerTimer()
}

func (state *gameState) getValidMoves() []validMove {
	moves := []validMove{}

	// Any player after the bring-in player may fold
	if state.currentBet > 0 || state.Round > 1 {
		moves = append(moves, validMove{Move: "FO", Name: "Fold"})
	}

	player := state.Players[state.ActivePlayer]

	if state.currentBet < LOW {
		// First round, BET BRINGIN (2) or BET LOW
		if state.currentBet == 0 {
			if state.Round == 1 {
				moves = append(moves, validMove{Move: "BB", Name: fmt.Sprint("Post ", BRINGIN)})
			} else {
				moves = append(moves, validMove{Move: "CH", Name: "Check"})
			}
		} else if player.Purse >= state.currentBet-player.Bet {
			moves = append(moves, validMove{Move: "CA", Name: "Call"})
		}
		if state.Round < 3 && player.Purse >= LOW {
			moves = append(moves, validMove{Move: "BL", Name: fmt.Sprint("Bet ", LOW)})
		} else if state.Round > 2 && player.Purse >= HIGH {
			moves = append(moves, validMove{Move: "BH", Name: fmt.Sprint("Bet ", HIGH)})
		}
	} else {
		if player.Purse >= state.currentBet-player.Bet {
			moves = append(moves, validMove{Move: "CA", Name: "Call"})
		}
		if state.Players[state.ActivePlayer].Purse >= state.currentBet-player.Bet+LOW {
			moves = append(moves, validMove{Move: "RL", Name: fmt.Sprint("Raise ", LOW)})
		}
	}

	return moves
}

// Creates a copy of the state and modifies it to be from the
// perspective of this client (e.g. player array, visible cards)
func (state *gameState) createClientState() gameState {

	stateCopy := *state

	setActivePlayer := false

	// Check if we are at the end of a round, or the game. If so, no player is active. This lets the client perform
	// end of round/game tasks/animation
	if state.gameOver || len(stateCopy.Players) < 2 || (stateCopy.ActivePlayer > -1 && ((state.currentBet > 0 && state.Players[state.ActivePlayer].Bet == state.currentBet) ||
		(state.currentBet == 0 && state.Players[state.ActivePlayer].Move != ""))) {
		stateCopy.ActivePlayer = -1
		setActivePlayer = true
	} else {
		stateCopy.MoveTime = int(time.Until(stateCopy.moveExpires).Seconds())
	}

	// Now, store a copy of state players, then loop
	// through and add to the state copy, starting
	// with this player first

	statePlayers := stateCopy.Players
	stateCopy.Players = []player{}

	// Loop through each player and create the hand, starting at this player, so all clients see the same order regardless of starting player
	for i := state.clientPlayer; i < state.clientPlayer+len(statePlayers); i++ {

		// Wrap around to beginning of playar array when needed
		playerIndex := i % len(statePlayers)

		// Update the ActivePlayer to be client relative
		if !setActivePlayer && playerIndex == stateCopy.ActivePlayer {
			setActivePlayer = true
			stateCopy.ActivePlayer = i - state.clientPlayer
		}

		player := statePlayers[playerIndex]
		player.Hand = ""

		switch player.Status {
		case 1:
			// Loop through and build hand string, taking
			// care to not disclose the first card of a hand to other players
			for cardIndex, card := range player.cards {
				if cardIndex > 0 || playerIndex == state.clientPlayer || (state.gameOver && !state.wonByFolds) {
					player.Hand += valueLookup[card.value] + suitLookup[card.suit]
				} else {
					player.Hand += "??"
				}
			}
		case 2:
			player.Hand = "??"
		}

		// Add this player to the copy of the state going out
		stateCopy.Players = append(stateCopy.Players, player)
	}

	// Determine valid moves for this player (if their turn)
	if stateCopy.ActivePlayer == 0 {
		stateCopy.ValidMoves = state.getValidMoves()
	}

	return stateCopy
}

// Ranks hand as an array of large to small values representing sets of 4 or less. Intended for 4 visible cards or simple AI
func getRank(cards []card) []int {
	rank := []int{}
	rankSuit := []int{}
	sets := map[int]int{}

	// Loop through hand once to create sets (cards of the same value)
	for i := 0; i < len(cards); i++ {
		sets[cards[i].value]++
	}

	// Loop through a second time to add the rank of each set (or single card)
	for i := 0; i < len(cards); i++ {
		val := cards[i].value
		set := sets[val]

		// Ranking highest value the lowest so ascending sort can be used
		rank = append(rank, 100*(5-set)-val)

		// Ranking with suit as a tie breaker
		rankSuit = append(rankSuit, 100*(5-set)-(val*4+cards[i].suit))

	}

	sort.Ints(rank)
	// Fill out empty 999s to make a 4 length to avoid bounds checks
	for len(rank) < 4 {
		rank = append(rank, 999)
	}
	sort.Ints(rankSuit)
	rank = append(rank, rankSuit...)
	for len(rank) < 8 {
		rank = append(rank, 999)
	}
	return rank
}
