package main

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBotAIPreFlop verifies a bot acts pre-flop with a premium hand
func TestBotAIPreFlop(t *testing.T) {
	useFastTimers(t)

	state := &GameState{
		Players: []Player{
			{Name: "Bot1", isBot: true, Status: STATUS_PLAYING, Purse: 1000,
				Hand:    parseCards(t, "AC", "KD"),
				profile: BotProfile{Name: "Aggressive", VPIP: 0.8, PFR: 0.9, BluffFrequency: 0.1}},
			{Name: "Player2", isBot: false, Status: STATUS_PLAYING, Purse: 1000, Hand: parseCards(t, "2C", "7D")},
		},
		ActivePlayer: 0,
		currentBet:   0,
		Round:        1, // Pre-flop
		rng:          rand.New(rand.NewSource(1)),
	}

	move := state.getBotMove()
	assert.NotEmpty(t, move, "bot must choose a move pre-flop")
	assert.NotEqual(t, "FO", move, "an aggressive bot should not fold AK when checking is free")
}

// TestBotAIPreFlopFoldsTrash verifies a tight bot folds a weak hand facing a bet
func TestBotAIPreFlopFoldsTrash(t *testing.T) {
	state := &GameState{
		Players: []Player{
			{Name: "Bot1", isBot: true, Status: STATUS_PLAYING, Purse: 1000, Bet: 0,
				Hand:    parseCards(t, "2C", "7D"), // Worst starting hand in poker
				profile: BotProfile{Name: "Tight", VPIP: 0.1, PFR: 0.5, BluffFrequency: 0.0}},
			{Name: "Player2", isBot: false, Status: STATUS_PLAYING, Purse: 990, Bet: 10},
		},
		ActivePlayer: 0,
		currentBet:   10,
		Round:        1,
		rng:          rand.New(rand.NewSource(1)),
	}

	assert.Equal(t, "FO", state.getBotMove(), "a tight bot must fold 7-2 offsuit facing a bet")
}

// TestBotAIPostFlopStrongHand verifies the post-flop AI is active (not the old
// check/fold workaround): a bot with a flopped set should bet or raise
func TestBotAIPostFlopStrongHand(t *testing.T) {
	state := &GameState{
		Players: []Player{
			{Name: "Bot1", isBot: true, Status: STATUS_PLAYING, Purse: 1000,
				Hand:    parseCards(t, "AC", "AD"), // Flopped top set
				profile: BotProfile{Name: "Aggressive", VPIP: 0.8, PFR: 0.9, BluffFrequency: 0.1}},
			{Name: "Player2", isBot: false, Status: STATUS_PLAYING, Purse: 1000, Hand: parseCards(t, "2C", "7D")},
		},
		ActivePlayer:   0,
		currentBet:     0,
		Round:          2, // Flop
		CommunityCards: parseCards(t, "AH", "7C", "4D"),
		rng:            rand.New(rand.NewSource(1)),
	}

	// Across several attempts an aggressive bot with top set must bet at least once
	betOrRaise := false
	for i := 0; i < 20 && !betOrRaise; i++ {
		move := state.getBotMove()
		assert.NotEqual(t, "FO", move, "bot must never fold top set with no bet facing")
		if move == "BL" || move == "BH" || move == "RL" || move == "RH" {
			betOrRaise = true
		}
	}
	assert.True(t, betOrRaise, "aggressive bot with a flopped set should bet or raise")
}

// TestBotAIPostFlopWeakHandFacingBet verifies a no-bluff bot folds air to a bet
func TestBotAIPostFlopWeakHandFacingBet(t *testing.T) {
	state := &GameState{
		Players: []Player{
			{Name: "Bot1", isBot: true, Status: STATUS_PLAYING, Purse: 1000, Bet: 0,
				Hand:    parseCards(t, "2C", "7D"), // Complete air on this board
				profile: BotProfile{Name: "Honest", VPIP: 0.5, PFR: 0.3, BluffFrequency: 0.0}},
			{Name: "Player2", isBot: false, Status: STATUS_PLAYING, Purse: 950, Bet: 50},
		},
		ActivePlayer:   0,
		currentBet:     50,
		Round:          2,
		CommunityCards: parseCards(t, "AH", "KC", "QD"),
		rng:            rand.New(rand.NewSource(1)),
	}

	assert.Equal(t, "FO", state.getBotMove(), "a non-bluffing bot should fold air facing a bet")
}

// TestBotAIAlwaysReturnsValidMove fuzzes the bot AI across many random states to
// guard against the historical getRank crash in post-flop evaluation
func TestBotAIAlwaysReturnsValidMove(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	for i := 0; i < 200; i++ {
		state := newBotTable(4, int64(i))
		state.newRound()

		// Advance to a random street so post-flop evaluation paths are exercised
		street := rng.Intn(3)
		for s := 0; s < street; s++ {
			state.dealCommunityCards(3 - min(s, 1)*2) // 3 then 1 then 1
			state.Round++
		}

		move := state.getBotMove()
		if move == "" {
			continue
		}
		valid := false
		for _, m := range state.getValidMoves() {
			if m.Move == move {
				valid = true
				break
			}
		}
		assert.True(t, valid, "bot move %q must be one of the valid moves (iteration %d)", move, i)
	}
}
