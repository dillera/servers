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
	state.moveExpires = time.Now().Add(1 * time.Second) // Set a short timer for testing
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

	// Example: Pre-flop (Round 1)
	// Run game logic until round ends or a winner is determined
	for state.Round <= 4 && countPlayersInHand(state) > 1 {
		// Simulate a short delay to allow RunGameLogic to process
		time.Sleep(10 * time.Millisecond)

		state.RunGameLogic()

		// Add assertions here to check game state after each RunGameLogic call
		// For example, check active player, pot, player moves, etc.
		// t.Logf("Round: %d, ActivePlayer: %d, Pot: %d, P1 Move: %s, P2 Move: %s",
		// 	state.Round, state.ActivePlayer, state.Pot, state.Players[0].Move, state.Players[1].Move)

		// This loop needs more sophisticated logic to advance rounds and handle betting.
		// For now, it will likely loop indefinitely or until an error.
		// We need to implement logic to detect end of betting round and advance to next round.
	}

	// Final assertions (e.g., winner, final pot)
	// t.Logf("Game Over. Winner: %s, Final Pot: %d", state.Winner, state.Pot)
	// t.Logf("Player1 Purse: %d, Player2 Purse: %d", state.Players[0].Purse, state.Players[1].Purse)

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