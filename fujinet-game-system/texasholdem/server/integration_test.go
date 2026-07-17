package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	initializeGameServer()
	// The hub tick drives game logic on a wall-clock timer; keep it off in tests
	// (the dedicated hub test re-enables it)
	hubTickerEnabled = false
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Multi-client test framework
//
// A simClient is a simulated FujiNet client that talks to the real HTTP API the
// same way hardware clients do: poll GET /state?table=T&player=P, and when it is
// this client's turn (ActivePlayer == 0 with ValidMoves present), submit a move
// via GET /move/<move>?table=T&player=P chosen by its policy.
// ---------------------------------------------------------------------------

// clientStateView mirrors the compact JSON the server sends to clients (the
// original 8-bit client wire format: single-character lower-case keys)
type clientPlayerView struct {
	Name   string `json:"n"`
	Status int    `json:"s"`
	Bet    int    `json:"b"`
	Move   string `json:"m"`
	Purse  int    `json:"p"`
	Hand   string `json:"h"`
}

type clientStateView struct {
	LastResult   string             `json:"l"`
	Round        int                `json:"r"`
	Pot          int                `json:"p"`
	ActivePlayer int                `json:"a"`
	MoveTime     int                `json:"m"`
	Viewing      int                `json:"v"`
	Community    string             `json:"c"`
	ValidMoves   []validMove        `json:"vm"`
	Players      []clientPlayerView `json:"pl"`
	Hash         string             `json:"z"`
}

type simClient struct {
	base   string
	table  string
	name   string
	client *http.Client
	policy func(moves []validMove) string

	mu    sync.Mutex
	views []clientStateView // every state observed, for post-run assertions
	moves []string          // every move submitted
}

func newSimClient(base, table, name string, policy func([]validMove) string) *simClient {
	return &simClient{base: base, table: table, name: name, client: &http.Client{Timeout: 5 * time.Second}, policy: policy}
}

func (sc *simClient) get(path string) ([]byte, error) {
	resp, err := sc.client.Get(sc.base + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", path, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (sc *simClient) pollState() (*clientStateView, error) {
	body, err := sc.get(fmt.Sprintf("/state?table=%s&player=%s", sc.table, url.QueryEscape(sc.name)))
	if err != nil {
		return nil, err
	}
	view := &clientStateView{}
	if err := json.Unmarshal(body, view); err != nil {
		return nil, fmt.Errorf("bad /state response %q: %w", string(body), err)
	}
	sc.mu.Lock()
	sc.views = append(sc.views, *view)
	sc.mu.Unlock()
	return view, nil
}

func (sc *simClient) submitMove(move string) error {
	_, err := sc.get(fmt.Sprintf("/move/%s?table=%s&player=%s", url.PathEscape(move), sc.table, url.QueryEscape(sc.name)))
	if err == nil {
		sc.mu.Lock()
		sc.moves = append(sc.moves, move)
		sc.mu.Unlock()
	}
	return err
}

// run polls and plays until the context is cancelled
func (sc *simClient) run(ctx context.Context) error {
	for ctx.Err() == nil {
		view, err := sc.pollState()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("client %s: %w", sc.name, err)
		}
		if view.ActivePlayer == 0 && len(view.ValidMoves) > 0 {
			if err := sc.submitMove(sc.policy(view.ValidMoves)); err != nil && ctx.Err() == nil {
				return fmt.Errorf("client %s move: %w", sc.name, err)
			}
		}
		time.Sleep(2 * time.Millisecond)
	}
	return nil
}

// Policies

func policyCallAny(moves []validMove) string {
	if m := findMove(moves, "CH"); m != "" {
		return m
	}
	if m := findMove(moves, "CA"); m != "" {
		return m
	}
	if m := findMove(moves, "AI"); m != "" {
		return m
	}
	return "FO"
}

func policyFoldUnlessFree(moves []validMove) string {
	if m := findMove(moves, "CH"); m != "" {
		return m
	}
	return "FO"
}

func policyAllIn(moves []validMove) string {
	if m := findMove(moves, "AI"); m != "" {
		return m
	}
	return policyCallAny(moves)
}

// Harness helpers

// useIntegrationTimers makes bots act instantly but holds finished hands open so
// the test can inspect the result before explicitly starting the next hand.
func useIntegrationTimers(t *testing.T) {
	t.Helper()
	useFastTimers(t)
	ENDGAME_TIME_LIMIT = time.Hour
}

// newHTTPTable creates an isolated test table and an httptest server for it
func newHTTPTable(t *testing.T, botCount int, seed int64) (*httptest.Server, string) {
	t.Helper()
	tableId := strings.ToLower(fmt.Sprintf("it-%s-%d", strings.ReplaceAll(t.Name(), "/", "-"), seed))
	state := createGameState(botCount, false)
	state.TableId = tableId
	state.serverName = tableId
	state.rng = rand.New(rand.NewSource(seed))
	stateMap.Store(tableId, state)

	server := httptest.NewServer(setupRouter())
	t.Cleanup(server.Close)
	return server, tableId
}

// withTable runs fn with the table's state under the table mutex
func withTable(tableId string, fn func(state *GameState)) {
	unlock := tableMutex.Lock(tableId)
	defer unlock()
	value, _ := stateMap.Load(tableId)
	fn(value.(*GameState))
}

// waitForHandEnd blocks until the current hand completes (gameOver true) and
// returns a snapshot taken under the table lock
func waitForHandEnd(t *testing.T, tableId string, timeout time.Duration) (winner string, chips int, players int, games int) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		done := false
		withTable(tableId, func(state *GameState) {
			if state.gameOver {
				done = true
				winner = state.Winner
				chips = totalChips(state)
				players = len(state.Players)
				games = state.GamesPlayed
			}
		})
		if done {
			return winner, chips, players, games
		}
		time.Sleep(3 * time.Millisecond)
	}
	t.Fatalf("hand on table %s did not complete within %v", tableId, timeout)
	return
}

// forceNextHand releases the endgame timer so the next poll starts a new hand
func forceNextHand(tableId string) {
	withTable(tableId, func(state *GameState) {
		state.moveExpires = time.Now().Add(-time.Second)
	})
}

// startClients launches the given clients concurrently and returns a stop
// function that cancels them and reports any client error
func startClients(t *testing.T, clients ...*simClient) (stop func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, len(clients))
	var wg sync.WaitGroup
	for _, sc := range clients {
		wg.Add(1)
		go func(sc *simClient) {
			defer wg.Done()
			errCh <- sc.run(ctx)
		}(sc)
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			cancel()
			wg.Wait()
			close(errCh)
			for err := range errCh {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

// TestHTTPSingleClientJoinsAndPlaysHand: one HTTP client at a bot table plays a
// complete hand through the real API and the JSON contract holds
func TestHTTPSingleClientJoinsAndPlaysHand(t *testing.T) {
	useIntegrationTimers(t)
	server, tableId := newHTTPTable(t, 3, 21)

	client := newSimClient(server.URL, tableId, "TestPlayer", policyCallAny)
	stop := startClients(t, client)
	defer stop()

	winner, chips, players, _ := waitForHandEnd(t, tableId, 30*time.Second)
	assert.NotEmpty(t, winner, "hand must produce a winner")
	assert.Equal(t, players*STARTING_PURSE, chips, "chips conserved through an HTTP-driven hand")
	stop()

	// JSON contract checks over everything the client observed
	client.mu.Lock()
	defer client.mu.Unlock()
	require.NotEmpty(t, client.views)
	sawOwnCards := false
	for _, view := range client.views {
		require.NotEmpty(t, view.Players, "state always includes players")
		assert.Equal(t, "TESTPLAYER", strings.ToUpper(view.Players[0].Name), "client is always pl[0] in its own view")

		// Other players' hole cards must be masked ("??" per card) while the hand
		// is live; they are only revealed at showdown (round 5)
		if view.Round >= 1 && view.Round <= 4 {
			for i := 1; i < len(view.Players); i++ {
				assert.Empty(t, strings.ReplaceAll(view.Players[i].Hand, "?", ""),
					"opponent hole cards must be masked during play, got %q", view.Players[i].Hand)
			}
		}
		// Own hand arrives as a 4-char card string like "KSKH"
		if len(view.Players[0].Hand) == 4 && !strings.Contains(view.Players[0].Hand, "?") {
			sawOwnCards = true
		}
		if view.ActivePlayer != 0 {
			assert.Empty(t, view.ValidMoves, "ValidMoves only present on the client's turn")
		}
	}
	assert.True(t, sawOwnCards, "client must see its own hole cards during the hand")
	assert.NotEmpty(t, client.moves, "client should have made at least one move")
}

// TestHTTPStateHashShortCircuit verifies the ?hash= optimization: when the state
// is unchanged the server responds with just "1"
func TestHTTPStateHashShortCircuit(t *testing.T) {
	useIntegrationTimers(t)
	server, tableId := newHTTPTable(t, 0, 3) // no bots: state is static while waiting

	client := newSimClient(server.URL, tableId, "Solo", nil)
	view, err := client.pollState()
	require.NoError(t, err)
	require.NotEmpty(t, view.Hash, "state JSON must expose the hash as z")

	body, err := client.get(fmt.Sprintf("/state?table=%s&player=Solo&hash=%s", tableId, view.Hash))
	require.NoError(t, err)
	assert.Equal(t, `"1"`, strings.TrimSpace(string(body)), "unchanged state with matching hash returns \"1\"")

	body, err = client.get(fmt.Sprintf("/state?table=%s&player=Solo&hash=stale", tableId))
	require.NoError(t, err)
	assert.NotEqual(t, `"1"`, strings.TrimSpace(string(body)), "stale hash returns the full state")
}

// TestHTTPMultiClientFullHands is the headline test: three concurrent HTTP
// clients with different strategies join a bot table and play several complete
// hands end-to-end
func TestHTTPMultiClientFullHands(t *testing.T) {
	useIntegrationTimers(t)
	server, tableId := newHTTPTable(t, 2, 31)

	alice := newSimClient(server.URL, tableId, "Alice", policyCallAny)
	bob := newSimClient(server.URL, tableId, "Bob", policyCallAny)
	carol := newSimClient(server.URL, tableId, "Carol", policyFoldUnlessFree)
	stop := startClients(t, alice, bob, carol)
	defer stop()

	const handsToPlay = 3
	buttons := []int{}
	for hand := 1; hand <= handsToPlay; hand++ {
		winner, chips, players, games := waitForHandEnd(t, tableId, 30*time.Second)
		assert.NotEmpty(t, winner, "hand %d must produce a winner", hand)
		assert.Equal(t, players*STARTING_PURSE, chips, "chips conserved after hand %d", hand)
		t.Logf("hand %d (game #%d) complete: %s", hand, games, winner)

		withTable(tableId, func(state *GameState) {
			buttons = append(buttons, state.buttonPos)
			// Refills would break the conservation assertion; with these policies
			// nobody should get near busting in 3 hands
			for _, p := range state.Players {
				assert.GreaterOrEqual(t, p.Purse, 25, "no player should be near busting")
			}
		})

		if hand < handsToPlay {
			forceNextHand(tableId)
			// Wait until the next hand actually starts before watching for its end
			deadline := time.Now().Add(10 * time.Second)
			for started := false; !started && time.Now().Before(deadline); {
				withTable(tableId, func(state *GameState) { started = !state.gameOver && state.Round >= 1 })
				time.Sleep(3 * time.Millisecond)
			}
		}
	}

	// The dealer button must have moved between hands
	distinct := map[int]bool{}
	for _, b := range buttons {
		distinct[b] = true
	}
	assert.Greater(t, len(distinct), 1, "dealer button must rotate across hands (saw %v)", buttons)

	// All three clients participated (joined and observed the game)
	for _, sc := range []*simClient{alice, bob, carol} {
		sc.mu.Lock()
		assert.NotEmpty(t, sc.views, "client %s observed game state", sc.name)
		sc.mu.Unlock()
	}
	assert.NotEmpty(t, alice.moves, "Alice should have acted in the hands")
}

// TestHTTPAllInScenario: a client that shoves all-in plays a complete hand and
// chip accounting survives
func TestHTTPAllInScenario(t *testing.T) {
	useIntegrationTimers(t)
	server, tableId := newHTTPTable(t, 2, 17)

	shover := newSimClient(server.URL, tableId, "Shover", policyAllIn)
	stop := startClients(t, shover)
	defer stop()

	winner, chips, players, _ := waitForHandEnd(t, tableId, 30*time.Second)
	assert.NotEmpty(t, winner)
	assert.Equal(t, players*STARTING_PURSE, chips, "chips conserved after an all-in hand")
}

// TestHTTPLeaveMidHand: one of two clients leaves mid-hand; the game still
// completes for the remaining players and no chips are lost
func TestHTTPLeaveMidHand(t *testing.T) {
	useIntegrationTimers(t)
	server, tableId := newHTTPTable(t, 2, 13)

	stayer := newSimClient(server.URL, tableId, "Stayer", policyCallAny)
	leaver := newSimClient(server.URL, tableId, "Leaver", policyCallAny)

	// Join both clients before the first hand starts (a /move call joins the
	// table but, unlike /state, does not advance the game)
	_, err := leaver.get(fmt.Sprintf("/move/CH?table=%s&player=Leaver", tableId))
	require.NoError(t, err)
	_, err = stayer.get(fmt.Sprintf("/move/CH?table=%s&player=Stayer", tableId))
	require.NoError(t, err)

	stopStayer := startClients(t, stayer)
	defer stopStayer()
	stopLeaver := startClients(t, leaver)
	defer stopLeaver()

	// Wait until the hand is underway with both humans dealt in
	deadline := time.Now().Add(10 * time.Second)
	for ready := false; !ready; {
		require.True(t, time.Now().Before(deadline), "hand with both humans never started")
		withTable(tableId, func(state *GameState) {
			if state.Round >= 1 && !state.gameOver {
				playing := 0
				for _, p := range state.Players {
					if !p.isBot && (p.Status == STATUS_PLAYING || p.Status == STATUS_ALL_IN) {
						playing++
					}
				}
				ready = playing == 2
			}
		})
		time.Sleep(3 * time.Millisecond)
	}

	// Stop the leaver's polling first so it cannot rejoin after leaving
	stopLeaver()
	_, err = leaver.get(fmt.Sprintf("/leave?table=%s&player=Leaver", tableId))
	require.NoError(t, err)

	winner, chips, _, _ := waitForHandEnd(t, tableId, 30*time.Second)
	assert.NotEmpty(t, winner, "hand completes after a player leaves")
	// The leaver is still seated (dropped only at next hand start), so the full
	// chip total must be intact: 2 bots + 2 humans
	assert.Equal(t, 4*STARTING_PURSE, chips, "chips conserved after a mid-hand leave")

	withTable(tableId, func(state *GameState) {
		for _, p := range state.Players {
			if p.Name == "Leaver" {
				assert.Equal(t, STATUS_LEFT, p.Status, "leaver is marked as left")
			}
		}
	})
}

// TestHTTPWithHubTicker runs a full hand with the websocket hub's background
// ticker enabled, validating (under -race) that the hub and HTTP handlers can
// safely drive the same table concurrently
func TestHTTPWithHubTicker(t *testing.T) {
	useIntegrationTimers(t)
	hubTickerEnabled = true
	t.Cleanup(func() { hubTickerEnabled = false })

	server, tableId := newHTTPTable(t, 3, 27)
	client := newSimClient(server.URL, tableId, "HubPlayer", policyCallAny)
	stop := startClients(t, client)
	defer stop()

	winner, chips, players, _ := waitForHandEnd(t, tableId, 45*time.Second)
	assert.NotEmpty(t, winner)
	assert.Equal(t, players*STARTING_PURSE, chips, "chips conserved with hub ticker running")
}
