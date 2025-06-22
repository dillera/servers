import websocket
import json
import time
import os

WEBSOCKET_URL = "ws://localhost:8080/ws"

# --- Card Formatting ---
def format_card(card):
    """Formats a card dictionary into a human-readable string."""
    if not card or 'Rank' not in card or 'Suit' not in card:
        # Handle empty or malformed card data for folded players
        return '??'
    
    ranks = {11: 'J', 12: 'Q', 13: 'K', 14: 'A'}
    suits = {0: '♣', 1: '♦', 2: '♥', 3: '♠'}
    
    rank_val = card['Rank']
    suit_val = card['Suit']
    
    rank_str = ranks.get(rank_val, str(rank_val))
    suit_str = suits.get(suit_val, '?')
    
    return f"{rank_str}{suit_str}"

def print_game_state(game_state):
    """Clears the console and prints a formatted view of the game state."""
    if not game_state:
        return

    os.system('cls' if os.name == 'nt' else 'clear')

    games_played = game_state.get('gamesPlayed', 0)
    round_num = game_state.get('Round', 0)
    pot = game_state.get('Pot', 0)
    current_bet = game_state.get('currentBet', 0)
    community_cards = game_state.get('CommunityCards', [])
    players = game_state.get('Players', [])
    winner = game_state.get('Winner')

    # A non-empty Winner string from the server indicates the previous game just ended.
    # The round will be 0 as the server prepares for the next game.
    if winner and round_num == 0:
        print("\n--- GAME OVER ---")
        print(f"Game #{games_played} Concluded")
        print(f"Winner: {winner}\n")
        # Return early to avoid printing the empty table for the next game.
        return

    round_name = game_state.get('roundName', f'Round {round_num}')
    print(f"--- Texas Hold'em --- Game #{games_played} --- {round_name} ---")
    
    community_str = " ".join(format_card(c) for c in community_cards) if community_cards else "Not dealt"
    print(f"Pot: ${pot:<6} | Current Bet: ${current_bet:<6} | Community Cards: {community_str}")
    print("-" * 80)
    print(f"{'Player':<15} {'Status':<12} {'Purse':<10} {'Bet':<10} {'Hand'}")
    print("-" * 80)

    for player in players:
        name = player.get('Name', 'Unknown')
        status_map = {0: "Waiting", 1: "Playing", 2: "Folded", 3: "All-In"}
        status = status_map.get(player.get('Status'), "Unknown")
        purse = player.get('Purse', 0)
        bet = player.get('Bet', 0)
        hand = player.get('Hand', [])
        
        hand_str = " ".join(format_card(c) for c in hand) if hand else "Hidden"
        
        print(f"{name:<15} {status:<12} ${purse:<9} ${bet:<9} {hand_str}")

# --- WebSocket Callbacks ---
def on_message(ws, message):
    try:
        message_data = json.loads(message)
        if message_data.get('type') == 'gameState':
            game_state = message_data.get('data')
            print_game_state(game_state)
        else:
            print(f"--- Received Message: {message_data.get('type')} ---\n{message}")
    except json.JSONDecodeError:
        print(f"Error decoding JSON: {message}")
    except Exception as e:
        print(f"An error occurred in on_message: {e}")

def on_error(ws, error):
    print(f"### Error: {error} ###")

def on_close(ws, close_status_code, close_msg):
    print(f"### Closed: {close_status_code} - {close_msg} ###")

def on_open(ws):
    print("--- Connection Opened ---")

def main():
    print("Attempting to connect to server...")
    # Add a delay to give the server time to start up
    time.sleep(1)

    # websocket.enableTrace(True) # Detailed WebSocket logging is now disabled
    ws = websocket.WebSocketApp(WEBSOCKET_URL,
                              on_open=on_open,
                              on_message=on_message,
                              on_error=on_error,
                              on_close=on_close)

    while True:
        ws.run_forever()
        print("--- WebSocket connection closed. Reconnecting in 5 seconds... ---")
        time.sleep(5)

if __name__ == "__main__":
    main()
