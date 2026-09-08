package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/ryanburnette/deadman/internal/config"
	"github.com/ryanburnette/deadman/internal/state"
)

func addCmd(args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	configPath := fs.String("config", getEnv("CONFIG_PATH", "./services.csv"), "services config path")
	interval := fs.Duration("interval", 0, "how often the service is expected to check in (e.g. 5m)")
	grace := fs.Duration("grace", time.Minute, "grace period after the interval before alerting")
	repeat := fs.Duration("repeat", 0, "resend the down alert at this interval while still down (0 = alert once)")
	email := fs.String("email", "", "notification email address")

	for _, a := range args {
		if a == "-h" || a == "-help" || a == "--help" {
			fmt.Println("Usage: deadman add [options] <name>")
			fmt.Println()
			fmt.Println("Options:")
			fs.SetOutput(os.Stdout)
			fs.PrintDefaults()
			os.Exit(0)
		}
	}
	fs.Parse(args)

	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "Usage: deadman add [options] <name>")
		os.Exit(1)
	}
	name := rest[0]

	if *interval <= 0 {
		fmt.Fprintln(os.Stderr, "error: -interval is required and must be positive")
		os.Exit(1)
	}
	if *email == "" {
		fmt.Fprintln(os.Stderr, "error: -email is required")
		os.Exit(1)
	}

	services, err := config.Load(*configPath)
	fatalIf(err)
	if _, ok := config.Find(services, name); ok {
		fmt.Fprintf(os.Stderr, "error: service %q already exists\n", name)
		os.Exit(1)
	}

	token, err := config.NewToken()
	fatalIf(err)

	services = append(services, config.Service{
		Name:     name,
		Token:    token,
		Interval: *interval,
		Grace:    *grace,
		Repeat:   *repeat,
		Email:    *email,
	})
	fatalIf(config.Save(*configPath, services))

	fmt.Printf("added %s\n", name)
	printURL(name, token)
}

func removeCmd(args []string) {
	fs := flag.NewFlagSet("remove", flag.ExitOnError)
	configPath := fs.String("config", getEnv("CONFIG_PATH", "./services.csv"), "services config path")
	statePath := fs.String("state", getEnv("STATE_PATH", "./state.json"), "state file path")

	for _, a := range args {
		if a == "-h" || a == "-help" || a == "--help" {
			fmt.Println("Usage: deadman remove [options] <name>")
			fmt.Println()
			fmt.Println("Options:")
			fs.SetOutput(os.Stdout)
			fs.PrintDefaults()
			os.Exit(0)
		}
	}
	fs.Parse(args)

	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "Usage: deadman remove [options] <name>")
		os.Exit(1)
	}
	name := rest[0]

	services, err := config.Load(*configPath)
	fatalIf(err)
	if _, ok := config.Find(services, name); !ok {
		fmt.Fprintf(os.Stderr, "error: no such service %q\n", name)
		os.Exit(1)
	}
	kept := services[:0]
	for _, s := range services {
		if s.Name != name {
			kept = append(kept, s)
		}
	}
	fatalIf(config.Save(*configPath, kept))

	st, err := state.Load(*statePath)
	fatalIf(err)
	delete(st, name)
	fatalIf(state.Save(*statePath, st))

	fmt.Printf("removed %s\n", name)
}

func setCmd(args []string) {
	fs := flag.NewFlagSet("set", flag.ExitOnError)
	configPath := fs.String("config", getEnv("CONFIG_PATH", "./services.csv"), "services config path")
	interval := fs.Duration("interval", 0, "how often the service is expected to check in")
	grace := fs.Duration("grace", 0, "grace period after the interval before alerting")
	repeat := fs.Duration("repeat", -1, "resend the down alert at this interval while still down (0 = alert once)")
	email := fs.String("email", "", "notification email address")
	regenToken := fs.Bool("regen-token", false, "generate a new check-in token")

	for _, a := range args {
		if a == "-h" || a == "-help" || a == "--help" {
			fmt.Println("Usage: deadman set [options] <name>")
			fmt.Println()
			fmt.Println("Options:")
			fs.SetOutput(os.Stdout)
			fs.PrintDefaults()
			os.Exit(0)
		}
	}
	fs.Parse(args)

	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "Usage: deadman set [options] <name>")
		os.Exit(1)
	}
	name := rest[0]

	services, err := config.Load(*configPath)
	fatalIf(err)
	svc, ok := config.Find(services, name)
	if !ok {
		fmt.Fprintf(os.Stderr, "error: no such service %q\n", name)
		os.Exit(1)
	}

	if *interval > 0 {
		svc.Interval = *interval
	}
	if *grace > 0 {
		svc.Grace = *grace
	}
	if *repeat >= 0 {
		svc.Repeat = *repeat
	}
	if *email != "" {
		svc.Email = *email
	}
	if *regenToken {
		token, err := config.NewToken()
		fatalIf(err)
		svc.Token = token
	}

	for i, s := range services {
		if s.Name == name {
			services[i] = svc
		}
	}
	fatalIf(config.Save(*configPath, services))

	fmt.Printf("updated %s\n", name)
	if *regenToken {
		printURL(name, svc.Token)
	}
}

func listCmd(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	configPath := fs.String("config", getEnv("CONFIG_PATH", "./services.csv"), "services config path")
	statePath := fs.String("state", getEnv("STATE_PATH", "./state.json"), "state file path")

	for _, a := range args {
		if a == "-h" || a == "-help" || a == "--help" {
			fmt.Println("Usage: deadman list [options]")
			fmt.Println()
			fmt.Println("Options:")
			fs.SetOutput(os.Stdout)
			fs.PrintDefaults()
			os.Exit(0)
		}
	}
	fs.Parse(args)

	services, err := config.Load(*configPath)
	fatalIf(err)
	st, err := state.Load(*statePath)
	fatalIf(err)

	if len(services) == 0 {
		fmt.Println("no services configured")
		return
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tSTATUS\tLAST SEEN\tINTERVAL\tGRACE\tREPEAT\tEMAIL")
	for _, s := range services {
		status := "unknown"
		lastSeen := "-"
		if ss, ok := st[s.Name]; ok {
			status = string(ss.Status)
			lastSeen = ss.LastSeen.Format(time.RFC3339)
		}
		repeat := "once"
		if s.Repeat > 0 {
			repeat = s.Repeat.String()
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", s.Name, status, lastSeen, s.Interval, s.Grace, repeat, s.Email)
	}
	tw.Flush()
}

func urlCmd(args []string) {
	fs := flag.NewFlagSet("url", flag.ExitOnError)
	configPath := fs.String("config", getEnv("CONFIG_PATH", "./services.csv"), "services config path")

	for _, a := range args {
		if a == "-h" || a == "-help" || a == "--help" {
			fmt.Println("Usage: deadman url [options] <name>")
			fmt.Println()
			fmt.Println("Options:")
			fs.SetOutput(os.Stdout)
			fs.PrintDefaults()
			os.Exit(0)
		}
	}
	fs.Parse(args)

	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "Usage: deadman url [options] <name>")
		os.Exit(1)
	}
	name := rest[0]

	services, err := config.Load(*configPath)
	fatalIf(err)
	svc, ok := config.Find(services, name)
	if !ok {
		fmt.Fprintf(os.Stderr, "error: no such service %q\n", name)
		os.Exit(1)
	}
	printURL(svc.Name, svc.Token)
}

func printURL(name, token string) {
	path := fmt.Sprintf("/checkin/%s/%s", name, token)
	base := os.Getenv("PUBLIC_URL")
	if base == "" {
		fmt.Printf("check-in path: %s\n", path)
		fmt.Println("(set PUBLIC_URL to print the full URL)")
		return
	}
	fmt.Printf("check-in url: %s%s\n", base, path)
}

func fatalIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
