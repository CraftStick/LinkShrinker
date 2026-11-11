package video

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	MaxBytes = 50 << 20 // Bot API upload limit
	timeout  = 3 * time.Minute
)

var hosts = []string{
	"tiktok.com", "instagram.com", "youtube.com", "youtu.be", "twitter.com", "x.com",
	"facebook.com", "fb.watch", "vk.com", "vk.ru", "vkvideo.ru", "reddit.com",
	"pinterest.com", "pin.it", "twitch.tv", "likee.video", "snapchat.com", "threads.net",
}

var sem = make(chan struct{}, 2) // 2 GB RAM: at most two downloads at once

func Supported(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	h := strings.ToLower(u.Hostname())
	for _, s := range hosts {
		if h == s || strings.HasSuffix(h, "."+s) {
			return true
		}
	}
	return false
}

// Download returns the path to an mp4 and a cleanup func.
func Download(raw string) (string, func(), error) {
	sem <- struct{}{}
	defer func() { <-sem }()
	dir, err := os.MkdirTemp("", "vid")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(dir) }
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "yt-dlp", "--no-playlist", "--no-warnings",
		"--max-filesize", fmt.Sprint(MaxBytes),
		"-f", "bv*[height<=720][vcodec^=avc]+ba[ext=m4a]/b[height<=720][ext=mp4]/b[height<=720]/b",
		"--merge-output-format", "mp4", "-o", filepath.Join(dir, "v.%(ext)s"), "--", raw)
	out, err := cmd.CombinedOutput()
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("yt-dlp: %v: %s", err, tail(string(out)))
	}
	files, _ := filepath.Glob(filepath.Join(dir, "v.*"))
	if len(files) == 0 {
		cleanup()
		return "", nil, fmt.Errorf("no file (file too large?): %s", tail(string(out)))
	}
	if st, err := os.Stat(files[0]); err != nil || st.Size() > MaxBytes {
		cleanup()
		return "", nil, fmt.Errorf("file too large")
	}
	return files[0], cleanup, nil
}

func tail(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 300 {
		s = s[len(s)-300:]
	}
	return s
}
