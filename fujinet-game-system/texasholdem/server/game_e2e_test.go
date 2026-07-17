package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFullHandDealToShowdown plays one complete bot hand through the real engine
// and verifies phase order, board size, a winner, and chip conservation
func TestFullHandDealToShowdown(t *testing.T) {
	useFastTimers(t)

	// Search a few seeds for a hand that reaches showdown (bots may fold out)
	for seed := int64(1); seed <= 50; seed++ {
		state := newBotTable(4, seed)
		trace := playHand(t, state, 500)

		assert.Equal(t, 4*STARTING_PURSE, totalChips(state), "chips conserved (seed %d)", seed)
		assert.NotEmpty(t, state.Winner, "hand must produce a winner (seed %d)", seed)

		if !state.wonByFolds {
			// Reached showdown: full phase progression and 5 community cards
			assert.Equal(t, []string{"Pre-flop", "Flop", "Turn", "River", "Showdown"}, trace.phases,
				"phase order for showdown hand (seed %d)", seed)
			assert.Len(t, state.CommunityCards, 5, "5 community cards at showdown")
			assert.Contains(t, state.Winner, "won with", "showdown result names the hand")
			return
		}
	}
	t.Fatal("no seed in 1..50 produced a showdown hand")
}

// TestChipConservationAcrossHands plays many consecutive hands and asserts the
// total chip count never changes (catches pot double-counting and split leaks)
func TestChipConservationAcrossHands(t *testing.T) {
	useFastTimers(t)
	state := newBotTable(4, 99)

	expected := 4 * STARTING_PURSE
	hands := 0
	for iter := 0; iter < 20000 && hands < 15; iter++ {
		wasOver := state.gameOver
		state.RunGameLogic()
		if state.gameOver && !wasOver {
			hands++
			// Bots below 25 chips get refilled at next-hand start; account for that
			// by only asserting while nobody is that short
			assert.Equal(t, expected, totalChips(state), "chips conserved after hand %d", hands)
			for _, p := range state.Players {
				if p.Purse < 25 {
					t.Logf("stopping after hand %d: a bot is nearly busted and will be refilled", hands)
					return
				}
			}
		}
	}
	require.GreaterOrEqual(t, hands, 5, "should complete several hands")
}

// TestButtonRotationAndBlinds verifies the dealer button advances every hand and
// the blinds follow it
func TestButtonRotationAndBlinds(t *testing.T) {
	useFastTimers(t)
	state := newBotTable(3, 3)

	buttons := []int{}
	prevGames := 0
	for iter := 0; iter < 20000 && len(buttons) < 6; iter++ {
		state.RunGameLogic()
		if state.GamesPlayed > prevGames && state.Round >= 1 && !state.gameOver {
			prevGames = state.GamesPlayed
			buttons = append(buttons, state.buttonPos)

			// Blinds sit immediately left of the button
			n := len(state.Players)
			sb := (state.buttonPos + 1) % n
			bb := (state.buttonPos + 2) % n
			assert.True(t, strings.HasPrefix(state.Players[sb].Move, "POST"),
				"hand %d: seat after button posted SB", state.GamesPlayed)
			assert.True(t, strings.HasPrefix(state.Players[bb].Move, "POST"),
				"hand %d: second seat after button posted BB", state.GamesPlayed)
		}
	}
	require.Len(t, buttons, 6, "should observe 6 hands")

	for i := 1; i < len(buttons); i++ {
		assert.Equal(t, (buttons[i-1]+1)%3, buttons[i],
			"button must advance one seat between hand %d and %d (got %v)", i, i+1, buttons)
	}
}

// TestCorrectWinnerByHandRank rigs the deck so the winner is known and verifies
// the right player is paid
func TestCorrectWinnerByHandRank(t *testing.T) {
	useFastTimers(t)
	state := newBotTable(3, 1)

	// Seat 0 flops the nut flush; seat 1 gets two pair; seat 2 gets junk
	rigDeck(t, state,
		[][]string{{"AS", "KS"}, {"JC", "TC"}, {"7D", "2H"}},
		[]string{"QS", "JS", "TS", "3D", "2C"})

	// Make bots call everything down so the hand reaches showdown deterministically:
	// give them all a calling-station profile
	for i := range state.Players {
		state.Players[i].profile = BotProfile{Name: "Station", VPIP: 1.0, PFR: 0.0, BluffFrequency: 0.0}
	}

	winnerName := state.Players[0].Name
	playHand(t, state, 500)

	assert.False(t, state.wonByFolds, "rigged hand should reach showdown")
	assert.Contains(t, state.Winner, winnerName, "the royal flush must win")
	assert.Contains(t, state.Winner, "won with", "result should name the winning hand")
	assert.Equal(t, 3*STARTING_PURSE, totalChips(state), "chips conserved")

	// The winner cannot have lost chips, everyone else cannot have gained
	assert.GreaterOrEqual(t, state.Players[0].Purse, STARTING_PURSE)
	assert.LessOrEqual(t, state.Players[1].Purse, STARTING_PURSE)
	assert.LessOrEqual(t, state.Players[2].Purse, STARTING_PURSE)
}

// TestAllInSidePots rigs a three-way all-in with different stack sizes and
// verifies main pot / side pot distribution
func TestAllInSidePots(t *testing.T) {
	useFastTimers(t)

	t.Run("Short_Stack_Wins_Main_Pot_Only", func(t *testing.T) {
		state := newBotTable(3, 1)
		state.Players[0].Purse = 100  // short stack, best hand
		state.Players[1].Purse = 300  // middle stack, second-best hand
		state.Players[2].Purse = 1000 // big stack, worst hand

		// Short stack gets quad aces, middle gets a flush, big stack junk
		rigDeck(t, state,
			[][]string{{"AC", "AD"}, {"KH", "QH"}, {"7D", "2S"}},
			[]string{"AH", "AS", "9H", "5H", "3C"})

		state.newRound()

		// Drive everyone all-in manually via valid moves
		for state.countByStatus(STATUS_PLAYING) > 0 && !state.gameOver {
			allin := findMove(state.getValidMoves(), "AI")
			require.NotEmpty(t, allin, "all-in must be available")
			require.True(t, state.performMove("AI", true))
			if state.ActivePlayer < 0 {
				break
			}
		}
		state.RunGameLogic()
		require.True(t, state.gameOver, "hand must complete after everyone is all-in")

		// Main pot: 3 x 100 = 300 -> quads (seat 0)
		// Side pot 1: 2 x 200 = 400 -> flush (seat 1)
		// Side pot 2: 1 x 700 = 700 -> returned to seat 2 (only contributor)
		assert.Equal(t, 300, state.Players[0].Purse, "short stack wins only the main pot")
		assert.Equal(t, 400, state.Players[1].Purse, "middle stack wins the side pot")
		assert.Equal(t, 700, state.Players[2].Purse, "big stack gets uncontested excess back")
		assert.Equal(t, 1400, totalChips(state), "chips conserved")
	})

	t.Run("Big_Stack_Wins_Everything", func(t *testing.T) {
		state := newBotTable(3, 1)
		state.Players[0].Purse = 100
		state.Players[1].Purse = 300
		state.Players[2].Purse = 1000

		// Big stack gets the quads this time
		rigDeck(t, state,
			[][]string{{"KH", "QH"}, {"7D", "2S"}, {"AC", "AD"}},
			[]string{"AH", "AS", "9H", "5H", "3C"})

		state.newRound()
		for state.countByStatus(STATUS_PLAYING) > 0 && !state.gameOver {
			require.True(t, state.performMove("AI", true))
			if state.ActivePlayer < 0 {
				break
			}
		}
		state.RunGameLogic()
		require.True(t, state.gameOver)

		assert.Equal(t, 0, state.Players[0].Purse, "short stack busts")
		assert.Equal(t, 0, state.Players[1].Purse, "middle stack busts")
		assert.Equal(t, 1400, state.Players[2].Purse, "big stack wins main pot, side pot, and excess")
		assert.Equal(t, 1400, totalChips(state), "chips conserved")
	})
}

// TestAllInRunout verifies that when all players are all-in before the river the
// board is automatically run out to showdown
func TestAllInRunout(t *testing.T) {
	useFastTimers(t)
	state := newBotTable(2, 1)
	rigDeck(t, state,
		[][]string{{"AC", "AD"}, {"KH", "KD"}},
		[]string{"9H", "5S", "3C", "2D", "7C"})
	state.newRound()

	// Both players shove pre-flop
	require.True(t, state.performMove("AI", true))
	require.True(t, state.performMove("AI", true))
	state.RunGameLogic()

	assert.True(t, state.gameOver, "hand completes without further input")
	assert.Len(t, state.CommunityCards, 5, "board fully run out")
	assert.Contains(t, state.Winner, state.Players[0].Name, "aces hold up on this board")
	assert.Equal(t, 2*STARTING_PURSE, totalChips(state), "chips conserved")
}

// TestBotsPlayManyHandsWithoutCrashing runs a 6-bot table for many hands as a
// guard against evaluator crashes (the historical getRank panic)
func TestBotsPlayManyHandsWithoutCrashing(t *testing.T) {
	useFastTimers(t)
	state := newBotTable(6, 12345)

	hands := 0
	sawLateStreet := false
	for iter := 0; iter < 50000 && hands < 10; iter++ {
		wasOver := state.gameOver
		state.RunGameLogic()
		if state.Round >= 3 && !state.gameOver {
			sawLateStreet = true
		}
		if state.gameOver && !wasOver {
			hands++
			assert.NotEmpty(t, state.Winner, "hand %d has a winner", hands)
		}
	}
	require.GreaterOrEqual(t, hands, 10, "should complete 10 hands")
	assert.True(t, sawLateStreet, "at least one hand should reach the turn or river")
}

// TestHumanTimeoutFolds documents that a human who exceeds the move timer is
// folded automatically
func TestHumanTimeoutFolds(t *testing.T) {
	useFastTimers(t)
	PLAYER_TIME_LIMIT = -1 // Human timer already expired

	state := newBotTable(2, 4)
	state.Players = append(state.Players, Player{Name: "Human", Status: STATUS_WAITING, Purse: STARTING_PURSE, lastPing: state.Players[0].lastPing})
	// Bots must not fold, so the hand is still live when the human's turn comes
	for i := range state.Players {
		if state.Players[i].isBot {
			state.Players[i].profile = BotProfile{Name: "Station", VPIP: 1.0, PFR: 0.0, BluffFrequency: 0.0}
		}
	}
	state.newRound()

	humanSeat := -1
	for i, p := range state.Players {
		if p.Name == "Human" {
			humanSeat = i
			break
		}
	}
	require.GreaterOrEqual(t, humanSeat, 0)

	// Play forward until it's the human's turn, then let the engine time them out
	for iter := 0; iter < 1000; iter++ {
		if state.ActivePlayer == humanSeat && state.Players[humanSeat].Status == STATUS_PLAYING {
			state.RunGameLogic()
			assert.Equal(t, "FOLD", state.Players[humanSeat].Move, "timed-out human is folded")
			assert.Equal(t, STATUS_FOLDED, state.Players[humanSeat].Status)
			return
		}
		if state.gameOver {
			break
		}
		state.RunGameLogic()
	}
	t.Fatal("human never became the active player")
}

// TestPotArithmeticSanity verifies pot equals the sum of all contributions mid-hand
func TestPotArithmeticSanity(t *testing.T) {
	useFastTimers(t)
	state := newBotTable(4, 8)
	state.newRound()

	// After blinds: pot == SB+BB and equals sum of totalBets
	sum := 0
	for _, p := range state.Players {
		sum += p.totalBet
	}
	assert.Equal(t, state.Pot, sum, "pot equals sum of contributions after blinds")

	// Run several engine steps mid-hand and re-verify
	for i := 0; i < 5 && !state.gameOver; i++ {
		state.RunGameLogic()
		sum = 0
		for _, p := range state.Players {
			sum += p.totalBet
		}
		assert.Equal(t, state.Pot, sum, "pot equals sum of contributions (step %d)", i)
	}
}
