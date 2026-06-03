package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/kirby88/vix/internal/config"
)

func main() {
	cred := config.ResolveProviderCredential("anthropic", true)
	if cred.Value == "" {
		fmt.Println("no credential found")
		return
	}
	fmt.Printf("source: %s\n", cred.Source)

	body := `{"model":"claude-sonnet-4-6","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", strings.NewReader(body))
	req.Header.Set("content-type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	if cred.Source == "oauth-token" {
		req.Header.Set("Authorization", "Bearer "+cred.Value)
	} else {
		req.Header.Set("x-api-key", cred.Value)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("request error:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Println("status:", resp.Status)
	fmt.Println("ALL response headers:")
	for k, vv := range resp.Header {
		fmt.Printf("  %s: %s\n", k, strings.Join(vv, ", "))
	}
	b, _ := io.ReadAll(resp.Body)
	s := string(b)
	if len(s) > 400 {
		s = s[:400]
	}
	fmt.Println(s)
}
