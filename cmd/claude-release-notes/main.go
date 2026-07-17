// Command claude-release-notes fetches new anthropics/claude-code releases,
// translates them to Japanese, and emails them as one HTML digest. It takes
// no flags; all configuration comes from the YAML config file at the XDG
// default location.
package main

import (
	"os"

	"claude-release-notes/internal/app"
	"claude-release-notes/internal/config"
	"claude-release-notes/internal/github"
	"claude-release-notes/internal/mail"
	"claude-release-notes/internal/state"
	"claude-release-notes/internal/translate"
)

const (
	gmailSMTPHost = "smtp.gmail.com"
	gmailSMTPPort = 587
)

func main() {
	os.Exit(run())
}

// run wires the concrete leaf implementations into the app pipeline and
// maps its outcome to an exit code. app.Run writes all progress and error
// output to the provided writers, so run itself has no logic worth unit
// testing beyond compilation.
func run() int {
	deps := app.Dependencies{
		Config: app.ConfigLoaderFunc(func() (config.Config, error) {
			return config.Load("")
		}),
		State: state.NewStore(""),
		NewFetcher: func(token string) app.ReleaseFetcher {
			return github.NewClient(token, "")
		},
		Translator: translate.NewTranslator(""),
		Build:      mail.Build,
		NewSender: func(account, appPassword string) app.MailSender {
			return mail.NewSender(gmailSMTPHost, gmailSMTPPort, account, appPassword)
		},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := app.Run(deps); err != nil {
		return 1
	}
	return 0
}
