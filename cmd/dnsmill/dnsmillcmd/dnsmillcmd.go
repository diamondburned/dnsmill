package dnsmillcmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/spf13/pflag"
	"libdb.so/dnsmill"
)

var (
	verbose       = false
	dryRun        = false
	jsonLog       = false
	format        = ""
	listProviders = false
)

func init() {
	log.SetFlags(0)

	pflag.BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")
	pflag.BoolVar(&dryRun, "dry-run", false, "enable dry-run mode")
	pflag.BoolVarP(&jsonLog, "json-log", "j", false, "log in JSON output instead of text")
	pflag.StringVarP(&format, "format", "f", "yaml", "profile format (json or yaml, empty to autodetect)")
	pflag.BoolVar(&listProviders, "list-providers", false, "list available DNS providers then exit")

	pflag.Usage = func() {
		log.Printf("Usage:")
		log.Printf("  %s [flags] <profile-path>\n", os.Args[0])
		log.Printf("Flags:")
		pflag.PrintDefaults()
	}
}

// Main is the entry point for the dnsmill command.
func Main() {
	pflag.Parse()

	if listProviders {
		printProviders()
		os.Exit(0)
	}

	if len(pflag.Args()) != 1 {
		pflag.Usage()
		os.Exit(1)
	}

	slogLevel := slog.LevelInfo
	if verbose {
		slogLevel = slog.LevelDebug
	}

	slogHandler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:   slogLevel,
		NoColor: os.Getenv("NO_COLOR") != "" || !isatty.IsTerminal(os.Stderr.Fd()),
	})

	logger := slog.New(slogHandler)
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	profilePath := pflag.Arg(0)
	if !run(ctx, logger, profilePath) {
		os.Exit(1)
	}
}

func printProviders() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, provider := range dnsmill.ListProviders() {
		fmt.Fprintf(w, "%s\t%s\n", provider.Name, provider.DocURL)
	}
	if err := w.Flush(); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, logger *slog.Logger, profilePath string) bool {
	logger = logger.With("profile", profilePath)

	if format == "" {
		switch strings.ToLower(filepath.Ext(profilePath)) {
		case ".json":
			format = "json"
		case ".yaml", ".yml":
			format = "yaml"
		}
	}

	var parseProfile func(io.Reader) (*dnsmill.Profile, error)
	switch format {
	case "json":
		parseProfile = dnsmill.ParseProfileAsJSON
	case "yaml":
		parseProfile = dnsmill.ParseProfileAsYAML
	default:
		logger.Error(
			"unsupported profile format",
			"format", format)
	}

	f, err := os.Open(profilePath)
	if err != nil {
		logger.Error("failed to open profile", tint.Err(err))
		return false
	}
	defer f.Close()

	p, err := parseProfile(f)
	if err != nil {
		logger.Error("failed to parse profile", tint.Err(err))
		return false
	}

	// close file early, we don't need it anymore
	f.Close()

	if err := p.Apply(ctx, logger, dryRun); err != nil {
		logger.Error("failed to apply profile", tint.Err(err))
		return false
	}

	return true
}
