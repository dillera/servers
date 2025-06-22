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
			holeCards:      []card{{Rank: 12, Suit: 0}, {Rank: 11, Suit: 0}}, // Ace, King of Spades
			communityCards: []card{{Rank: 10, Suit: 0}, {Rank: 9, Suit: 0}, {Rank: 8, Suit: 0}, {Rank: 7, Suit: 0}, {Rank: 6, Suit: 0}}, // 10, 9, 8, 7, 6 of Spades
			expectedHandName:   cardrank.RoyalFlush.Name(),
		},
		{
			name: "Straight Flush",
			holeCards:      []card{{Rank: 5, Suit: 1}, {Rank: 4, Suit: 1}}, // 7, 6 of Hearts
			communityCards: []card{{Rank: 3, Suit: 1}, {Rank: 2, Suit: 1}, {Rank: 1, Suit: 1}, {Rank: 0, Suit: 1}, {Rank: 12, Suit: 2}}, // 5, 4, 3, 2 of Hearts, Ace of Diamonds
			expectedHandName:   cardrank.StraightFlush.Name(),
		},
		{
			name: "Four of a Kind",
			holeCards:      []card{{Rank: 8, Suit: 0}, {Rank: 8, Suit: 1}}, // 10 of Spades, 10 of Hearts
			communityCards: []card{{Rank: 8, Suit: 2}, {Rank: 8, Suit: 3}, {Rank: 5, Suit: 0}, {Rank: 4, Suit: 1}, {Rank: 3, Suit: 2}}, // 10 of Diamonds, 10 of Clubs, 7, 6, 5
			expectedHandName:   cardrank.FourOfAKind.Name(),
		},
		{
			name: "Full House",
			holeCards:      []card{{Rank: 12, Suit: 0}, {Rank: 12, Suit: 1}}, // Ace of Spades, Ace of Hearts
			communityCards: []card{{Rank: 12, Suit: 2}, {Rank: 5, Suit: 0}, {Rank: 5, Suit: 1}, {Rank: 4, Suit: 2}, {Rank: 3, Suit: 3}}, // Ace of Diamonds, 7 of Spades, 7 of Hearts, 6, 5
			expectedHandName:   cardrank.FullHouse.Name(),
		},
		{
			name: "Flush",
			holeCards:      []card{{Rank: 12, Suit: 0}, {Rank: 10, Suit: 0}}, // Ace, Queen of Spades
			communityCards: []card{{Rank: 8, Suit: 0}, {Rank: 5, Suit: 0}, {Rank: 2, Suit: 0}, {Rank: 1, Suit: 1}, {Rank: 0, Suit: 2}}, // 10, 7, 4 of Spades, 3 of Hearts, 2 of Diamonds
			expectedHandName:   cardrank.Flush.Name(),
		},
		{
			name: "Straight",
			holeCards:      []card{{Rank: 5, Suit: 0}, {Rank: 4, Suit: 1}}, // 7 of Spades, 6 of Hearts
			communityCards: []card{{Rank: 3, Suit: 2}, {Rank: 2, Suit: 3}, {Rank: 1, Suit: 0}, {Rank: 12, Suit: 1}, {Rank: 11, Suit: 2}}, // 5 of Diamonds, 4 of Clubs, 3 of Spades, Ace of Hearts, King of Diamonds
			expectedHandName:   cardrank.Straight.Name(),
		},
		{
			name: "Three of a Kind",
			holeCards:      []card{{Rank: 9, Suit: 0}, {Rank: 9, Suit: 1}}, // Jack of Spades, Jack of Hearts
			communityCards: []card{{Rank: 9, Suit: 2}, {Rank: 5, Suit: 0}, {Rank: 4, Suit: 1}, {Rank: 3, Suit: 2}, {Rank: 2, Suit: 3}}, // Jack of Diamonds, 7, 6, 5, 4
			expectedHandName:   cardrank.ThreeOfAKind.Name(),
		},
		{
			name: "Two Pair",
			holeCards:      []card{{Rank: 12, Suit: 0}, {Rank: 12, Suit: 1}}, // Ace of Spades, Ace of Hearts
			communityCards: []card{{Rank: 5, Suit: 0}, {Rank: 5, Suit: 1}, {Rank: 4, Suit: 2}, {Rank: 3, Suit: 3}, {Rank: 2, Suit: 0}}, // 7 of Spades, 7 of Hearts, 6, 5, 4
			expectedHandName:   cardrank.TwoPair.Name(),
		},
		{
			name: "Pair",
			holeCards:      []card{{Rank: 12, Suit: 0}, {Rank: 11, Suit: 1}}, // Ace of Spades, King of Hearts
			communityCards: []card{{Rank: 12, Suit: 2}, {Rank: 5, Suit: 0}, {Rank: 4, Suit: 1}, {Rank: 3, Suit: 2}, {Rank: 2, Suit: 3}}, // Ace of Diamonds, 7, 6, 5, 4
			expectedHandName:   cardrank.Pair.Name(),
		},
		{
			name: "High Card",
			holeCards:      []card{{Rank: 12, Suit: 0}, {Rank: 10, Suit: 1}}, // Ace of Spades, Queen of Hearts
			communityCards: []card{{Rank: 8, Suit: 2}, {Rank: 6, Suit: 3}, {Rank: 4, Suit: 0}, {Rank: 2, Suit: 1}, {Rank: 0, Suit: 2}}, // 10, 8, 6, 4, 2
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