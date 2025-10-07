<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { Player } from '$lib/player/player.svelte';
	import { PlayIcon, PauseIcon } from '@lucide/svelte';
	import Search from '$lib/component/search.svelte';
	import { formatTrackName, type TrackList } from '$lib/query/track';

	const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';

	const player = new Player({
		wsUrl: `${protocol}://${window.location.host}/api/ws`,
		sampleRate: 48000,
		channels: 1,
		bytesPerSample: 2,
		frameSize: 480,
		bufferLatencyMs: 50
	});

	let volumeSliderValue = $state(Number(localStorage.getItem('volume') ?? '0.5'));

	// need references
	let canvasEl: HTMLCanvasElement;
	let ctx: CanvasRenderingContext2D;

	$effect(() => {
		player.setVolume(volumeSliderValue);
		localStorage.setItem('volume', volumeSliderValue.toString());
	});

	onMount(() => {
		ctx = canvasEl.getContext('2d')!;

		function draw() {
			const data = player.visualData;
			const width = (canvasEl.width = window.innerWidth);
			const height = (canvasEl.height = window.innerHeight);
			const centerX = width / 2;
			const centerY = height / 2;
			const radius = Math.min(width, height) / 5;

			ctx.clearRect(0, 0, width, height);

			// Background: deep dark blue
			ctx.fillStyle = 'rgba(5, 10, 25, 0.95)';
			ctx.fillRect(0, 0, width, height);

			const bars = data.length;
			const angleStep = (Math.PI * 2) / bars;

			for (let i = 0; i < bars; i++) {
				const value = data[i];
				const barLength = value * radius * 1.2;
				const angle = i * angleStep - Math.PI / 2;

				const hue = 180 + value * 60; // teal-cyan
				const saturation = 70 + value * 15;
				const lightness = 40 + value * 20;

				const x1 = centerX + Math.cos(angle) * radius;
				const y1 = centerY + Math.sin(angle) * radius;
				const x2 = centerX + Math.cos(angle) * (radius + barLength);
				const y2 = centerY + Math.sin(angle) * (radius + barLength);

				// Create gradient that fades toward the center
				const grad = ctx.createLinearGradient(x1, y1, x2, y2);
				grad.addColorStop(0, `hsla(${hue}, ${saturation}%, ${lightness}%, 0)`); // Fades in near center
				grad.addColorStop(0.4, `hsla(${hue}, ${saturation}%, ${lightness}%, 0.5)`); // mid fade
				grad.addColorStop(1, `hsla(${hue}, ${saturation}%, ${lightness}%, 1)`); // bright outer tip

				ctx.strokeStyle = grad;
				ctx.lineWidth = 2 + value;

				ctx.beginPath();
				ctx.moveTo(x1, y1);
				ctx.lineTo(x2, y2);
				ctx.stroke();
			}

			requestAnimationFrame(draw);
		}

		draw();
	});

	function handleConnectionToggle() {
		if (player.status === 'connected') player.disconnect();
		else if (player.status === 'not_connected') player.connect();
	}

	let trackList: TrackList = $state({ folders: [], tracks: [] });

	let activeTrackIntervalId: number | undefined = undefined;
	let activeTrack: any = $state();

	onMount(async () => {
		const res = await fetch('/api/tracks');
		trackList = await res.json();

		const trackRes = await fetch('/api/track');
		activeTrack = await trackRes.json();

		activeTrackIntervalId = setInterval(async () => {
			const res = await fetch('/api/track');
			activeTrack = await res.json();
		}, 5000);
	});

	onDestroy(() => {
		if (activeTrackIntervalId) {
			clearInterval(activeTrackIntervalId);
		}
	});
</script>

<main class="relative h-screen w-screen overflow-hidden bg-neutral-950">
	<canvas bind:this={canvasEl} class="absolute inset-0"></canvas>

	<Search {trackList} />

	<!-- Controls docked at bottom -->
	<div
		class="absolute bottom-0 left-0 right-0 flex items-center justify-between rounded border border-white/10 bg-gray-900/80 px-6 py-4"
	>
		<div class="flex items-center">
			<button
				onclick={handleConnectionToggle}
				class="cursor-pointer rounded-full bg-gray-500 p-3 shadow-lg hover:scale-105"
			>
				{#if player.status === 'connected'}
					<PauseIcon class="h-6 w-6 text-white" />
				{:else}
					<PlayIcon class="h-6 w-6 text-white" />
				{/if}
			</button>

			{#if activeTrack}
				<div class="flex flex-col gap-1 pl-6 text-white">
					<div>{formatTrackName(activeTrack)}</div>
				</div>
			{/if}
		</div>

		<!-- <div>
			<span class="text-sm font-semibold text-gray-400">{player.status}</span>
		</div> -->

		<div class="flex items-center gap-4">
			<input
				bind:value={volumeSliderValue}
				aria-label="volume"
				type="range"
				min="0"
				max="1"
				step="0.01"
				class="cursor-pointer accent-gray-500"
			/>
		</div>
	</div>
</main>
