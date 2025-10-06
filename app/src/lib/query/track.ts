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
