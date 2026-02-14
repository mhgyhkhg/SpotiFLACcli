package backend

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	spotifySize300 = "ab67616d00001e02"
	spotifySize640 = "ab67616d0000b273"
	spotifySizeMax = "ab67616d000082c1"
)

// --- Struct Definitions ---

type CoverClient struct {
	httpClient *http.Client
}

type CoverDownloadRequest struct {
	CoverURL       string `json:"cover_url"`
	TrackName      string `json:"track_name"`
	ArtistName     string `json:"artist_name"`
	AlbumName      string `json:"album_name"`
	AlbumArtist    string `json:"album_artist"`
	ReleaseDate    string `json:"release_date"`
	OutputDir      string `json:"output_dir"`
	FilenameFormat string `json:"filename_format"`
	TrackNumber    bool   `json:"track_number"`
	Position       int    `json:"position"`
	DiscNumber     int    `json:"disc_number"`
}

type CoverDownloadResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	File          string `json:"file,omitempty"`
	Error         string `json:"error,omitempty"`
	AlreadyExists bool   `json:"already_exists,omitempty"`
}

type HeaderDownloadRequest struct {
	HeaderURL  string `json:"header_url"`
	ArtistName string `json:"artist_name"`
	OutputDir  string `json:"output_dir"`
}

type HeaderDownloadResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	File          string `json:"file,omitempty"`
	Error         string `json:"error,omitempty"`
	AlreadyExists bool   `json:"already_exists,omitempty"`
}

type GalleryImageDownloadRequest struct {
	ImageURL   string `json:"image_url"`
	ArtistName string `json:"artist_name"`
	ImageIndex int    `json:"image_index"`
	OutputDir  string `json:"output_dir"`
}

type GalleryImageDownloadResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	File          string `json:"file,omitempty"`
	Error         string `json:"error,omitempty"`
	AlreadyExists bool   `json:"already_exists,omitempty"`
}

type AvatarDownloadRequest struct {
	AvatarURL  string `json:"avatar_url"`
	ArtistName string `json:"artist_name"`
	OutputDir  string `json:"output_dir"`
}

type AvatarDownloadResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	File          string `json:"file,omitempty"`
	Error         string `json:"error,omitempty"`
	AlreadyExists bool   `json:"already_exists,omitempty"`
}

// --- Initialization ---

func NewCoverClient() *CoverClient {
	return &CoverClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// --- Exported Standalone Functions (For main.go) ---

func DownloadCover(req *CoverDownloadRequest) (*CoverDownloadResponse, error) {
	return NewCoverClient().DownloadCover(req)
}

func DownloadAvatar(req AvatarDownloadRequest) (*AvatarDownloadResponse, error) {
	return NewCoverClient().DownloadAvatar(req)
}

func DownloadHeader(req HeaderDownloadRequest) (*HeaderDownloadResponse, error) {
	return NewCoverClient().DownloadHeader(req)
}

func DownloadGalleryImage(req GalleryImageDownloadRequest) (*GalleryImageDownloadResponse, error) {
	return NewCoverClient().DownloadGalleryImage(req)
}

// --- Receiver Methods ---

func (c *CoverClient) DownloadCover(req *CoverDownloadRequest) (*CoverDownloadResponse, error) {
	if req.CoverURL == "" {
		return &CoverDownloadResponse{Success: false, Error: "Cover URL is required"}, fmt.Errorf("URL required")
	}

	// Spotify High-Res Logic
	if strings.Contains(req.CoverURL, "i.scdn.co") {
		re := regexp.MustCompile(`/ab67616d[0-9a-f]{8}`)
		req.CoverURL = re.ReplaceAllString(req.CoverURL, "/"+spotifySizeMax)
	}

	outputDir := req.OutputDir
	if outputDir == "" {
		outputDir = "downloads"
	}

	artistFolder := filepath.Join(outputDir, sanitizeFilename(req.ArtistName))
	_ = os.MkdirAll(artistFolder, 0755)

	filename := sanitizeFilename(req.TrackName) + "_Cover.jpg"
	filePath := filepath.Join(artistFolder, filename)

	return c.downloadFileGeneric(req.CoverURL, filePath)
}

func (c *CoverClient) DownloadAvatar(req AvatarDownloadRequest) (*AvatarDownloadResponse, error) {
	if req.AvatarURL == "" {
		return &AvatarDownloadResponse{Success: false, Error: "Avatar URL is required"}, fmt.Errorf("URL required")
	}

	artistFolder := filepath.Join(req.OutputDir, sanitizeFilename(req.ArtistName))
	_ = os.MkdirAll(artistFolder, 0755)

	filePath := filepath.Join(artistFolder, sanitizeFilename(req.ArtistName)+"_Avatar.jpg")
	
	res, err := c.downloadFileGeneric(req.AvatarURL, filePath)
	if err != nil {
		return &AvatarDownloadResponse{Success: false, Error: err.Error()}, err
	}
	return &AvatarDownloadResponse{Success: res.Success, File: res.File, Message: res.Message}, nil
}

func (c *CoverClient) DownloadHeader(req HeaderDownloadRequest) (*HeaderDownloadResponse, error) {
	if req.HeaderURL == "" {
		return &HeaderDownloadResponse{Success: false, Error: "Header URL is required"}, fmt.Errorf("URL required")
	}

	artistFolder := filepath.Join(req.OutputDir, sanitizeFilename(req.ArtistName))
	_ = os.MkdirAll(artistFolder, 0755)

	filePath := filepath.Join(artistFolder, sanitizeFilename(req.ArtistName)+"_Header.jpg")

	res, err := c.downloadFileGeneric(req.HeaderURL, filePath)
	if err != nil {
		return &HeaderDownloadResponse{Success: false, Error: err.Error()}, err
	}
	return &HeaderDownloadResponse{Success: res.Success, File: res.File, Message: res.Message}, nil
}

func (c *CoverClient) DownloadGalleryImage(req GalleryImageDownloadRequest) (*GalleryImageDownloadResponse, error) {
	if req.ImageURL == "" {
		return &GalleryImageDownloadResponse{Success: false, Error: "Gallery URL required"}, fmt.Errorf("URL required")
	}

	artistFolder := filepath.Join(req.OutputDir, sanitizeFilename(req.ArtistName))
	_ = os.MkdirAll(artistFolder, 0755)

	filename := fmt.Sprintf("%s_Gallery_%d.jpg", sanitizeFilename(req.ArtistName), req.ImageIndex+1)
	filePath := filepath.Join(artistFolder, filename)

	res, err := c.downloadFileGeneric(req.ImageURL, filePath)
	if err != nil {
		return &GalleryImageDownloadResponse{Success: false, Error: err.Error()}, err
	}
	return &GalleryImageDownloadResponse{Success: res.Success, File: res.File, Message: res.Message}, nil
}

// --- Shared Private Helper ---

func (c *CoverClient) downloadFileGeneric(url, filePath string) (*CoverDownloadResponse, error) {
	// Check if exists
	if info, err := os.Stat(filePath); err == nil && info.Size() > 0 {
		return &CoverDownloadResponse{Success: true, File: filePath, AlreadyExists: true, Message: "Exists"}, nil
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: %d", resp.StatusCode)
	}

	out, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return &CoverDownloadResponse{Success: true, File: filePath, Message: "Downloaded"}, err
}
func (c *CoverClient) DownloadCoverToPath(coverURL, outputPath string, embedMaxQualityCover bool) error {
	if coverURL == "" {
		return fmt.Errorf("cover URL is required")
	}

	downloadURL := convertSmallToMedium(coverURL)
	if embedMaxQualityCover {
		downloadURL = c.getMaxResolutionURL(downloadURL)
	}

	resp, err := c.httpClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download cover: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download cover: HTTP %d", resp.StatusCode)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write cover file: %v", err)
	}

	return nil
}
 func convertSmallToMedium(imageURL string) string {
	if strings.Contains(imageURL, spotifySize300) {
		return strings.Replace(imageURL, spotifySize300, spotifySize640, 1)
	}
	return imageURL
}

func (c *CoverClient) getMaxResolutionURL(imageURL string) string {

	mediumURL := convertSmallToMedium(imageURL)
	if strings.Contains(mediumURL, spotifySize640) {
		return strings.Replace(mediumURL, spotifySize640, spotifySizeMax, 1)
	}
	return mediumURL
}