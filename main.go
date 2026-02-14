
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
"time"
"encoding/csv"
 "regexp" // Added for ISRC validation
	"jj/backend" 
)
// --- MISSING PIECES ADDED BELOW ---
var isrcRegex = regexp.MustCompile(`^[A-Z]{2}[A-Z0-9]{3}\d{2}\d{5}$`)

func isValidISRC(isrc string) bool {
	return isrcRegex.MatchString(isrc)
}
func getISRCsFromCSV(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("csv is empty")
	}

	// 1. Find the column index for "ISRC" (it has a trailing space in your file)
	header := records[0]
	isrcColIdx := -1
	for i, col := range header {
		if strings.TrimSpace(col) == "ISRC" {
			isrcColIdx = i
			break
		}
	}

	if isrcColIdx == -1 {
		return nil, fmt.Errorf("column 'ISRC' not found")
	}

	// 2. Extract ISRCs into a slice
	var isrcs []string
	for _, row := range records[1:] { // skip header
		if len(row) > isrcColIdx {
			isrcs = append(isrcs, row[isrcColIdx])
		}
	}

	return isrcs, nil
} 
func formatMs(ms int) string {
	d := time.Duration(ms) * time.Millisecond
	return fmt.Sprintf("%02d:%02d", int(d.Minutes()), int(d.Seconds())%60)
}
 
// handleResponse determines the underlying type and extracts metadata
func handleResponse(data interface{}) {
	switch v := data.(type) {
	case *backend.TrackResponse:
		// Extract the TrackMetadata and pass it to the nice print function
		printTrackNice(v.Track) 

	case *backend.AlbumResponsePayload:
		fmt.Printf("--- Album: %s by %s (%d tracks) ---\n", v.AlbumInfo.Name, v.AlbumInfo.Artists, v.AlbumInfo.TotalTracks)
		for _, t := range v.TrackList {
			fmt.Printf("%d. %s [%dms]\n", t.TrackNumber, t.Name, t.DurationMS)
		}

	case *backend.PlaylistResponsePayload:
		fmt.Printf("--- Playlist: %s (Owner: %s) ---\n", v.PlaylistInfo.Owner.Name, v.PlaylistInfo.Owner.DisplayName)
		for i, t := range v.TrackList {
			fmt.Printf("%d. %s - %s\n", i+1, t.Name, t.Artists)
		}
	default:
		fmt.Printf("Unknown data type: %T\n", v)
	}
}

// Updated function signature to use TrackMetadata directly
func printTrackNice(t backend.TrackMetadata) {
	fmt.Println("\n✨ TRACK DETAILS")
	fmt.Println("--------------------------------------------------")
	fmt.Printf("Title:       %s\n", t.Name)
	fmt.Printf("Artist(s):   %s\n", t.Artists)
	fmt.Printf("Album:       %s\n", t.AlbumName)
	fmt.Printf("Release:     %s\n", t.ReleaseDate)
	fmt.Printf("Duration:    %s\n", formatMs(t.DurationMS))
	fmt.Printf("Track No:    Disc %d, Track %d\n", t.DiscNumber, t.TrackNumber)
	fmt.Printf("ISRC:        %s\n", t.ISRC)
	fmt.Printf("Spotify ID:  %s\n", t.SpotifyID)
	fmt.Printf("External:    %s\n", t.ExternalURL)
	
	if t.Copyright != "" {
		fmt.Printf("Copyright:   %s\n", t.Copyright)
	}
	if t.Publisher != "" {
		fmt.Printf("Publisher:   %s\n", t.Publisher)
	}
	fmt.Println("--------------------------------------------------")
}
func main() {
	query := flag.String("q", "", "Spotify Track ID or URL")
	source := flag.String("source", "tidal", "tidal, amazon, or qobuz")
	quality := flag.String("quality", "LOSSLESS", "Audio quality (LOSSLESS, HI_RES)")
	outputDir := flag.String("out", "./downloads", "Output directory")
	flag.Parse()

	if *query == "" {
		fmt.Println("Usage: spoti-dl -q <spotify_id> -source [tidal|amazon|qobuz]")
		os.Exit(1)
	}

	ctx := context.Background()
	 

	// 1. Fetch Metadata from Spotify
	
	fmt.Printf("[1/3] ?? Fetching Spotify Metadata...\n")
	// Uses the client's internal logic via GetFilteredSpotifyData
		url := fmt.Sprintf("https://open.spotify.com/playlist/%s", *query)
data, err := backend.GetFilteredSpotifyData(ctx, url, false, 10)
if err != nil {
    log.Fatalf("Failed to fetch playlist data: %v", err)
}
   


	// ... inside main() after fetching data ...
// 1. Prepare a slice to hold all tracks to be downloaded
var tracksToDownload []backend.TrackMetadata

switch v := data.(type) {
case backend.TrackResponse:
    // If it's a single track, add it to our list
    tracksToDownload = append(tracksToDownload, v.Track)

case backend.PlaylistResponsePayload:
    // If it's a playlist, map each AlbumTrackMetadata to TrackMetadata
  
    for _, t := range v.TrackList {
        tracksToDownload = append(tracksToDownload, backend.TrackMetadata{
            Name:         t.Name,
            Artists:      t.Artists,
            AlbumName:    t.AlbumName,
            AlbumArtist:  t.AlbumArtist,
            ReleaseDate:  t.ReleaseDate,
            TotalTracks:  t.TotalTracks,
            DiscNumber:   t.DiscNumber,
            TotalDiscs:   t.TotalDiscs,
            ISRC:         t.ISRC,
           
            TrackNumber:  t.TrackNumber,
            SpotifyID:    t.SpotifyID,
            ExternalURL:  t.ExternalURL,
            Images:       t.Images,
        })
	 
    }
 
default:
    log.Fatalf("Received unhandled data type: %T", v)
}

var failedIDs []string

		 
// 2. Loop through the tracks (This fixes the "undefined: trackMeta" errors)

for i, trackMeta := range tracksToDownload {
// for j, colors := range colors {
// j=j+1
    fmt.Printf("\n[%d/%d] Processing: %s \n", i+1, len(tracksToDownload), trackMeta.Artists)

    // 3. Download Cover
    coverReq := &backend.CoverDownloadRequest{
        CoverURL:    trackMeta.Images,
        TrackName:   trackMeta.Name,
        ArtistName:  trackMeta.Artists,
        OutputDir:   *outputDir,
    }
    coverResp, _ := backend.DownloadCover(coverReq)
    localCover := ""
    if coverResp != nil && coverResp.Success {
        localCover = coverResp.File
    }

    // 4. Provider Switching Logic
    var finalPath string
    var err error

    switch strings.ToLower(*source) {
	case "qobuz":
        // Map quality: 6 = 16-bit, 7 = 24-bit
        qCode := "6"
        if strings.ToUpper(*quality) == "HI_RES" {
            qCode = "7"
        }
deezerISRC := trackMeta.ISRC
			// If ISRC is not valid (looks like a Spotify ID), try to fetch from Deezer
			if len(deezerISRC) != 12 || !isValidISRC(deezerISRC) {
				fmt.Printf("ISRC is invalid (%s), fetching from Deezer...\n", deezerISRC)
				songlinkClient := backend.NewSongLinkClient()
				deezerURL, err := songlinkClient.GetDeezerURLFromSpotify(trackMeta.SpotifyID)
				if err == nil {
					deezerISRC, err = backend.GetDeezerISRC(deezerURL)
				}
			}

			if deezerISRC == "" || !isValidISRC(deezerISRC) {
				fmt.Println("❌ Could not obtain a valid ISRC for Qobuz. Skipping.")
				continue
			}

			fmt.Printf("Using ISRC: %s\n", deezerISRC)

        fmt.Printf("🔍 Searching Qobuz by ISRC: %s\n", trackMeta.ISRC)
        qDownloader := backend.NewQobuzDownloader()
        finalPath, err = qDownloader.DownloadByISRC(
            deezerISRC,               // 1. ISRC
            *outputDir,                   // 2. Output Dir
            qCode,                        // 3. Quality
            "{track}. {artist} - {title}", // 4. Format
            true,                         // 5. includeTrackNumber
            i+1,                          // 6. position
            trackMeta.Name,               // 7. trackTitle
            trackMeta.Artists,            // 8. artists
            trackMeta.AlbumName,          // 9. albumTitle
            trackMeta.AlbumArtist,        // 10. albumArtist
            trackMeta.ReleaseDate,        // 11. releaseDate
            true,                         // 12. useAlbumTrackNumber
            localCover,                   // 13. coverPath
            true,                         // 14. embedMaxQualityCover
            trackMeta.TrackNumber,        // 15. spotifyTrackNumber
            trackMeta.DiscNumber,         // 16. spotifyDiscNumber
            trackMeta.TotalTracks,        // 17. spotifyTotalTracks
            trackMeta.TotalDiscs,         // 18. spotifyTotalDiscs
            trackMeta.Copyright,          // 19. spotifyCopyright
            trackMeta.Publisher,          // 20. spotifyPublisher
            trackMeta.ExternalURL,        // 21. spotifyURL
        )
    case "amazon":
        aDownloader := backend.NewAmazonDownloader()
        finalPath, err = aDownloader.DownloadBySpotifyID(
            trackMeta.SpotifyID, *outputDir, *quality, "{track}. {artist} - {title}",
            true, 1, trackMeta.Name, trackMeta.Artists, trackMeta.AlbumName,
            trackMeta.AlbumArtist, trackMeta.ReleaseDate, localCover,
            trackMeta.TrackNumber, trackMeta.DiscNumber, trackMeta.TotalTracks,
            true, trackMeta.TotalDiscs, trackMeta.Copyright, trackMeta.Publisher, trackMeta.ExternalURL,
        )

    case "tidal": // Tidal
        slClient := backend.NewSongLinkClient()
        links, _ := slClient.GetAllURLsFromSpotify(trackMeta.SpotifyID)
        tDownloader := backend.NewTidalDownloader("")
		// --- ADD THIS DEBUG PRINT BLOCK ---
		fmt.Printf("\n[DEBUG] Passing to DownloadByURLWithFallback:\n")
		if err != nil || links == nil || links.TidalURL == "" {
        fmt.Println("⚠️  Warning: Tidal URL is EMPTY/NULL. Download will likely fail.")
		failedIDs = append(failedIDs, trackMeta.SpotifyID) // Add to failed list
        continue
    } else {
        fmt.Printf("🔗 Tidal URL:  %s\n", links.TidalURL)
    }
		fmt.Printf("  Quality:     %s\n", *quality)
		fmt.Printf("  Track:       %s (No. %d)\n", trackMeta.Name, trackMeta.TrackNumber)
		fmt.Printf("  Artist:      %s\n", trackMeta.Artists)
		fmt.Printf("  Album:       %s\n", trackMeta.AlbumName)
		fmt.Printf("  ISRC:        %s\n", trackMeta.ISRC)
		fmt.Printf("  Cover Path:  %s\n", localCover)
		fmt.Println("--------------------------------------------------")
        finalPath, err = tDownloader.DownloadByURLWithFallback(
            links.TidalURL, *outputDir, *quality, "{track}. {artist} - {title}",
            true, trackMeta.TrackNumber, trackMeta.Name, trackMeta.Artists, 
            trackMeta.AlbumName, trackMeta.AlbumArtist, trackMeta.ReleaseDate, 
            true, localCover, true, trackMeta.TrackNumber, trackMeta.TotalTracks,
            trackMeta.DiscNumber, trackMeta.TotalDiscs, trackMeta.ISRC,
            trackMeta.Copyright, trackMeta.ExternalURL,
        )
    }

    if err != nil {
        fmt.Printf("❌ Download failed for %s: %v\n", trackMeta.Name, err)
        continue // Skip to next track in playlist instead of crashing
    }
// 4. Final Summary Report
if len(failedIDs) > 0 {
    fmt.Println("\n" + strings.Repeat("!", 40))
    fmt.Printf("🚨 FINISHED WITH %d FAILURES\n", len(failedIDs))
    fmt.Println("Failed Spotify IDs:")
    for _, id := range failedIDs {
        fmt.Printf(" - %s\n", id)
    }
    fmt.Println(strings.Repeat("!", 40))
} else {
    fmt.Println("\n✅ All tracks downloaded successfully!")
}
    // 5. Final Technical Analysis
    report, err := backend.AnalyzeTrack(finalPath)
    if err == nil {
        fmt.Printf("Success: %s (%d-bit / %dHz)\n", finalPath, report.BitsPerSample, report.SampleRate)
    }
}
}
