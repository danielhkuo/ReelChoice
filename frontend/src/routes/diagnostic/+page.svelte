<script lang="ts">
	import { onMount, onDestroy } from 'svelte';

	// State variables
	let apiStatus = 'pending';
	let wsStatus = 'pending';
	let apiResponse = '';
	let wsMessages: string[] = [];
	let ws: WebSocket | null = null;
	let partyId = 'test-party-' + Math.random().toString(36).substr(2, 9);

	// Test the REST API
	async function testAPI() {
		apiStatus = 'pending';
		apiResponse = 'Testing API...';
		
		try {
			// Test the simple test endpoint
			const response = await fetch('/api/test');
			const data = await response.json();
			
			if (response.ok) {
				apiStatus = 'success';
				apiResponse = `✅ API Test: ${data.message}`;
				
				// Now test party creation
				await testPartyCreation();
			} else {
				apiStatus = 'error';
				apiResponse = `❌ API Error: ${response.status} ${response.statusText}`;
			}
		} catch (error) {
			apiStatus = 'error';
			apiResponse = `❌ API Error: ${error}`;
		}
	}

	async function testPartyCreation() {
		try {
			const response = await fetch('/api/party', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					name: 'Test Party for Diagnostics'
				})
			});

			const data = await response.json();
			
			if (response.ok) {
				apiResponse += `\n✅ Party Creation: Created party "${data.name}" with ID: ${data.id}`;
				partyId = data.id; // Use the real party ID for WebSocket tests
			} else {
				apiResponse += `\n❌ Party Creation Error: ${response.status} ${response.statusText}`;
			}
		} catch (error) {
			apiResponse += `\n❌ Party Creation Error: ${error}`;
		}
	}

	// Test WebSocket connection
	function testWebSocket() {
		wsStatus = 'pending';
		wsMessages = ['Connecting to WebSocket...'];
		
		try {
			// Close existing connection if any
			if (ws) {
				ws.close();
			}

			// Create WebSocket connection
			const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
			const wsUrl = `${protocol}//${window.location.host}/ws?partyId=${partyId}`;
			
			ws = new WebSocket(wsUrl);

			ws.onopen = () => {
				wsStatus = 'success';
				wsMessages = [...wsMessages, '✅ WebSocket connected successfully!'];
				
				// Send a test message
				setTimeout(() => {
					if (ws && ws.readyState === WebSocket.OPEN) {
						const testMessage = {
							type: 'test',
							data: 'Hello from diagnostic tool!',
							timestamp: Date.now()
						};
						ws.send(JSON.stringify(testMessage));
						wsMessages = [...wsMessages, `📤 Sent: ${JSON.stringify(testMessage, null, 2)}`];
					}
				}, 1000);
			};

			ws.onmessage = (event) => {
				try {
					const message = JSON.parse(event.data);
					wsMessages = [...wsMessages, `📥 Received: ${JSON.stringify(message, null, 2)}`];
				} catch (error) {
					wsMessages = [...wsMessages, `📥 Received (raw): ${event.data}`];
				}
			};

			ws.onerror = (error) => {
				wsStatus = 'error';
				wsMessages = [...wsMessages, `❌ WebSocket error: ${error}`];
			};

			ws.onclose = (event) => {
				if (wsStatus !== 'error') {
					wsMessages = [...wsMessages, `🔌 WebSocket closed: Code ${event.code}, Reason: ${event.reason || 'No reason provided'}`];
				}
			};

		} catch (error) {
			wsStatus = 'error';
			wsMessages = [...wsMessages, `❌ WebSocket connection error: ${error}`];
		}
	}

	// Send a ping message
	function sendPing() {
		if (ws && ws.readyState === WebSocket.OPEN) {
			const pingMessage = {
				type: 'ping',
				data: 'ping',
				timestamp: Date.now()
			};
			ws.send(JSON.stringify(pingMessage));
			wsMessages = [...wsMessages, `📤 Sent ping: ${JSON.stringify(pingMessage, null, 2)}`];
		} else {
			wsMessages = [...wsMessages, '❌ WebSocket not connected'];
		}
	}

	// Clear WebSocket messages
	function clearWSMessages() {
		wsMessages = [];
	}

	// Run tests on component mount
	onMount(() => {
		testAPI();
	});

	// Clean up WebSocket on component destroy
	onDestroy(() => {
		if (ws) {
			ws.close();
		}
	});
</script>

<svelte:head>
	<title>ReelChoice Backend Diagnostic Tool</title>
</svelte:head>

<div class="container">
	<h1>🎬 ReelChoice Backend Diagnostic Tool</h1>
	<p>This tool validates that your Go backend's REST API and WebSocket hub are functioning correctly.</p>

	<div class="nav">
		<a href="/" class="nav-link">← Back to Home</a>
	</div>

	<!-- API Testing Section -->
	<div class="test-section">
		<h2>🔌 REST API Test</h2>
		<div class="status {apiStatus}">
			Status: {apiStatus.toUpperCase()}
		</div>
		<button on:click={testAPI}>Test API</button>
		<div class="card">
			<h3>API Response:</h3>
			<div class="log">{apiResponse || 'No response yet...'}</div>
		</div>
	</div>

	<!-- WebSocket Testing Section -->
	<div class="test-section">
		<h2>🔄 WebSocket Test</h2>
		<div class="status {wsStatus}">
			Status: {wsStatus.toUpperCase()}
		</div>
		<div class="controls">
			<button on:click={testWebSocket}>Connect WebSocket</button>
			<button on:click={sendPing} disabled={wsStatus !== 'success'}>Send Ping</button>
			<button on:click={clearWSMessages}>Clear Messages</button>
		</div>
		<div class="card">
			<h3>WebSocket Messages:</h3>
			<div class="log">
				{#each wsMessages as message}
					{message}
					{'\n'}
				{/each}
				{#if wsMessages.length === 0}
					No messages yet...
				{/if}
			</div>
		</div>
	</div>

	<!-- Connection Info -->
	<div class="test-section">
		<h2>ℹ️ Connection Info</h2>
		<div class="card">
			<p><strong>Backend URL:</strong> http://localhost:8081</p>
			<p><strong>API Endpoint:</strong> /api/test</p>
			<p><strong>WebSocket Endpoint:</strong> /ws?partyId={partyId}</p>
			<p><strong>Party ID:</strong> {partyId}</p>
		</div>
	</div>

	<!-- Instructions -->
	<div class="test-section">
		<h2>📋 Instructions</h2>
		<div class="card">
			<ol>
				<li>Make sure your Go backend is running on port 8081</li>
				<li>Click "Test API" to verify REST endpoints are working</li>
				<li>Click "Connect WebSocket" to test real-time functionality</li>
				<li>Use "Send Ping" to test bidirectional communication</li>
				<li>Check the logs for detailed information about each test</li>
			</ol>
			<p><strong>Expected Results:</strong></p>
			<ul>
				<li>✅ API should return a success message and create a party</li>
				<li>✅ WebSocket should connect and echo test messages</li>
				<li>✅ Ping should receive a pong response</li>
			</ul>
		</div>
	</div>
</div>

<style>
	.container {
		max-width: 1000px;
		margin: 0 auto;
		padding: 2rem;
	}

	.nav {
		margin-bottom: 2rem;
	}

	.nav-link {
		color: #646cff;
		text-decoration: none;
		font-weight: 500;
	}

	.nav-link:hover {
		text-decoration: underline;
	}

	.controls {
		margin: 1rem 0;
	}

	.controls button {
		margin-right: 0.5rem;
	}

	button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	ol, ul {
		text-align: left;
	}

	li {
		margin: 0.5rem 0;
	}
</style> 