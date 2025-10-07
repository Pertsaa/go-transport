package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	SampleRate     = 48000                   // 48 kHz
	Channels       = 2                       // Stereo
	BitDepth       = 16                      // 16-bit PCM
	FrameSize      = SampleRate / 100        // 480 samples per 10ms per channel
	BytesPerSample = BitDepth / 8 * Channels // 4 bytes per sample frame
	AudioDir       = "audio/output"          // Base directory for audio files
)

var audioFrame = make([]byte, FrameSize*Channels*(BitDepth/8))

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[*websocket.Conn]bool)
var register = make(chan *websocket.Conn)
var unregister = make(chan *websocket.Conn)

var (
	activeTrackMux   sync.RWMutex
	activeTrackPath  = ""
	streamStopSignal = make(chan struct{})
)

type Folder struct {
	Name       string `json:"name"`
	TrackCount int    `json:"track_count"`
}

type Track struct {
	Folder   string `json:"folder"`
	Name     string `json:"name"`
	Duration int    `json:"duration"` // in seconds
}

type TrackList struct {
	Folders []Folder `json:"folders"`
	Tracks  []Track  `json:"tracks"`
}

type PlayBody struct {
	Folder string `json:"folder"`
	Track  string `json:"track"`
}

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer ws.Close()

	register <- ws

	for {
		if _, _, err := ws.ReadMessage(); err != nil {
			unregister <- ws
			break
		}
	}
}

func manageClients() {
	for {
		select {
		case client := <-register:
			clients[client] = true
			log.Printf("New client connected. Total clients: %d", len(clients))
		case client := <-unregister:
			delete(clients, client)
			log.Printf("Client disconnected. Total clients: %d", len(clients))
		}
	}
}

func broadcastFrame(frame []byte) {
	for client := range clients {
		err := client.WriteMessage(websocket.BinaryMessage, frame)
		if err != nil {
			log.Printf("Write error to client: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}

func streamFile(filePath string) error {
	activeTrackMux.RLock()
	currentPath := activeTrackPath
	activeTrackMux.RUnlock()

	if filePath != currentPath {
		return nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	log.Printf("🎵 Streaming file: %s", filePath)

	delay := time.Duration(FrameSize) * time.Second / time.Duration(SampleRate)
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			n, err := io.ReadFull(f, audioFrame)
			if err == io.EOF {
				return nil
			} else if err == io.ErrUnexpectedEOF {
				if n > 0 {
					broadcastFrame(audioFrame[:n])
				}
				return nil
			} else if err != nil {
				return err
			}

			broadcastFrame(audioFrame)

		case <-streamStopSignal:
			log.Println("🛑 Stream stop signal received. Switching track.")
			return nil
		}
	}
}

func streamAudioFiles() {
	activeTrackMux.Lock()
	if activeTrackPath == "" {
		if files, err := filepath.Glob(filepath.Join(AudioDir, "*", "*.pcm")); err == nil && len(files) > 0 {
			sort.Strings(files)
			activeTrackPath = files[0]
			log.Printf("Default track set: %s", activeTrackPath)
		}
	}
	activeTrackMux.Unlock()

	for {
		activeTrackMux.RLock()
		currentTrack := activeTrackPath
		activeTrackMux.RUnlock()

		if currentTrack == "" {
			log.Println("No active track set. Waiting for /play...")
			time.Sleep(3 * time.Second)
			continue
		}

		if err := streamFile(currentTrack); err != nil {
			log.Printf("Error streaming %s: %v", currentTrack, err)
			time.Sleep(2 * time.Second)
		}
		activeTrackMux.RLock()
		nextTrack := activeTrackPath
		activeTrackMux.RUnlock()

		if currentTrack == nextTrack {
			files, err := filepath.Glob(filepath.Join(AudioDir, "*", "*.pcm"))
			if err != nil {
				log.Printf("Error collecting track list for loop: %v", err)
				time.Sleep(3 * time.Second)
				continue
			}
			sort.Strings(files)

			currentIndex := -1
			for i, file := range files {
				if file == currentTrack {
					currentIndex = i
					break
				}
			}

			if currentIndex != -1 && currentIndex < len(files)-1 {
				activeTrackMux.Lock()
				activeTrackPath = files[currentIndex+1]
				activeTrackMux.Unlock()
				log.Printf("Automatically advancing to next track: %s", activeTrackPath)
			} else {
				activeTrackMux.Lock()
				activeTrackPath = files[0]
				activeTrackMux.Unlock()
				log.Printf("Playlist finished, starting over with: %s", activeTrackPath)
				if len(files) == 0 {
					activeTrackMux.Lock()
					activeTrackPath = ""
					activeTrackMux.Unlock()
				}
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func getTrackInfoFromPath(filePath string) (Track, error) {
	relPath, err := filepath.Rel(AudioDir, filePath)
	if err != nil {
		return Track{}, err
	}
	parts := strings.Split(relPath, string(os.PathSeparator))
	if len(parts) < 2 {
		return Track{}, os.ErrInvalid
	}

	folderName := parts[0]
	trackName := parts[len(parts)-1]

	info, err := os.Stat(filePath)
	if err != nil {
		return Track{}, err
	}

	size := info.Size()
	duration := int(size / int64(SampleRate*BytesPerSample))

	return Track{
		Folder:   folderName,
		Name:     trackName,
		Duration: duration,
	}, nil
}

func loadTracks() (TrackList, error) {
	trackList := TrackList{
		Folders: []Folder{},
		Tracks:  []Track{},
	}

	entries, err := os.ReadDir(AudioDir)
	if err != nil {
		return trackList, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		folderName := entry.Name()
		folderPath := filepath.Join(AudioDir, folderName)

		files, err := os.ReadDir(folderPath)
		if err != nil {
			continue
		}

		trackCount := 0
		folderTracks := []Track{}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".pcm") {
				continue
			}

			filePath := filepath.Join(folderPath, file.Name())
			track, err := getTrackInfoFromPath(filePath)
			if err != nil {
				log.Printf("Error getting track info for %s: %v", filePath, err)
				continue
			}

			trackCount++
			folderTracks = append(folderTracks, track)
		}

		trackList.Tracks = append(trackList.Tracks, folderTracks...)
		trackList.Folders = append(trackList.Folders, Folder{
			Name:       folderName,
			TrackCount: trackCount,
		})
	}

	return trackList, nil
}

func handleTrackList(w http.ResponseWriter, r *http.Request) {
	trackList, err := loadTracks()
	if err != nil {
		http.Error(w, "Failed to load tracks: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trackList)
}

func handleTrackGet(w http.ResponseWriter, r *http.Request) {
	activeTrackMux.RLock()
	currentPath := activeTrackPath
	activeTrackMux.RUnlock()

	if currentPath == "" {
		http.Error(w, "No track is currently playing", http.StatusNoContent)
		return
	}

	track, err := getTrackInfoFromPath(currentPath)
	if err != nil {
		http.Error(w, "Failed to get active track info", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(track)
}

func handlePlay(w http.ResponseWriter, r *http.Request) {
	var body PlayBody

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	targetPath := filepath.Join(AudioDir, body.Folder, body.Track)
	if !strings.HasSuffix(targetPath, ".pcm") {
		targetPath += ".pcm"
	}

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		http.Error(w, "Track file not found: "+targetPath, http.StatusNotFound)
		return
	}

	activeTrackMux.Lock()
	oldPath := activeTrackPath
	activeTrackPath = targetPath
	activeTrackMux.Unlock()

	if oldPath != targetPath && oldPath != "" {
		log.Printf("Change active track from '%s' to '%s'. Signalling stream stop.", oldPath, targetPath)
		select {
		case streamStopSignal <- struct{}{}:
		default:
			log.Println("Note: streamStopSignal was full, relying on next loop iteration.")
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Track set and streaming initiated."))
}

func main() {
	go manageClients()
	go streamAudioFiles()

	http.HandleFunc("POST /play", withCORS(handlePlay))
	http.HandleFunc("GET /track", withCORS(handleTrackGet))
	http.HandleFunc("GET /tracks", withCORS(handleTrackList))
	http.HandleFunc("GET /ws", handleConnections)

	log.Println("🎧 Server started on http://localhost:8080. WS on /ws")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
