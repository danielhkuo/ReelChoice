<script lang="ts">
	export let participants: Array<{
		id: string;
		username: string;
		isHost: boolean;
		joinedAt: string;
	}> = [];

	function formatJoinTime(joinedAt: string): string {
		const date = new Date(joinedAt);
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}
</script>

<div class="participants-container">
	<div class="participants-header">
		<h3>👥 Participants ({participants.length})</h3>
	</div>
	
	<div class="participants-list">
		{#each participants as participant (participant.id)}
			<div class="participant" class:is-host={participant.isHost}>
				<div class="participant-info">
					<div class="participant-name">
						{participant.username}
						{#if participant.isHost}
							<span class="host-badge">👑 Host</span>
						{/if}
					</div>
					<div class="participant-meta">
						Joined at {formatJoinTime(participant.joinedAt)}
					</div>
				</div>
				<div class="participant-status">
					<div class="status-indicator online"></div>
				</div>
			</div>
		{:else}
			<div class="empty-state">
				<p>No participants yet</p>
			</div>
		{/each}
	</div>
</div>

<style>
	.participants-container {
		background: #1a1a1a;
		border: 1px solid #333;
		border-radius: 8px;
		overflow: hidden;
	}

	.participants-header {
		padding: 1rem;
		background: #222;
		border-bottom: 1px solid #333;
	}

	.participants-header h3 {
		margin: 0;
		color: #646cff;
		font-size: 1.1rem;
	}

	.participants-list {
		padding: 0.5rem 0;
		max-height: 400px;
		overflow-y: auto;
	}

	.participant {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem 1rem;
		border-bottom: 1px solid #2a2a2a;
		transition: background-color 0.2s;
	}

	.participant:last-child {
		border-bottom: none;
	}

	.participant:hover {
		background: rgba(100, 108, 255, 0.05);
	}

	.participant.is-host {
		background: rgba(255, 255, 144, 0.05);
		border-left: 3px solid #ffff90;
	}

	.participant-info {
		flex: 1;
	}

	.participant-name {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-weight: 500;
		color: #fff;
		margin-bottom: 0.25rem;
	}

	.host-badge {
		background: #5a5a2d;
		color: #ffff90;
		padding: 0.15rem 0.4rem;
		border-radius: 3px;
		font-size: 0.7rem;
		font-weight: 600;
	}

	.participant-meta {
		font-size: 0.8rem;
		color: #888;
	}

	.participant-status {
		display: flex;
		align-items: center;
	}

	.status-indicator {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: #90ee90;
		box-shadow: 0 0 0 2px rgba(144, 238, 144, 0.3);
	}

	.status-indicator.online {
		background: #90ee90;
		animation: pulse 2s infinite;
	}

	@keyframes pulse {
		0% {
			box-shadow: 0 0 0 0 rgba(144, 238, 144, 0.7);
		}
		70% {
			box-shadow: 0 0 0 6px rgba(144, 238, 144, 0);
		}
		100% {
			box-shadow: 0 0 0 0 rgba(144, 238, 144, 0);
		}
	}

	.empty-state {
		padding: 2rem 1rem;
		text-align: center;
		color: #888;
	}

	.empty-state p {
		margin: 0;
		font-style: italic;
	}

	/* Custom scrollbar */
	.participants-list::-webkit-scrollbar {
		width: 6px;
	}

	.participants-list::-webkit-scrollbar-track {
		background: #222;
	}

	.participants-list::-webkit-scrollbar-thumb {
		background: #444;
		border-radius: 3px;
	}

	.participants-list::-webkit-scrollbar-thumb:hover {
		background: #555;
	}
</style> 