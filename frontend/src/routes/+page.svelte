<script lang="ts">
	import { goto } from '$app/navigation';

	let partyName = '';
	let isCreating = false;
	let error = '';

	async function createParty() {
		if (!partyName.trim()) {
			error = 'Please enter a party name';
			return;
		}

		isCreating = true;
		error = '';

		try {
			const response = await fetch('/api/party', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					name: partyName.trim()
				})
			});

			const data = await response.json();

			if (response.ok) {
				// Redirect to the party page
				goto(`/party/${data.id}`);
			} else {
				error = data.error || 'Failed to create party';
			}
		} catch (err) {
			error = 'Network error. Please check if the backend is running.';
			console.error('Party creation error:', err);
		} finally {
			isCreating = false;
		}
	}

	function handleKeyPress(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			createParty();
		}
	}
</script>

<svelte:head>
	<title>ReelChoice - Movie Selection Made Easy</title>
	<meta name="description" content="End the 'what should we watch?' debate with democratic movie selection." />
</svelte:head>

<div class="hero">
	<div class="hero-content">
		<h1>🎬 ReelChoice</h1>
		<p class="tagline">End the "what should we watch?" debate forever</p>
		<p class="description">
			Create a party, nominate movies, and use ranked-choice voting to find the perfect film everyone will enjoy.
		</p>
		
		<div class="create-party-form">
			<h2>Create a New Party</h2>
			<div class="form-group">
				<input
					type="text"
					bind:value={partyName}
					placeholder="Enter party name (e.g., 'Friday Movie Night')"
					on:keypress={handleKeyPress}
					disabled={isCreating}
					class="party-input"
				/>
				<button 
					on:click={createParty} 
					disabled={isCreating || !partyName.trim()}
					class="create-button"
				>
					{isCreating ? 'Creating...' : 'Create Party'}
				</button>
			</div>
			
			{#if error}
				<div class="error">{error}</div>
			{/if}
		</div>

		<div class="features">
			<div class="feature">
				<h3>🗳️ Democratic Voting</h3>
				<p>Two-phase process: nominate movies, then rank your preferences with RCV</p>
			</div>
			<div class="feature">
				<h3>⚡ Real-Time Sync</h3>
				<p>See nominations and votes appear instantly without refreshing</p>
			</div>
			<div class="feature">
				<h3>👑 Host Control</h3>
				<p>Party creator controls the flow from nominations to final voting</p>
			</div>
		</div>

		<div class="dev-links">
			<a href="/diagnostic" class="dev-link">🔧 Backend Diagnostic Tool</a>
		</div>
	</div>
</div>

<style>
	.hero {
		min-height: 100vh;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 2rem;
		text-align: center;
	}

	.hero-content {
		max-width: 800px;
		width: 100%;
	}

	h1 {
		font-size: 4rem;
		margin-bottom: 1rem;
		background: linear-gradient(45deg, #646cff, #ff6b6b);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		background-clip: text;
	}

	.tagline {
		font-size: 1.5rem;
		margin-bottom: 1rem;
		color: #888;
	}

	.description {
		font-size: 1.1rem;
		margin-bottom: 3rem;
		color: #ccc;
		line-height: 1.6;
	}

	.create-party-form {
		background: #1a1a1a;
		border: 1px solid #333;
		border-radius: 12px;
		padding: 2rem;
		margin-bottom: 3rem;
	}

	.create-party-form h2 {
		margin-top: 0;
		margin-bottom: 1.5rem;
		color: #fff;
	}

	.form-group {
		display: flex;
		gap: 1rem;
		flex-wrap: wrap;
		align-items: center;
		justify-content: center;
	}

	.party-input {
		flex: 1;
		min-width: 300px;
		padding: 1rem;
		border: 2px solid #333;
		border-radius: 8px;
		background: #0a0a0a;
		color: white;
		font-size: 1rem;
		transition: border-color 0.3s;
	}

	.party-input:focus {
		outline: none;
		border-color: #646cff;
	}

	.party-input:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.create-button {
		padding: 1rem 2rem;
		background: linear-gradient(45deg, #646cff, #ff6b6b);
		border: none;
		border-radius: 8px;
		color: white;
		font-size: 1rem;
		font-weight: 600;
		cursor: pointer;
		transition: transform 0.2s, opacity 0.3s;
		min-width: 140px;
	}

	.create-button:hover:not(:disabled) {
		transform: translateY(-2px);
	}

	.create-button:disabled {
		opacity: 0.6;
		cursor: not-allowed;
		transform: none;
	}

	.error {
		color: #ff6b6b;
		margin-top: 1rem;
		padding: 0.75rem;
		background: rgba(255, 107, 107, 0.1);
		border: 1px solid rgba(255, 107, 107, 0.3);
		border-radius: 6px;
	}

	.features {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
		gap: 2rem;
		margin-bottom: 3rem;
	}

	.feature {
		padding: 1.5rem;
		background: #1a1a1a;
		border: 1px solid #333;
		border-radius: 8px;
	}

	.feature h3 {
		margin-top: 0;
		margin-bottom: 0.5rem;
		color: #646cff;
	}

	.feature p {
		margin: 0;
		color: #ccc;
		line-height: 1.5;
	}

	.dev-links {
		padding-top: 2rem;
		border-top: 1px solid #333;
	}

	.dev-link {
		display: inline-block;
		padding: 0.5rem 1rem;
		background: #333;
		color: #ccc;
		text-decoration: none;
		border-radius: 6px;
		font-size: 0.9rem;
		transition: background-color 0.3s;
	}

	.dev-link:hover {
		background: #444;
		color: #fff;
	}

	@media (max-width: 768px) {
		h1 {
			font-size: 3rem;
		}

		.form-group {
			flex-direction: column;
		}

		.party-input {
			min-width: 100%;
		}

		.features {
			grid-template-columns: 1fr;
		}
	}
</style>
