<script lang="ts">
	import { formatTrackName, type TrackList } from '$lib/query/track';
	import { FolderIcon, ListMusicIcon, Music2Icon, SearchIcon } from '@lucide/svelte';

	type Props = {
		trackList: TrackList;
	};

	const { trackList }: Props = $props();

	let open = $state(false);

	function handleShortcuts(e: KeyboardEvent) {
		if ((e.key === 'k' || e.key === 'f') && (e.metaKey || e.ctrlKey)) {
			e.preventDefault();
			open = true;
		}
	}

	async function playFolder(folder: TrackList['folders'][number]) {
		await fetch('/api/play', {
			method: 'POST',
			body: JSON.stringify({ folder: folder.name, track: '' })
		});
	}

	async function playTrack(track: TrackList['tracks'][number]) {
		await fetch('/api/play', {
			method: 'POST',
			body: JSON.stringify({ folder: track.folder, track: track.name })
		});
	}
</script>

<svelte:window onkeydown={handleShortcuts} />

<button
	command="show-modal"
	commandfor="dialog"
	class="rounded-md bg-gray-800/80 px-2.5 py-1.5 text-sm font-semibold text-white hover:bg-gray-700/90"
>
	Open command palette
</button>

<el-dialog {open} onopen={() => (open = true)} onclose={() => (open = false)}>
	<dialog id="dialog" class="backdrop:bg-transparent">
		<el-dialog-backdrop
			class="data-closed:opacity-0 data-enter:duration-300 data-enter:ease-out data-leave:duration-200 data-leave:ease-in fixed inset-0 bg-gray-900/50 transition-opacity"
		></el-dialog-backdrop>

		<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
		<div
			tabindex="0"
			class="fixed inset-0 w-screen overflow-y-auto p-4 focus:outline-none sm:p-6 md:p-20"
		>
			<el-dialog-panel
				class="data-closed:scale-95 data-closed:opacity-0 data-enter:duration-300 data-enter:ease-out data-leave:duration-200 data-leave:ease-in mx-auto block max-w-2xl transform overflow-hidden rounded-xl bg-gray-900/80 shadow-2xl outline-1 -outline-offset-1 outline-white/10 backdrop-blur-sm backdrop-filter transition-all"
			>
				<el-command-palette>
					<div class="grid grid-cols-1 border-b border-white/10">
						<!-- svelte-ignore a11y_autofocus -->
						<input
							type="text"
							autofocus
							placeholder="Search..."
							class="outline-hidden col-start-1 row-start-1 h-12 w-full bg-transparent pl-11 pr-4 text-base text-white placeholder:text-gray-400 sm:text-sm"
						/>
						<SearchIcon
							data-slot="icon"
							aria-hidden="true"
							class="pointer-events-none col-start-1 row-start-1 ml-4 size-5 self-center text-gray-500"
						/>
					</div>

					<el-command-list class="block max-h-80 scroll-py-2 overflow-y-auto">
						<el-defaults class="block divide-y divide-white/10">
							<div class="p-2">
								<h2 class="sr-only">Quick actions</h2>
								<div class="text-sm text-gray-300">
									{#each trackList.folders as folder (folder.name)}
										<button
											onclick={() => playFolder(folder)}
											class="focus:outline-hidden group flex w-full cursor-default select-none items-center rounded-md px-3 py-2 text-left aria-selected:bg-white/5 aria-selected:text-white"
										>
											<ListMusicIcon
												data-slot="icon"
												aria-hidden="true"
												class="size-6 flex-none text-gray-500 group-aria-selected:text-white"
											/>
											<span class="ml-3 flex-auto truncate">{folder.name}</span>
											<span
												aria-hidden="true"
												class="ml-3 hidden flex-none text-gray-400 group-aria-selected:inline"
											>
												Play ({folder.track_count} tracks)
											</span>
										</button>
									{/each}
								</div>
							</div>
						</el-defaults>

						<el-command-group hidden class="block p-2 text-sm text-gray-300">
							{#each trackList.folders as folder (folder.name)}
								<button
									onclick={() => playFolder(folder)}
									type="button"
									hidden
									class="focus:outline-hidden group flex w-full cursor-default select-none items-center rounded-md px-3 py-2 text-left aria-selected:bg-white/5 aria-selected:text-white"
								>
									<ListMusicIcon
										data-slot="icon"
										aria-hidden="true"
										class="size-6 flex-none text-gray-500 group-aria-selected:text-white"
									/>
									<span class="ml-3 flex-auto truncate">{folder.name}</span>
									<span
										aria-hidden="true"
										class="ml-3 hidden flex-none text-gray-400 group-aria-selected:inline"
									>
										Play ({folder.track_count} tracks)
									</span>
								</button>
							{/each}
							{#each trackList.tracks as track (track.folder + track.name)}
								<button
									onclick={() => playTrack(track)}
									type="button"
									hidden
									class="focus:outline-hidden group flex w-full cursor-default select-none items-center rounded-md px-3 py-2 text-left aria-selected:bg-white/5 aria-selected:text-white"
								>
									<Music2Icon
										data-slot="icon"
										aria-hidden="true"
										class="size-6 flex-none text-gray-500 group-aria-selected:text-white"
									/>
									<span class="ml-3 flex-auto truncate">{formatTrackName(track)}</span>
									<span
										aria-hidden="true"
										class="ml-3 hidden flex-none text-gray-400 group-aria-selected:inline"
									>
										Play ({Math.ceil(track.duration / 60)} min)
									</span>
								</button>
							{/each}
						</el-command-group>
					</el-command-list>

					<el-no-results hidden class="block px-6 py-14 text-center sm:px-14">
						<FolderIcon data-slot="icon" aria-hidden="true" class="mx-auto size-6 text-gray-500" />
						<p class="mt-4 text-sm text-gray-200">
							We couldn't find anything with that term. Please try again.
						</p>
					</el-no-results>
				</el-command-palette>
			</el-dialog-panel>
		</div>
	</dialog>
</el-dialog>
