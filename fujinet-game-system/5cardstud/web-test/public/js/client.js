// client.js

const socket = new WebSocket('ws://localhost:8080/ws'); // Connect to the Go server's WebSocket port

const startGameButton = document.getElementById('start-game-button');

startGameButton.addEventListener('click', () => {
    socket.send(JSON.stringify({ type: 'startGame' }));
    startGameButton.disabled = true; // Disable button after starting game
});

// DOM Elements
const communityCardsDiv = document.getElementById('community-cards').querySelector('.card-display');
const gameRoundSpan = document.getElementById('game-round');
const gamePotSpan = document.getElementById('game-pot');
const gameCurrentBetSpan = document.getElementById('game-current-bet');
const gameActivePlayerSpan = document.getElementById('game-active-player');



// Helper to render cards
function renderCards(cardDisplayElement, cards, isHidden = false) {
    cardDisplayElement.innerHTML = '';
    if (cards) {
        cards.forEach(card => {
            const cardDiv = document.createElement('div');
            cardDiv.classList.add('card');
            if (isHidden) {
                cardDiv.classList.add('hidden');
                cardDiv.textContent = '🂠'; // Card back symbol
            } else {
                // Assuming card is an object like { Rank: 14, Suit: 3 } (Ace of Spades)
                // Convert to displayable string (e.g., "A♠")
                const rankMap = { 14: 'A', 13: 'K', 12: 'Q', 11: 'J', 10: 'T' };
                const suitMap = { 0: '♣', 1: '♦', 2: '♥', 3: '♠' };
                const rank = rankMap[card.Rank] || card.Rank;
                const suit = suitMap[card.Suit];
                cardDiv.textContent = `${rank}${suit}`;
            }
            cardDisplayElement.appendChild(cardDiv);
        });
    }
}

// Function to update the UI based on game state
function updateUI(gameState) {
    console.log('Received game state:', gameState);

    // Update game info
    gameRoundSpan.textContent = gameState.Round;
    gamePotSpan.textContent = gameState.Pot;
    gameCurrentBetSpan.textContent = gameState.currentBet;
    gameActivePlayerSpan.textContent = gameState.ActivePlayer;

    // Update community cards
    renderCards(communityCardsDiv, gameState.CommunityCards);

    // Update player info dynamically
    const playersContainer = document.getElementById('players-container');
    playersContainer.innerHTML = ''; // Clear existing player elements

    console.log('Players array from gameState:', gameState.Players);
    gameState.Players.forEach((player, index) => {
        const playerArea = document.createElement('div');
        playerArea.id = `player-${index}`;
        playerArea.classList.add('player-area');
        if (player.IsHuman) { // Assuming server sends IsHuman flag
            playerArea.classList.add('human-player');
        }

        // Create and append elements programmatically
        const h3 = document.createElement('h3');
        h3.textContent = player.Name;
        playerArea.appendChild(h3);

        const statusP = document.createElement('p');
        statusP.innerHTML = `Status: <span class="player-status"></span>`;
        playerArea.appendChild(statusP);

        const purseP = document.createElement('p');
        purseP.innerHTML = `Purse: <span class="player-purse"></span>`;
        playerArea.appendChild(purseP);

        const betP = document.createElement('p');
        betP.innerHTML = `Bet: <span class="player-bet"></span>`;
        playerArea.appendChild(betP);

        const playerCardsDiv = document.createElement('div');
        playerCardsDiv.classList.add('player-cards', 'card-display');
        playerArea.appendChild(playerCardsDiv);

        const playerActionsDiv = document.createElement('div');
        playerActionsDiv.classList.add('player-actions');
        playerActionsDiv.style.display = 'none'; // Initially hidden

        const foldButton = document.createElement('button');
        foldButton.classList.add('action-button');
        foldButton.dataset.move = 'FO';
        foldButton.textContent = 'Fold';
        playerActionsDiv.appendChild(foldButton);

        const checkButton = document.createElement('button');
        checkButton.classList.add('action-button');
        checkButton.dataset.move = 'CH';
        checkButton.textContent = 'Check';
        playerActionsDiv.appendChild(checkButton);

        const callButton = document.createElement('button');
        callButton.classList.add('action-button');
        callButton.dataset.move = 'CALL';
        callButton.textContent = 'Call';
        playerActionsDiv.appendChild(callButton);

        const betButton = document.createElement('button');
        betButton.classList.add('action-button');
        betButton.dataset.move = 'BET';
        betButton.textContent = 'Bet';
        playerActionsDiv.appendChild(betButton);

        const betAmountInput = document.createElement('input');
        betAmountInput.type = 'number';
        betAmountInput.classList.add('bet-amount');
        betAmountInput.placeholder = 'Amount';
        playerActionsDiv.appendChild(betAmountInput);

        playerArea.appendChild(playerActionsDiv);
        playersContainer.appendChild(playerArea);

        // Update player data
        statusP.querySelector('.player-status').textContent = player.Status === 1 ? 'Playing' : (player.Status === 2 ? 'Folded' : 'Left');
        purseP.querySelector('.player-purse').textContent = player.Purse;
        betP.querySelector('.player-bet').textContent = player.Bet;

        if (player.IsHuman) { // Assuming server sends IsHuman flag for the human player
            renderCards(playerCardsDiv, player.Hand);
        } else {
            renderCards(playerCardsDiv, player.Hand, true); // Hide bot's cards
        }

        // Handle action buttons for the active human player
        if (index === gameState.ActivePlayer && player.IsHuman) {
            playerActionsDiv.style.display = 'flex';
        } else {
            playerActionsDiv.style.display = 'none';
        }

        // Attach event listeners for action buttons after they are in the DOM
        playerActionsDiv.querySelectorAll('.action-button').forEach(button => {
            button.addEventListener('click', (event) => {
                const moveType = event.target.dataset.move;
                let moveString = moveType;

                if (moveType === 'BET' || moveType === 'CALL') {
                    const amount = betAmountInput.value;
                    if (amount) {
                        moveString = `${moveType} ${amount}`;
                    } else if (moveType === 'CALL') {
                        moveString = 'CALL';
                    }
                }
                console.log('Sending move:', moveString);
                socket.send(JSON.stringify({ type: 'playerMove', data: moveString }));
            });
        });
    });
}

// WebSocket Event Listeners
socket.onopen = (event) => {
    console.log('WebSocket: Connected to server', event);
    // Request initial game state or start game
    // Only send startGame if the button is not disabled (i.e., first connection)
    if (!startGameButton.disabled) {
        socket.send(JSON.stringify({ type: 'startGame' }));
        startGameButton.disabled = true; // Disable button after starting game
    }
};

socket.onmessage = (event) => {
    console.log('WebSocket: Received raw message:', event.data);
    try {
        const message = JSON.parse(event.data);
        console.log('Received message:', message);
        // Handle different message types from the server
        if (message.type === 'gameState') {
            console.log('Parsed game state data:', message.data); // Log the parsed game state
            updateUI(message.data);
        } else if (message.type === 'gameStarted') {
            console.log('Game started confirmation received!');
            // You might want to re-enable the button or change UI here
            startGameButton.disabled = false; // Re-enable for testing purposes
        } else if (message.type === 'playerAction') {
            // Handle player action updates
            console.log('Player action received:', message.data);
        }
    } catch (e) {
        console.error('WebSocket: Error parsing message:', e, event.data);
    }
};

socket.onclose = (event) => {
    console.log('WebSocket: Disconnected from server', event);
    console.log('WebSocket Close Code:', event.code);
    console.log('WebSocket Close Reason:', event.reason);
    console.log('WebSocket Was Clean:', event.wasClean);
};

socket.onerror = (error) => {
    console.error('WebSocket: Error', error);
};

// Handle player actions
document.querySelectorAll('.action-button').forEach(button => {
    button.addEventListener('click', () => {
        const moveType = button.dataset.move;
        let moveString = moveType;

        if (moveType === 'BET' || moveType === 'CALL') { // Assuming CALL can also have an amount for raising
            const betAmountInput = button.parentNode.querySelector('.bet-amount');
            const amount = betAmountInput.value;
            if (amount) {
                moveString = `${moveType} ${amount}`;
            } else if (moveType === 'CALL') {
                // If no amount specified for CALL, assume it's to match currentBet
                // The server will handle the actual amount based on currentBet
                moveString = 'CALL';
            }
        }
        console.log('Sending move:', moveString);
        socket.send(JSON.stringify({ type: 'playerMove', data: moveString }));
    });
});
