import { getLogarithmicVolume } from '$lib/query/track';

type PlayerOptions = {
	wsUrl: string;
	sampleRate: number;
	channels: number;
	bytesPerSample: number;
	frameSize: number;
	bufferLatencyMs: number;
};

type PlayerStatus = 'not_connected' | 'connecting' | 'connected' | 'error';

export class Player {
	#options: PlayerOptions;
	#ws: WebSocket | undefined;
	#audioCtx: AudioContext | undefined;
	#gainNode: GainNode | undefined;
	#analyser: AnalyserNode | undefined;
	#frequencyDataArray: Uint8Array<ArrayBuffer> | undefined;

	#startTime = 0;
	#bufferedSeconds = 0;
	#frameQueue: Float32Array[] = [];
	#playTimer: number | undefined;

	#volume = $state(1);
	#status = $state<PlayerStatus>('not_connected');
	#visualData = $state<number[]>(new Array(64).fill(0)); // smooth visual array

	constructor(options: PlayerOptions) {
		this.#options = options;
	}

	get volume() {
		return this.#volume;
	}

	get status() {
		return this.#status;
	}

	get visualData() {
		return this.#visualData;
	}

	connect() {
		if (this.#status !== 'not_connected') return;

		this.#audioCtx = new AudioContext({ sampleRate: this.#options.sampleRate });
		this.#gainNode = this.#audioCtx.createGain();
		this.#gainNode.gain.value = getLogarithmicVolume(this.#volume);

		// Setup analyser node for frequency visualization
		this.#analyser = this.#audioCtx.createAnalyser();
		this.#analyser.fftSize = 2048;
		this.#frequencyDataArray = new Uint8Array(this.#analyser.frequencyBinCount);

		// Connect graph: gain -> analyser -> destination
		this.#gainNode.connect(this.#analyser);
		this.#analyser.connect(this.#audioCtx.destination);

		this.#startTime = this.#audioCtx.currentTime;
		this.#bufferedSeconds = 0;
		this.#frameQueue = [];

		this.#ws = new WebSocket(this.#options.wsUrl);
		this.#ws.binaryType = 'arraybuffer';
		this.#status = 'connecting';

		this.#ws.onopen = () => {
			this.#status = 'connected';
			this.#startVisualizerLoop();
		};

		this.#ws.onmessage = (event) => {
			const arrayBuffer = event.data;
			const int16View = new Int16Array(arrayBuffer);
			const float32Data = new Float32Array(int16View.length);

			for (let i = 0; i < int16View.length; i++) {
				float32Data[i] = int16View[i] / 32768;
			}

			this.#frameQueue.push(float32Data);

			const frameDuration = this.#options.frameSize / this.#options.sampleRate;
			this.#bufferedSeconds += frameDuration;

			if (
				this.#bufferedSeconds >= this.#options.bufferLatencyMs / 1000 &&
				this.#audioCtx?.state === 'suspended'
			) {
				this.#audioCtx.resume();
			}
		};

		this.#ws.onclose = () => this.disconnect();
		this.#ws.onerror = (err) => {
			console.error('WebSocket error:', err);
			this.#status = 'error';
		};

		// Playback loop
		this.#playTimer = window.setInterval(() => this.#playNextFrame(), 5);
	}

	disconnect() {
		if (this.#status !== 'connected' && this.#status !== 'error') return;

		if (this.#ws) this.#ws.close();
		if (this.#playTimer) clearInterval(this.#playTimer);
		if (this.#audioCtx && this.#audioCtx.state !== 'closed') this.#audioCtx.close();

		this.#frameQueue = [];
		this.#bufferedSeconds = 0;

		// Smooth fade-out visualizer to zero
		const decay = () => {
			let anyActive = false;
			this.#visualData = this.#visualData.map((v) => {
				const nv = v * 0.9;
				if (nv > 0.01) anyActive = true;
				return nv;
			});
			if (anyActive) requestAnimationFrame(decay);
		};
		decay();

		this.#status = 'not_connected';
	}

	setVolume(volume: number) {
		this.#volume = volume;
		if (!this.#gainNode) return;
		this.#gainNode.gain.value = getLogarithmicVolume(volume);
	}

	#playNextFrame() {
		if (!this.#audioCtx || !this.#gainNode || this.#audioCtx.state === 'closed') return;
		if (this.#frameQueue.length === 0) return;

		const data = this.#frameQueue.shift();
		if (!data) return;

		const buffer = this.#audioCtx.createBuffer(
			this.#options.channels,
			data.length / this.#options.channels,
			this.#options.sampleRate
		);
		if (this.#options.channels === 2) {
			const left = buffer.getChannelData(0);
			const right = buffer.getChannelData(1);

			for (let i = 0, j = 0; i < left.length; i++, j += 2) {
				left[i] = data[j]; // L
				right[i] = data[j + 1]; // R
			}
		} else {
			buffer.getChannelData(0).set(data);
		}

		const source = this.#audioCtx.createBufferSource();
		source.buffer = buffer;
		source.connect(this.#gainNode);

		const now = this.#audioCtx.currentTime;
		if (this.#startTime < now) this.#startTime = now;

		source.start(this.#startTime);
		this.#startTime += buffer.duration;
		this.#bufferedSeconds = Math.max(0, this.#bufferedSeconds - buffer.duration);
	}

	#startVisualizerLoop() {
		if (!this.#analyser || !this.#frequencyDataArray) return;

		const loop = () => {
			if (this.#status !== 'connected') return;

			this.#analyser!.getByteFrequencyData(this.#frequencyDataArray!);

			const bins = 64;
			const step = Math.floor(this.#frequencyDataArray!.length / bins);
			const newValues: number[] = [];

			for (let i = 0; i < bins; i++) {
				let sum = 0;
				for (let j = 0; j < step; j++) {
					sum += this.#frequencyDataArray![i * step + j];
				}
				const avg = sum / step / 255;
				newValues.push(avg);
			}

			// Smooth interpolation for fluid motion
			this.#visualData = this.#visualData.map((prev, i) => {
				const target = newValues[i] ?? 0;
				return prev + (target - prev) * 0.25; // lerp
			});

			requestAnimationFrame(loop);
		};

		requestAnimationFrame(loop);
	}
}
