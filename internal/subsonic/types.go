package subsonic

import (
	"bytes"
	"encoding/json"
	"strings"
)

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Song struct {
	ID                 string `json:"id"`
	Parent             string `json:"parent"`
	Title              string `json:"title"`
	Album              string `json:"album"`
	Artist             string `json:"artist"`
	AlbumArtist        string `json:"albumArtist"`
	DisplayAlbumArtist string `json:"displayAlbumArtist"`
	DisplayArtist      string `json:"displayArtist"`
	Track              int    `json:"track"`
	Year               int    `json:"year"`
	Genre              string `json:"genre"`
	Duration           int    `json:"duration"`
	BitRate            int    `json:"bitRate"`
	Suffix             string `json:"suffix"`
	ContentType        string `json:"contentType"`
	Size               int64  `json:"size"`
	CoverArt           string `json:"coverArt"`
	SamplingRate       int    `json:"samplingRate"`
	BitDepth           int    `json:"bitDepth"`
	IsDir              bool   `json:"isDir"`
}

func (s Song) EffectiveArtist() string {
	if s.DisplayArtist != "" {
		return s.DisplayArtist
	}
	return s.Artist
}

func (s Song) Codec() string {
	switch strings.ToLower(s.Suffix) {
	case "flac":
		return "flac"
	case "alac":
		return "alac"
	case "m4a":
		if strings.Contains(strings.ToLower(s.ContentType), "alac") {
			return "alac"
		}
		return "aac"
	case "mp3":
		return "mp3"
	case "aac":
		return "aac"
	case "opus":
		return "opus"
	case "ogg", "oga":
		ct := strings.ToLower(s.ContentType)
		if strings.Contains(ct, "flac") {
			return "flac"
		}
		if strings.Contains(ct, "opus") {
			return "opus"
		}
		return "vorbis"
	case "wav", "aiff", "aif":
		return "pcm"
	case "wma":
		return "wma"
	case "wv":
		return "wavpack"
	case "ape":
		return "ape"
	case "dsf", "dff":
		return "dsd"
	default:
		if s.Suffix != "" {
			return strings.ToLower(s.Suffix)
		}
		if i := strings.Index(s.ContentType, "/"); i >= 0 {
			return strings.ToLower(s.ContentType[i+1:])
		}
		return ""
	}
}

func (s Song) QualityLabel() string {
	codec := s.Codec()
	isLossless := codec == "flac" || codec == "alac" || codec == "pcm" || codec == "wavpack" || codec == "ape"
	if isLossless {
		if s.BitDepth > 16 || s.SamplingRate > 48000 {
			return "Hi-Res Lossless"
		}
		return "Lossless"
	}
	if s.BitRate >= 256 {
		return "High"
	}
	return "Low"
}

type SongList []Song

func (l *SongList) UnmarshalJSON(b []byte) error {
	trimmed := bytes.TrimSpace(b)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*l = nil
		return nil
	}
	if trimmed[0] == '[' {
		var list []Song
		if err := json.Unmarshal(trimmed, &list); err != nil {
			return err
		}
		*l = list
		return nil
	}
	var single Song
	if err := json.Unmarshal(trimmed, &single); err != nil {
		return err
	}
	*l = []Song{single}
	return nil
}

type LyricsListResponse struct {
	LyricsList struct {
		StructuredLyrics []StructuredLyric `json:"structuredLyrics"`
	} `json:"lyricsList"`
}

type StructuredLyric struct {
	Lang          string       `json:"lang"`
	Synced        bool         `json:"synced"`
	Offset        int          `json:"offset"`
	DisplayTitle  string       `json:"displayTitle"`
	DisplayArtist string       `json:"displayArtist"`
	Lines         []LyricsLine `json:"line"`
}

type LyricsLine struct {
	Start int    `json:"start"`
	Value string `json:"value"`
}

type StandardLyricsResponse struct {
	Lyrics struct {
		Artist  string `json:"artist"`
		Title   string `json:"title"`
		Content string `json:"content"`
	} `json:"lyrics"`
}
