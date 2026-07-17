package main

import (
	"testing"

	"github.com/cardrank/cardrank"
	"github.com/stretchr/testify/assert"
)

// TestGetRank verifies hand evaluation using the live deck convention
// (Rank 2..14 with Ace=14, Suit 0..3). Lower EvalRank = better hand.
func TestGetRank(t *testing.T) {
	tests := []struct {
		name             string
		holeCards        []string
		communityCards   []string
		expectedHandName string
	}{
		{
			name:             "Royal Flush",
			holeCards:        []string{"AS", "KS"},
			communityCards:   []string{"QS", "JS", "TS", "3D", "2C"},
			expectedHandName: "StraightFlush", // Royal flush is the best straight flush
		},
		{
			name:             "Straight Flush",
			holeCards:        []string{"9H", "8H"},
			communityCards:   []string{"7H", "6H", "5H", "2C", "3D"},
			expectedHandName: cardrank.StraightFlush.Name(),
		},
		{
			name:             "Four of a Kind",
			holeCards:        []string{"TC", "TD"},
			communityCards:   []string{"TH", "TS", "5C", "4D", "3H"},
			expectedHandName: cardrank.FourOfAKind.Name(),
		},
		{
			name:             "Full House",
			holeCards:        []string{"AC", "AD"},
			communityCards:   []string{"AH", "7C", "7D", "4H", "3S"},
			expectedHandName: cardrank.FullHouse.Name(),
		},
		{
			name:             "Flush",
			holeCards:        []string{"AS", "QS"},
			communityCards:   []string{"TS", "7S", "4S", "3H", "2D"},
			expectedHandName: cardrank.Flush.Name(),
		},
		{
			name:             "Straight",
			holeCards:        []string{"7C", "6D"},
			communityCards:   []string{"5H", "4S", "3C", "AD", "KH"},
			expectedHandName: cardrank.Straight.Name(),
		},
		{
			name:             "Wheel Straight (A-2-3-4-5)",
			holeCards:        []string{"AC", "2D"},
			communityCards:   []string{"3H", "4S", "5C", "9D", "KH"},
			expectedHandName: cardrank.Straight.Name(),
		},
		{
			name:             "Three of a Kind",
			holeCards:        []string{"JC", "JD"},
			communityCards:   []string{"JH", "7C", "6D", "5H", "2S"},
			expectedHandName: cardrank.ThreeOfAKind.Name(),
		},
		{
			name:             "Two Pair",
			holeCards:        []string{"AC", "AD"},
			communityCards:   []string{"7C", "7D", "6H", "5S", "2C"},
			expectedHandName: cardrank.TwoPair.Name(),
		},
		{
			name:             "Pair",
			holeCards:        []string{"AC", "KD"},
			communityCards:   []string{"AH", "7C", "5D", "4H", "2S"},
			expectedHandName: cardrank.Pair.Name(),
		},
		{
			name:             "High Card",
			holeCards:        []string{"AC", "QD"},
			communityCards:   []string{"TH", "8S", "6C", "4D", "2H"},
			expectedHandName: cardrank.HighCard.Name(),
		},
	}

	// Verify each hand category and that ranks strictly worsen down the list
	// (lower EvalRank = better hand)
	prevRank := 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rank := getRank(parseCards(t, tt.holeCards...), parseCards(t, tt.communityCards...))
			assert.NotEmpty(t, rank)
			got := cardrank.EvalRank(rank[0]).Name()
			assert.Equal(t, tt.expectedHandName, got, "hand category for %s", tt.name)
			assert.Greater(t, rank[0], prevRank, "%s should rank worse (higher value) than the previous, stronger hand", tt.name)
			prevRank = rank[0]
		})
	}
}

// TestGetRankKingsAndAces guards the historical bug where cards with Rank 13 (King)
// and 14 (Ace) were silently dropped by the evaluator
func TestGetRankKingsAndAces(t *testing.T) {
	// Pair of aces must beat a pair of kings, which must beat a pair of queens
	board := []string{"9C", "7D", "5H", "3S", "2C"}
	aces := getRank(parseCards(t, "AC", "AD"), parseCards(t, board...))[0]
	kings := getRank(parseCards(t, "KC", "KD"), parseCards(t, board...))[0]
	queens := getRank(parseCards(t, "QC", "QD"), parseCards(t, board...))[0]

	assert.Less(t, aces, kings, "pair of aces must beat pair of kings")
	assert.Less(t, kings, queens, "pair of kings must beat pair of queens")

	// All must evaluate as a pair (not high card, which happens if K/A were dropped)
	for _, r := range []int{aces, kings, queens} {
		assert.Equal(t, cardrank.Pair.Name(), cardrank.EvalRank(r).Name())
	}
}

// TestGetRankDoesNotCorruptHand guards the append-aliasing bug: evaluating a hand
// must not mutate the caller's hole card slice
func TestGetRankDoesNotCorruptHand(t *testing.T) {
	// Build a hole-card slice with extra capacity so a naive append would write
	// into its backing array
	backing := make([]card, 2, 7)
	copy(backing, parseCards(t, "AC", "KD"))
	community := parseCards(t, "QH", "JS", "TC")

	getRank(backing, community)

	assert.Equal(t, parseCards(t, "AC", "KD"), backing, "hole cards must not be mutated by evaluation")
}

// TestGetPlayerWithBestVisibleHand verifies the best-hand selection uses the
// correct sort direction (lower EvalRank = better)
func TestGetPlayerWithBestVisibleHand(t *testing.T) {
	state := &GameState{
		Players: []Player{
			{Name: "HighCard", Status: STATUS_PLAYING, Hand: parseCards(t, "2C", "7D")},
			{Name: "RoyalFlush", Status: STATUS_PLAYING, Hand: parseCards(t, "AS", "KS")},
			{Name: "PairOfNines", Status: STATUS_PLAYING, Hand: parseCards(t, "9C", "9D")},
			{Name: "Folded", Status: STATUS_FOLDED, Hand: parseCards(t, "AC", "AD")},
		},
		CommunityCards: parseCards(t, "QS", "JS", "TS", "4H", "3D"),
	}

	assert.Equal(t, 1, state.getPlayerWithBestVisibleHand(true), "royal flush should be the best hand")
	assert.Equal(t, 0, state.getPlayerWithBestVisibleHand(false), "high card should be the worst hand")
}
