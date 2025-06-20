package main

import (
	"testing"
	"time"
)

func TestTexasHoldemGameFlow(t *testing.T) {
	// Scenario: A simple game with a bot and a human player
	// The bot will make decisions, and we'll observe game progression.

	// Initialize game state
	state := createGameState(2, false) // Create a game state with 2 players, not registering with lobby
	state.Players[0].isBot = true
	state.Players[1].isBot = false

	// Initialize game state (some fields are set by createGameState)
	state.Players[0].cards = make([]card, 2)
	state.Players[1].cards = make([]card, 2)
	state.moveExpires = time.Now() // Set move expiration to now for immediate processing in tests
	state.Round = 0 // Pre-game setup
	state.Pot = 0
	state.CommunityCards = make([]card, 0)

	state.newRound() // Initialize the first round, dealing cards and setting blinds

	// Deal initial cards (simplified for test, normally handled by game start)
	state.Players[0].cards[0] = state.deck[state.deckIndex]
	state.deckIndex++
	state.Players[0].cards[1] = state.deck[state.deckIndex]
	state.deckIndex++
	state.Players[1].cards[0] = state.deck[state.deckIndex]
	state.deckIndex++
	state.Players[1].cards[1] = state.deck[state.deckIndex]
	state.deckIndex++

	// Simulate game progression through rounds
	// This will be an iterative process, calling RunGameLogic and asserting state.

	// Simulate game progression through rounds
	maxTurns := 100 // Set a maximum number of turns to prevent infinite loops
	turn := 0

	for state.Round <= 4 && countPlayersInHand(state) > 1 && turn < maxTurns {
		turn++
		t.Logf("--- Turn %d: Before RunGameLogic ---", turn)
		t.Logf("Round: %d, ActivePlayer: %d, Pot: %d, P1 Status: %d, P2 Status: %d, P1 Move: %s, P2 Move: %s",
			state.Round, state.ActivePlayer, state.Pot, state.Players[0].Status, state.Players[1].Status, state.Players[0].Move, state.Players[1].Move)
		t.Logf("Community Cards: %v", state.CommunityCards)

		state.RunGameLogic()
		state.moveExpires = time.Now() // Force move expiration for testing

		t.Logf("--- Turn %d: After RunGameLogic ---", turn)
		t.Logf("Round: %d, ActivePlayer: %d, Pot: %d", state.Round, state.ActivePlayer, state.Pot)
		t.Logf("P1 Purse: %d, P2 Purse: %d", state.Players[0].Purse, state.Players[1].Purse)
		t.Logf("Community Cards: %v", state.CommunityCards)
		t.Logf("---------------------------")

		// RunGameLogic handles advancing the active player
	}

	// Final assertions (e.g., winner, final pot)


	// For now, just ensure the test runs without panicking.
	t.Log("TestTexasHoldemGameFlow completed.")
}

// Helper function to count players still in the hand
func countPlayersInHand(state *GameState) int {
	count := 0
	for _, player := range state.Players {
		if player.Status == STATUS_PLAYING {
			count++
		}
	}
	return count
}