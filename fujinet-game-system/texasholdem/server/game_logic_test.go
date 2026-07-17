package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestShuffleDeck ensures the deck is properly initialized and shuffled.
func TestShuffleDeck(t *testing.T) {
	state := &GameState{}
	state.initializeDeck()

	// A standard deck should have 52 cards.
	assert.Equal(t, 52, len(state.Deck), "Deck should have 52 cards after initialization")

	// Create a copy of the initial deck to compare against.
	initialDeck := make([]card, len(state.Deck))
	copy(initialDeck, state.Deck)

	// Shuffle the deck.
	state.shuffleDeck()

	// The shuffled deck should still have 52 cards.
	assert.Equal(t, 52, len(state.Deck), "Deck should still have 52 cards after shuffling")

	// The shuffled deck should not be identical to the initial, ordered deck.
	// This is a probabilistic test, but with a 52-card deck, the chance of shuffling to the same order is astronomically low.
	assert.NotEqual(t, initialDeck, state.Deck, "Shuffled deck should not be the same as the initial deck")
}

// TestDealHoleCards ensures that the correct number of cards are dealt to players.
func TestDealHoleCards(t *testing.T) {
	state := &GameState{
		Players: []Player{
			{Name: "Player 1", Status: STATUS_PLAYING},
			{Name: "Player 2", Status: STATUS_PLAYING},
			{Name: "Player 3", Status: STATUS_PLAYING},
		},
	}
	state.initializeDeck()
	state.shuffleDeck()

	// Deal the hole cards
	state.dealHoleCards()

	// Each player should have exactly 2 cards.
	for _, p := range state.Players {
		assert.Equal(t, 2, len(p.Hand), "Each player should have 2 hole cards")
	}

	// The deck should now have 52 - (3 players * 2 cards) = 46 cards left.
	assert.Equal(t, 6, state.deckIndex, "Deck index should be at 6 after dealing 2 cards to 3 players")
}
