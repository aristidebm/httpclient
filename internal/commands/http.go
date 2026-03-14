package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"httpclient/internal/executor"
	"httpclient/internal/model"
	"httpclient/internal/repl"
)

var acceptShortcuts = map[string]string{
	"json":  "application/json",
	"xml":   "application/xml",
	"pdf":   "application/pdf",
	"excel": "application/vnd.ms-excel",
	"xls":   "application/vnd.ms-excel",
	"xlsx":  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"csv":   "text/csv",
	"html":  "text/html",
	"text":  "text/plain",
	"zip":   "application/zip",
}

type httpCmd struct {
	method string
}

func (h *httpCmd) Name() string {
	return strings.ToLower(h.method)
}

func (h *httpCmd) Aliases() []string {
	return nil
}

func (h *httpCmd) Help() string {
	return fmt.Sprintf("Execute %s request", h.method)
}

func (h *httpCmd) Run(ctx *repl.ShellContext, args []string) error {
	// Find first non-flag argument as endpoint
	// and collect all flags with their values
	var endpoint string
	var flagArgs []string

	flagsWithValue := map[string]bool{
		"-p": true, "-d": true, "-H": true, "-a": true,
		"--timeout": true, "--note": true,
	}

	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			// Check if this flag expects a value
			if flagsWithValue[arg] && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else if endpoint == "" {
			endpoint = arg
		}
		i++
	}

	if endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}

	fs := flag.NewFlagSet(h.method, flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Printf("Usage: /%s [flags] <endpoint>\n", h.Name())
		fmt.Println("Flags:")
		fs.PrintDefaults()
	}

	var params multiValue
	var data multiValue
	var headers multiValue
	var accept string
	var form bool
	var timeout int
	var save bool
	var printTime bool
	var showDoc bool
	var note string

	fs.Var(&params, "p", "Query params (key:value or key=expr)")
	fs.Var(&data, "d", "Request body fields or **varname= for splat")
	fs.Var(&headers, "H", "Request headers (key:value)")
	fs.StringVar(&accept, "a", "", "Accept header shortcut")
	fs.BoolVar(&form, "form", false, "Send body as form-encoded")
	fs.IntVar(&timeout, "timeout", 0, "Per-request timeout in seconds")
	fs.BoolVar(&save, "save", false, "Auto-save binary response to file")
	fs.BoolVar(&printTime, "t", false, "Print execution time")
	fs.BoolVar(&showDoc, "doc", false, "Show OpenAPI docs instead of executing")
	fs.StringVar(&note, "note", "", "Attach a note to this request")

	err := fs.Parse(flagArgs)
	if err != nil {
		return err
	}

	env := ctx.Tree.CurrentEnv()
	if env == nil {
		return fmt.Errorf("no environment selected")
	}

	if showDoc && ctx.OpenAPI != nil {
		return showOpenAPIDoc(ctx, h.method, endpoint)
	}

	// Convert ctx.Vars to map[string]any for ResolveVars
	shellVars := make(map[string]any)
	for k, v := range ctx.Vars {
		shellVars[k] = v.Value
	}

	req := &model.Request{
		Method:      h.method,
		URL:         endpoint,
		Headers:     make(map[string]string),
		Params:      make(map[string]string),
		Vars:        ctx.Vars,
		ContentType: "application/json",
		Note:        note,
	}

	for _, p := range params {
		if strings.Contains(p, ":") {
			parts := strings.SplitN(p, ":", 2)
			req.Params[parts[0]] = parts[1]
		} else if strings.Contains(p, "=") {
			parts := strings.SplitN(p, "=", 2)
			resolved, _ := model.ResolveVars(parts[1], shellVars)
			req.Params[parts[0]] = resolved
		}
	}

	for _, hdr := range headers {
		parts := strings.SplitN(hdr, ":", 2)
		if len(parts) == 2 {
			req.Headers[parts[0]] = parts[1]
		}
	}

	if accept != "" {
		if shortcut, ok := acceptShortcuts[accept]; ok {
			req.Headers["Accept"] = shortcut
		} else {
			req.Headers["Accept"] = accept
		}
	}

	if len(data) > 0 {
		bodyMap := make(map[string]any)
		isSplat := false

		for _, d := range data {
			if strings.HasSuffix(d, "=") {
				varName := strings.TrimSuffix(d, "=")
				if val, ok := ctx.Vars.Get(varName); ok {
					if strVal, ok := val.Value.(string); ok {
						req.Body = []byte(strVal)
						isSplat = true
					} else {
						req.Body, _ = json.Marshal(val.Value)
						isSplat = true
					}
				}
				continue
			}

			if strings.Contains(d, ":") {
				parts := strings.SplitN(d, ":", 2)
				bodyMap[parts[0]] = parts[1]
			} else if strings.Contains(d, "=") {
				parts := strings.SplitN(d, "=", 2)
				resolved, _ := model.ResolveVars(parts[1], shellVars)
				bodyMap[parts[0]] = resolved
			}
		}

		if !isSplat && len(bodyMap) > 0 {
			if form {
				formData := url.Values{}
				for k, v := range bodyMap {
					formData.Set(k, fmt.Sprintf("%v", v))
				}
				req.Body = []byte(formData.Encode())
				req.ContentType = "application/x-www-form-urlencoded"
			} else {
				req.Body, _ = json.Marshal(bodyMap)
				req.ContentType = "application/json"
			}
		}
	}

	var client *executor.Client
	if timeout > 0 {
		client = executor.NewClient(time.Duration(timeout) * time.Second)
	} else {
		client = ctx.Executor
	}

	err = client.Execute(req, env)
	if err != nil {
		return err
	}

	session := ctx.Tree.Current()
	session.AddRequest(req)

	ctx.LastResp = req.Response
	ctx.LastReqID = req.ID

	if len(req.Response.Body) > 0 {
		var parsedData any
		json.Unmarshal(req.Response.Body, &parsedData)
		ctx.LastData = parsedData
	} else {
		ctx.LastData = string(req.Response.RawBody)
	}

	repl.PrintResponse(req.Response)

	if printTime {
		fmt.Printf("Time: %v\n", req.Duration)
	}

	if save && !isTextResponse(req.Response) {
		return saveBinaryResponse(req.Response, req.ID)
	}

	return nil
}

func (h *httpCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	// Complete OpenAPI routes if spec is loaded
	spec := ctx.CurrentSpec()
	if spec != nil && partial != "" {
		var routes []string
		for _, r := range spec.Routes {
			if strings.HasPrefix(r.Path, partial) {
				routes = append(routes, r.Path)
			}
		}
		return routes
	}
	return nil
}

type multiValue []string

func (m *multiValue) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func (m *multiValue) String() string {
	return ""
}

func showOpenAPIDoc(ctx *repl.ShellContext, method, path string) error {
	fmt.Printf("OpenAPI docs for %s %s not yet implemented\n", method, path)
	return nil
}

func isTextResponse(resp *model.Response) bool {
	ct := resp.Headers["Content-Type"]
	return strings.Contains(ct, "text/") ||
		strings.Contains(ct, "application/json") ||
		strings.Contains(ct, "application/xml")
}

func saveBinaryResponse(resp *model.Response, reqID string) error {
	filename := fmt.Sprintf("httpclient_%s_%d", reqID, time.Now().Unix())
	if ct := resp.Headers["Content-Type"]; strings.Contains(ct, "pdf") {
		filename += ".pdf"
	} else if strings.Contains(ct, "zip") {
		filename += ".zip"
	} else if strings.Contains(ct, "excel") || strings.Contains(ct, "spreadsheet") {
		filename += ".xlsx"
	} else if strings.Contains(ct, "ms-excel") {
		filename += ".xls"
	}

	err := os.WriteFile(filename, resp.RawBody, 0644)
	if err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}
	fmt.Printf("Saved to: %s\n", filename)
	return nil
}

type getCmd struct{ httpCmd }
type postCmd struct{ httpCmd }
type putCmd struct{ httpCmd }
type deleteCmd struct{ httpCmd }
type patchCmd struct{ httpCmd }

type requestCmd struct{}

func (c *requestCmd) Name() string      { return "request" }
func (c *requestCmd) Aliases() []string { return nil }
func (c *requestCmd) Help() string {
	return "Make arbitrary HTTP request: /request <method> <endpoint>"
}

func (c *requestCmd) Run(ctx *repl.ShellContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: /request <method> <endpoint>")
	}

	method := strings.ToUpper(args[0])
	endpoint := args[1]

	// Create a temporary httpCmd and run it
	http := &httpCmd{method: method}
	return http.Run(ctx, []string{endpoint})
}

func (c *requestCmd) Complete(ctx *repl.ShellContext, partial string) []string {
	return []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE"}
}

func init() {
	repl.Register(&getCmd{httpCmd{method: "GET"}})
	repl.Register(&postCmd{httpCmd{method: "POST"}})
	repl.Register(&putCmd{httpCmd{method: "PUT"}})
	repl.Register(&deleteCmd{httpCmd{method: "DELETE"}})
	repl.Register(&patchCmd{httpCmd{method: "PATCH"}})
	repl.Register(&requestCmd{})
}
