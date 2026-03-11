package repl

import (
	"fmt"
	"time"

	"cdapi/internal/model"
	"github.com/fatih/color"
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
		fmt.Println(string(resp.Body))
	}
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
