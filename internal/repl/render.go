package repl

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fatih/color"
	"httpclient/internal/model"
)

func PrintResponse(resp *model.Response) {
	if resp == nil {
		return
	}

	statusColor := color.FgGreen
	switch {
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		statusColor = color.FgCyan
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		statusColor = color.FgYellow
	case resp.StatusCode >= 500:
		statusColor = color.FgRed
	}

	statusLine := fmt.Sprintf("%d %s", resp.StatusCode, resp.Status)
	fmt.Println(color.New(statusColor).Sprint(statusLine))

	if len(resp.Body) > 0 {
		// Check for binary response
		ct := resp.Headers["Content-Type"]
		if isBinaryContentType(ct) {
			size := len(resp.RawBody)
			fmt.Printf("Content-Type: %s\n", ct)
			fmt.Printf("Size: %d bytes\n", size)
			fmt.Println("Binary response. Use /save to download.")
			return
		}

		// Check for long response - page it
		if len(resp.Body) > 4096 && isTTY() {
			pager := os.Getenv("PAGER")
			if pager == "" {
				pager = "less"
			}
			cmd := exec.Command(pager)
			cmd.Stdin = strings.NewReader(string(resp.Body))
			cmd.Stdout = os.Stdout
			cmd.Run()
			return
		}

		fmt.Println(string(resp.Body))
	}
}

func isBinaryContentType(ct string) bool {
	if ct == "" {
		return false
	}
	ct = strings.ToLower(ct)
	textTypes := []string{"text/", "application/json", "application/xml", "application/javascript"}
	for _, t := range textTypes {
		if strings.Contains(ct, t) {
			return false
		}
	}
	return true
}

func isTTY() bool {
	return false // Simplified - pager not automatically invoked
}

func PrintRequest(req *model.Request) {
	if req == nil {
		return
	}

	note := ""
	if req.Note != "" {
		note = fmt.Sprintf(" — %s", req.Note)
	}

	status := ""
	if req.Response != nil {
		status = fmt.Sprintf(" → %d %s (%s)",
			req.Response.StatusCode,
			req.Response.Status,
			req.Duration.Round(time.Millisecond).String())
	}

	fmt.Printf("[%s] %s %s%s%s\n",
		req.ID,
		req.Method,
		req.URL,
		status,
		note)
}

func PrintError(err error) {
	if err == nil {
		return
	}
	color.New(color.FgRed).Printf("Error: %v\n", err)
}

func PrintInfo(msg string) {
	color.New(color.FgBlue).Println(msg)
}

func PrintSuccess(msg string) {
	color.New(color.FgGreen).Println(msg)
}
