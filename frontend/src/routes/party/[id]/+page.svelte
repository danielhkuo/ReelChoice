<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import ParticipantList from '$lib/components/ParticipantList.svelte';

	// Types
	interface Participant {
		id: string;
		username: string;
		isHost: boolean;
		joinedAt: string;
	}

	interface Party {
		id: string;
		name: string;
		hostId: string;
		state: string;
		participants: Participant[];
		createdAt: string;
	}

	// State
	let party: Party | null = null;
	let currentUser: Participant | null = null;
	let ws: WebSocket | null = null;
	let isLoading = true;
	let error = '';
	
	// Join modal state
	let showJoinModal = false;
	let username = '';
	let isJoining = false;
	let joinError = '';

	const partyId = $page.params.id;

	onMount(async () => {
		await loadParty();
	});

	onDestroy(() => {
		if (ws) {
			ws.close();
		}
	});

	async function loadParty() {
		try {
			const response = await fetch(`/api/party/${partyId}`);
			if (response.ok) {
				party = await response.json();
				
				// Check if we need to show join modal
				// For now, always show join modal if no current user
				// In a real app, this would check session/authentication
				if (!currentUser) {
					showJoinModal = true;
				} else {
					connectWebSocket();
				}
			} else if (response.status === 404) {
				error = 'Party not found';
			} else {
				error = 'Failed to load party';
			}
		} catch (err) {
			error = 'Network error. Please check if the backend is running.';
			console.error('Error loading party:', err);
		} finally {
			isLoading = false;
		}
	}

	async function joinParty() {
		if (!username.trim()) {
			joinError = 'Please enter a username';
			return;
		}

		isJoining = true;
		joinError = '';

		try {
			const response = await fetch(`/api/party/${partyId}/join`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					username: username.trim()
				})
			});

			const data = await response.json();

			if (response.ok) {
				currentUser = data.participant;
				party = data.party;
				showJoinModal = false;
				connectWebSocket();
				
				// Broadcast user joined to other clients
				if (ws && ws.readyState === WebSocket.OPEN) {
					ws.send(JSON.stringify({
						type: 'user_joined',
						data: party,
						timestamp: Date.now()
					}));
				}
			} else {
				joinError = data.error || 'Failed to join party';
			}
		} catch (err) {
			joinError = 'Network error. Please try again.';
			console.error('Error joining party:', err);
		} finally {
			isJoining = false;
		}
	}

	function connectWebSocket() {
		if (!party || !currentUser) return;

		try {
			const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
			const wsUrl = `${protocol}//${window.location.host}/ws?partyId=${party.id}`;
			
			ws = new WebSocket(wsUrl);

			ws.onopen = () => {
				console.log('WebSocket connected');
			};

			ws.onmessage = (event) => {
				try {
					const message = JSON.parse(event.data);
					handleWebSocketMessage(message);
				} catch (error) {
					console.error('Error parsing WebSocket message:', error);
				}
			};

			ws.onerror = (error) => {
				console.error('WebSocket error:', error);
			};

			ws.onclose = (event) => {
				console.log('WebSocket closed:', event.code, event.reason);
				// TODO: Implement reconnection logic
			};

		} catch (error) {
			console.error('WebSocket connection error:', error);
		}
	}

	function handleWebSocketMessage(message: any) {
		switch (message.type) {
			case 'participant_update':
				// Update party data when participants change
				if (message.data && message.data.participants) {
					party = { ...party!, participants: message.data.participants };
				}
				break;
			
			case 'pong':
				console.log('Received pong from server');
				break;
				
			default:
				console.log('Unknown message type:', message.type);
		}
	}

	function handleJoinKeyPress(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			joinParty();
		}
	}
</script>

<svelte:head>
	<title>{party ? `${party.name} - ReelChoice` : 'Loading Party - ReelChoice'}</title>
</svelte:head>

{#if isLoading}
	<div class="loading">
		<h1>Loading party...</h1>
	</div>
{:else if error}
	<div class="error-container">
		<h1>😞 Oops!</h1>
		<p>{error}</p>
		<button on:click={() => goto('/')} class="home-button">
			← Back to Home
		</button>
	</div>
{:else if party}
	<div class="party-container">
		<header class="party-header">
			<div class="party-info">
				<h1>{party.name}</h1>
				<p class="party-id">Party ID: {party.id}</p>
				<div class="party-state">
					<span class="state-badge state-{party.state}">{party.state.toUpperCase()}</span>
				</div>
			</div>
			<nav class="header-nav">
				<a href="/" class="nav-link">← Home</a>
				<a href="/diagnostic" class="nav-link">🔧 Diagnostic</a>
			</nav>
		</header>

		{#if currentUser}
			<div class="welcome-message">
				<p>Welcome, <strong>{currentUser.username}</strong>! 
				{#if currentUser.isHost}
					<span class="host-badge">👑 Host</span>
				{/if}
				</p>
			</div>

			<div class="party-content">
				<div class="participants-section">
					<ParticipantList participants={party.participants} />
				</div>

				<div class="main-content">
					<div class="phase-info">
						<h2>🏛️ Lobby Phase</h2>
						<p>Waiting for all participants to join before starting movie nominations.</p>
						
						{#if currentUser.isHost}
							<div class="host-controls">
								<p><strong>As the host, you can:</strong></p>
								<ul>
									<li>Wait for more people to join</li>
									<li>Start the nomination phase when ready</li>
									<li>Control the flow of the party</li>
								</ul>
								<button class="action-button" disabled>
									Start Nominations (Coming Soon)
								</button>
							</div>
						{:else}
							<div class="participant-info">
								<p>The host will start the nomination phase when everyone is ready.</p>
							</div>
						{/if}
					</div>
				</div>
			</div>
		{/if}
	</div>
{/if}

<!-- Join Modal -->
{#if showJoinModal}
	<div class="modal-overlay">
		<div class="modal">
			<h2>Join "{party?.name}"</h2>
			<p>Enter your username to join this movie selection party.</p>
			
			<div class="form-group">
				<input
					type="text"
					bind:value={username}
					placeholder="Enter your username"
					on:keypress={handleJoinKeyPress}
					disabled={isJoining}
					class="username-input"
					maxlength="20"
					autocomplete="off"
				/>
				<button 
					on:click={joinParty} 
					disabled={isJoining || !username.trim()}
					class="join-button"
				>
					{isJoining ? 'Joining...' : 'Join Party'}
				</button>
			</div>
			
			{#if joinError}
				<div class="error">{joinError}</div>
			{/if}
		</div>
	</div>
{/if}

<style>
	.loading {
		display: flex;
		justify-content: center;
		align-items: center;
		min-height: 100vh;
		text-align: center;
	}

	.error-container {
		display: flex;
		flex-direction: column;
		justify-content: center;
		align-items: center;
		min-height: 100vh;
		text-align: center;
		gap: 1rem;
	}

	.home-button {
		padding: 0.75rem 1.5rem;
		background: #646cff;
		color: white;
		border: none;
		border-radius: 6px;
		cursor: pointer;
		text-decoration: none;
		font-weight: 500;
	}

	.party-container {
		max-width: 1200px;
		margin: 0 auto;
		padding: 1rem;
	}

	.party-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: 2rem;
		padding-bottom: 1rem;
		border-bottom: 1px solid #333;
	}

	.party-info h1 {
		margin: 0 0 0.5rem 0;
		font-size: 2.5rem;
		background: linear-gradient(45deg, #646cff, #ff6b6b);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		background-clip: text;
	}

	.party-id {
		margin: 0 0 1rem 0;
		color: #888;
		font-family: 'Courier New', monospace;
		font-size: 0.9rem;
	}

	.state-badge {
		padding: 0.25rem 0.75rem;
		border-radius: 1rem;
		font-size: 0.8rem;
		font-weight: 600;
		text-transform: uppercase;
	}

	.state-lobby {
		background: #2d5a2d;
		color: #90ee90;
	}

	.header-nav {
		display: flex;
		gap: 1rem;
	}

	.nav-link {
		color: #646cff;
		text-decoration: none;
		font-weight: 500;
		padding: 0.5rem 1rem;
		border-radius: 6px;
		transition: background-color 0.3s;
	}

	.nav-link:hover {
		background: rgba(100, 108, 255, 0.1);
	}

	.welcome-message {
		background: #1a1a1a;
		border: 1px solid #333;
		border-radius: 8px;
		padding: 1rem;
		margin-bottom: 2rem;
		text-align: center;
	}

	.host-badge {
		background: #5a5a2d;
		color: #ffff90;
		padding: 0.25rem 0.5rem;
		border-radius: 4px;
		font-size: 0.8rem;
		font-weight: 600;
	}

	.party-content {
		display: grid;
		grid-template-columns: 300px 1fr;
		gap: 2rem;
	}

	.participants-section {
		position: sticky;
		top: 1rem;
		height: fit-content;
	}

	.main-content {
		min-height: 400px;
	}

	.phase-info {
		background: #1a1a1a;
		border: 1px solid #333;
		border-radius: 8px;
		padding: 2rem;
	}

	.phase-info h2 {
		margin-top: 0;
		color: #646cff;
	}

	.host-controls {
		margin-top: 2rem;
		padding-top: 1rem;
		border-top: 1px solid #333;
	}

	.host-controls ul {
		margin: 1rem 0;
		padding-left: 1.5rem;
	}

	.action-button {
		padding: 0.75rem 1.5rem;
		background: linear-gradient(45deg, #646cff, #ff6b6b);
		border: none;
		border-radius: 6px;
		color: white;
		font-weight: 600;
		cursor: pointer;
		margin-top: 1rem;
	}

	.action-button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
		background: #333;
	}

	.participant-info {
		margin-top: 2rem;
		padding: 1rem;
		background: rgba(100, 108, 255, 0.1);
		border-radius: 6px;
		border: 1px solid rgba(100, 108, 255, 0.3);
	}

	/* Modal styles */
	.modal-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.8);
		display: flex;
		justify-content: center;
		align-items: center;
		z-index: 1000;
	}

	.modal {
		background: #1a1a1a;
		border: 1px solid #333;
		border-radius: 12px;
		padding: 2rem;
		max-width: 400px;
		width: 90%;
		text-align: center;
	}

	.modal h2 {
		margin-top: 0;
		color: #646cff;
	}

	.form-group {
		margin: 1.5rem 0;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.username-input {
		padding: 1rem;
		border: 2px solid #333;
		border-radius: 8px;
		background: #0a0a0a;
		color: white;
		font-size: 1rem;
		text-align: center;
	}

	.username-input:focus {
		outline: none;
		border-color: #646cff;
	}

	.join-button {
		padding: 1rem 2rem;
		background: linear-gradient(45deg, #646cff, #ff6b6b);
		border: none;
		border-radius: 8px;
		color: white;
		font-size: 1rem;
		font-weight: 600;
		cursor: pointer;
	}

	.join-button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.error {
		color: #ff6b6b;
		margin-top: 1rem;
		padding: 0.75rem;
		background: rgba(255, 107, 107, 0.1);
		border: 1px solid rgba(255, 107, 107, 0.3);
		border-radius: 6px;
	}

	@media (max-width: 768px) {
		.party-header {
			flex-direction: column;
			gap: 1rem;
		}

		.party-content {
			grid-template-columns: 1fr;
		}

		.participants-section {
			position: static;
		}
	}
</style> 