package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/openfluke/lpd/internal/catalog"
	"github.com/openfluke/lpd/internal/env"
	"github.com/openfluke/lpd/internal/run"
)

func main() {
	envFile := flag.String("env", ".env", "auto-start config file")
	testN := flag.Int("test", 0, "test id (0 = menu or .env TEST)")
	menu := flag.Bool("menu", false, "show interactive menu")
	flag.Parse()

	cfg := env.Load(*envFile)
	applyFlags(&cfg, *testN)

	ctx, stop := run.MainContext()
	defer stop()

	if cfg.OceanOnly {
		if err := run.OceanWatcher(cfg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	id := cfg.Test
	if *menu || (id == 0 && *testN == 0 && os.Getenv("LPD_TEST") == "") {
		id = pickMenu()
		if id == 0 {
			return
		}
		if id == 9 {
			cfg.OceanOnly = true
			if err := run.OceanWatcher(cfg); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
	}

	test, err := catalog.ByID(id)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if err := run.Test(ctx, run.Options{Config: cfg, Test: test}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func applyFlags(cfg *env.Config, testFlag int) {
	if testFlag > 0 {
		cfg.Test = testFlag
	}
	if f := flag.Lookup("env"); f != nil && len(os.Args) > 1 {
		for _, a := range os.Args[1:] {
			if strings.HasPrefix(a, "-test=") {
				break
			}
		}
	}
}

func pickMenu() int {
	catalog.Menu()
	fmt.Print("\nSelect test: ")
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(sc.Text()))
	if err != nil {
		return 0
	}
	return n
}
