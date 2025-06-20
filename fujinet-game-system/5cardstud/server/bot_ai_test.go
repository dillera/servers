package main

import (
	"testing"
)

func TestBotAI(t *testing.T) {
	// This is a very basic test for the bot AI, primarily to ensure it doesn't panic
	// and makes a valid move. More comprehensive tests would require mocking
	// random number generation and detailed state assertions.

	state := &GameState{
		Players: []Player{
			{Name: "Bot1", isBot: true, Status: STATUS_PLAYING, Purse: 1000, cards: []card{{value: 12, suit: 0}, {value: 11, suit: 1}}}, // Ace, King
			{Name: "Player2", isBot: false, Status: STATUS_PLAYING, Purse: 1000},
		},
		ActivePlayer: 0,
		currentBet:   0,
		Round:        1, // Pre-flop
	}

	// Simulate a bot's turn
	state.RunGameLogic()

	// Check if the bot made a move
	if state.Players[0].Move == "" {
		t.Errorf("Bot did not make a move")
	}

	// Test post-flop scenario with a strong hand
	state = &GameState{
		Players: []Player{
			{Name: "Bot1", isBot: true, Status: STATUS_PLAYING, Purse: 1000, cards: []card{{value: 12, suit: 0}, {value: 12, suit: 1}}}, // Pair of Aces
			{Name: "Player2", isBot: false, Status: STATUS_PLAYING, Purse: 1000},
		},
		ActivePlayer:   0,
		currentBet:     50,
		Round:          2, // Flop
		CommunityCards: []card{{value: 12, suit: 2}, {value: 5, suit: 0}, {value: 4, suit: 1}}, // Ace, 7, 6
	}

	state.RunGameLogic()

	if state.Players[0].Move == "" {
		t.Errorf("Bot did not make a move in post-flop scenario")
	}

}