package main

import (
	"testing"

	"github.com/cardrank/cardrank"
)

func TestGetRank(t *testing.T) {
	tests := []struct {
		name           string
		holeCards      []card
		communityCards []card
		expectedHandName string // Using the hand name from cardrank library
	}{
		{
			name: "Royal Flush",
			holeCards:      []card{{value: 12, suit: 0}, {value: 11, suit: 0}}, // Ace, King of Spades
			communityCards: []card{{value: 10, suit: 0}, {value: 9, suit: 0}, {value: 8, suit: 0}, {value: 7, suit: 0}, {value: 6, suit: 0}}, // 10, 9, 8, 7, 6 of Spades
			expectedHandName:   cardrank.RoyalFlush.Name(),
		},
		{
			name: "Straight Flush",
			holeCards:      []card{{value: 5, suit: 1}, {value: 4, suit: 1}}, // 7, 6 of Hearts
			communityCards: []card{{value: 3, suit: 1}, {value: 2, suit: 1}, {value: 1, suit: 1}, {value: 0, suit: 1}, {value: 12, suit: 2}}, // 5, 4, 3, 2 of Hearts, Ace of Diamonds
			expectedHandName:   cardrank.StraightFlush.Name(),
		},
		{
			name: "Four of a Kind",
			holeCards:      []card{{value: 8, suit: 0}, {value: 8, suit: 1}}, // 10 of Spades, 10 of Hearts
			communityCards: []card{{value: 8, suit: 2}, {value: 8, suit: 3}, {value: 5, suit: 0}, {value: 4, suit: 1}, {value: 3, suit: 2}}, // 10 of Diamonds, 10 of Clubs, 7, 6, 5
			expectedHandName:   cardrank.FourOfAKind.Name(),
		},
		{
			name: "Full House",
			holeCards:      []card{{value: 12, suit: 0}, {value: 12, suit: 1}}, // Ace of Spades, Ace of Hearts
			communityCards: []card{{value: 12, suit: 2}, {value: 5, suit: 0}, {value: 5, suit: 1}, {value: 4, suit: 2}, {value: 3, suit: 3}}, // Ace of Diamonds, 7 of Spades, 7 of Hearts, 6, 5
			expectedHandName:   cardrank.FullHouse.Name(),
		},
		{
			name: "Flush",
			holeCards:      []card{{value: 12, suit: 0}, {value: 10, suit: 0}}, // Ace, Queen of Spades
			communityCards: []card{{value: 8, suit: 0}, {value: 5, suit: 0}, {value: 2, suit: 0}, {value: 1, suit: 1}, {value: 0, suit: 2}}, // 10, 7, 4 of Spades, 3 of Hearts, 2 of Diamonds
			expectedHandName:   cardrank.Flush.Name(),
		},
		{
			name: "Straight",
			holeCards:      []card{{value: 5, suit: 0}, {value: 4, suit: 1}}, // 7 of Spades, 6 of Hearts
			communityCards: []card{{value: 3, suit: 2}, {value: 2, suit: 3}, {value: 1, suit: 0}, {value: 12, suit: 1}, {value: 11, suit: 2}}, // 5 of Diamonds, 4 of Clubs, 3 of Spades, Ace of Hearts, King of Diamonds
			expectedHandName:   cardrank.Straight.Name(),
		},
		{
			name: "Three of a Kind",
			holeCards:      []card{{value: 9, suit: 0}, {value: 9, suit: 1}}, // Jack of Spades, Jack of Hearts
			communityCards: []card{{value: 9, suit: 2}, {value: 5, suit: 0}, {value: 4, suit: 1}, {value: 3, suit: 2}, {value: 2, suit: 3}}, // Jack of Diamonds, 7, 6, 5, 4
			expectedHandName:   cardrank.ThreeOfAKind.Name(),
		},
		{
			name: "Two Pair",
			holeCards:      []card{{value: 12, suit: 0}, {value: 12, suit: 1}}, // Ace of Spades, Ace of Hearts
			communityCards: []card{{value: 5, suit: 0}, {value: 5, suit: 1}, {value: 4, suit: 2}, {value: 3, suit: 3}, {value: 2, suit: 0}}, // 7 of Spades, 7 of Hearts, 6, 5, 4
			expectedHandName:   cardrank.TwoPair.Name(),
		},
		{
			name: "Pair",
			holeCards:      []card{{value: 12, suit: 0}, {value: 11, suit: 1}}, // Ace of Spades, King of Hearts
			communityCards: []card{{value: 12, suit: 2}, {value: 5, suit: 0}, {value: 4, suit: 1}, {value: 3, suit: 2}, {value: 2, suit: 3}}, // Ace of Diamonds, 7, 6, 5, 4
			expectedHandName:   cardrank.Pair.Name(),
		},
		{
			name: "High Card",
			holeCards:      []card{{value: 12, suit: 0}, {value: 10, suit: 1}}, // Ace of Spades, Queen of Hearts
			communityCards: []card{{value: 8, suit: 2}, {value: 6, suit: 3}, {value: 4, suit: 0}, {value: 2, suit: 1}, {value: 0, suit: 2}}, // 10, 8, 6, 4, 2
			expectedHandName:   cardrank.HighCard.Name(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rank := getRank(tt.holeCards, tt.communityCards)
			if len(rank) == 0 || cardrank.EvalRank(rank[0]).Name() != tt.expectedHandName {
				t.Errorf("getRank() for %s = %s, want %s", tt.name, cardrank.EvalRank(rank[0]).Name(), tt.expectedHandName)
			}
		})
	}
}

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
