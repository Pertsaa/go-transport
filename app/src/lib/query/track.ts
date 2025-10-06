export type TrackList = {
	folders: {
		name: string;
		track_count: number;
	}[];
	tracks: {
		folder: string;
		name: string;
		duration: number;
	}[];
};

export function formatTrackName(track: TrackList['tracks'][number]) {
	const lastDot = track.name.lastIndexOf('.');
	if (lastDot <= 0) {
		return `${track.folder} - ${track.name}`;
	}
	return `${track.folder} - ${track.name.substring(0, lastDot)}`;
}

export function getLogarithmicVolume(linearVolume: number) {
	const LOG_BASE = 10;

	const clampedLinearVolume = Math.min(1, Math.max(0, linearVolume));

	if (clampedLinearVolume === 0) {
		return 0;
	}

	const logarithmicVolume = (Math.pow(LOG_BASE, clampedLinearVolume) - 1) / (LOG_BASE - 1);

	return Math.min(1.0, Math.max(0.0, logarithmicVolume));
}
