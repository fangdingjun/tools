package main

type config struct {
	Token       string   `json:"token"`
	TrustUsers  []string `json:"trust_users"`
	PollTimeout int      `json:"poll_timeout"`
	PollLimit   int      `json:"poll_limit"`
	FileDir     string   `json:"file_save_dir"`
	Debug       bool     `json:"api_debug"`
}
