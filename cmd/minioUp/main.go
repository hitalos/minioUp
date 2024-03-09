package main

import (
	"cmp"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/nexidian/gocliselect"

	"github.com/hitalos/minioUp/config"
	"github.com/hitalos/minioUp/services/minioClient"
)

var (
	onlyListing = flag.Bool("l", false, "Only list files (no upload)")
	configFile  = flag.String("c", "config.yml", "Config file")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n%[1]s [-c config.yml] <file1> <params1> [file2] [params2]…\nor\n%[1]s -l [-c config.yml]\n", os.Args[0])
}

func main() {
	flag.Parse()

	cfg := config.Config{}
	if err := cfg.Load(*configFile); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("config file not found\nExample at: https://github.com/hitalos/minioUp")
			os.Exit(1)
		}

		fmt.Println(err)
		os.Exit(1)
	}

	if len(cfg.Destinations) == 0 {
		fmt.Println("No destination(s) configured")
		os.Exit(1)
	}
	cfg.Dest = cfg.Destinations[0]

	if isTerminal(os.Stdin) {
		destIdx := chooseDestination(cfg.Destinations)
		if destIdx >= uint8(len(cfg.Destinations)) {
			os.Exit(0)
		}
		cfg.Dest = cfg.Destinations[destIdx]
	}

	if !*onlyListing {
		if len(flag.Args()) == 0 {
			usage()
			os.Exit(1)
		}

		upload(cfg)
		return
	}

	list(cfg)
}

func upload(cfg config.Config) {
	if len(flag.Args())%2 != 0 {
		fmt.Println(`Provide an even number of arguments: <file1> "<param 1>"`)
		os.Exit(1)
	}

	filepaths, params := []string{}, [][]string{}
	for i, p := range flag.Args() {
		if i%2 == 0 {
			filepaths = append(filepaths, p)
			continue
		}

		params = append(params, strings.Split(p, " "))
	}

	if len(filepaths) == 0 {
		fmt.Println("Provide at least one file")
		os.Exit(1)
	}

	fmt.Println("Uploading files…")
	if err := minioClient.Upload(cfg, filepaths, params); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Done!")
}

func list(cfg config.Config) {
	fmt.Println("Listing bucket/prefix content…")
	if err := minioClient.List(cfg); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func isTerminal(f *os.File) bool {
	o, _ := f.Stat()
	return (o.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

func chooseDestination(destinations []config.Destination) uint8 {
	menu := gocliselect.NewMenu("Choose a destination")
	for i, d := range destinations {
		menu.AddItem(cmp.Or[string](d.Name, d.Bucket), fmt.Sprintf("%d", i))
	}
	menu.AddItem("Cancel", fmt.Sprintf("%d", len(destinations)))

	menu.Display()

	return uint8(menu.CursorPos)
}
