package runner

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/deploys-app/api"
)

func (rn Runner) email(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.Email()

	var (
		resp any
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)
	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
	case "send":
		var (
			req         api.EmailSend
			to          string
			typ         string
			content     string
			contentFile string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.From.Email, "from", "", "from email")
		f.StringVar(&req.From.Name, "from-name", "", "from name")
		f.StringVar(&to, "to", "", "to emails (comma separated values)")
		f.StringVar(&req.Subject, "subject", "", "subject")
		f.StringVar(&typ, "type", "text", "body type (text, html)")
		f.StringVar(&content, "content", "", "body content")
		f.StringVar(&contentFile, "content-file", "", "read body content from file")
		f.Parse(args[1:])

		for _, addr := range splitComma(to) {
			req.To = append(req.To, api.EmailAddr{Email: addr})
		}
		switch typ {
		case "text", string(api.EmailTypeText):
			req.Body.Type = api.EmailTypeText
		case "html", string(api.EmailTypeHTML):
			req.Body.Type = api.EmailTypeHTML
		default:
			return fmt.Errorf("invalid body type: '%s'", typ)
		}
		if contentFile != "" {
			b, ferr := os.ReadFile(contentFile)
			if ferr != nil {
				return ferr
			}
			content = string(b)
		}
		req.Body.Content = content
		resp, err = s.Send(context.Background(), &req)
	case "list":
		var req api.EmailList
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}
