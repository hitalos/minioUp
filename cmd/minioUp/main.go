package main

import (
	"cmp"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nexidian/gocliselect"

	"github.com/hitalos/minioUp/config"
	"github.com/hitalos/minioUp/services/minioClient"
)

var (
	onlyListing = flag.Bool("l", false, "Only list files (no upload)")
	configFile  = flag.String("c", "config.yml", "Config file")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n%[1]s [-c config.yml] <file1> <param1> <value1> <param2> <value2>…\nor\n%[1]s -l [-c config.yml]\n", os.Args[0])
}

func main() {
	flag.Parse()

	cfg := config.Config{}
	if err := cfg.Parse(*configFile); err != nil {
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

	dest := cfg.Destinations[0]

	if isTerminal(os.Stdin) {
		destIdx := chooseDestination(cfg.Destinations)
		qtd := len(cfg.Destinations)
		if qtd < 255 && destIdx >= uint8(qtd) {
			os.Exit(0)
		}
		dest = cfg.Destinations[destIdx]
	}

	if err := minioClient.Init(cfg); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if *onlyListing {
		list(dest)
	}

	if len(flag.Args()) == 0 {
		usage()
		os.Exit(1)
	}

	upload(dest)
}

func upload(dest config.Destination) {
	if (len(flag.Args())-1)%2 != 0 {
		fmt.Println(`Provide an even number of arguments: <file1> "<param 1>" "<value 1>" <param 2> "<value 2>"…`)
		os.Exit(1)
	}

	f, err := os.Open(flag.Args()[0])
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", flag.Args()[0], err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		fmt.Printf("Error getting file info for %s: %v\n", flag.Args()[0], err)
		os.Exit(1)
	}
	if info.IsDir() {
		fmt.Printf("Error: %s is a directory, not a file\n", flag.Args()[0])
		os.Exit(1)
	}

	filename := filepath.Base(flag.Args()[0])
	params := make(map[string]string, 0)
	for i := 1; i < len(flag.Args()); i += 2 {
		if i+1 >= len(flag.Args()) {
			fmt.Printf("Error: missing value for parameter %s\n", flag.Args()[i])
			os.Exit(1)
		}
		params[flag.Args()[i]] = flag.Args()[i+1]
	}

	fmt.Println("Uploading files…")
	if err := minioClient.Upload(dest, f, filename, info.Size(), params); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Done!")
}

func list(dest config.Destination) {
	fmt.Println("Listing bucket/prefix content…")

	list, err := minioClient.List(dest)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, obj := range list {
		fmt.Printf("%s\t%d\n", obj.Key[len(dest.Prefix)+1:], obj.Size)
	}
}

func isTerminal(f *os.File) bool {
	o, _ := f.Stat()
	return (o.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

func chooseDestination(destinations []config.Destination) uint8 {
	menu := gocliselect.NewMenu("Choose a destination")
	for i, d := range destinations {
		menu.AddItem(cmp.Or(d.Name, d.Bucket), fmt.Sprintf("%d", i))
	}
	menu.AddItem("Cancel", fmt.Sprintf("%d", len(destinations)))

	menu.Display()

	if menu.CursorPos < 0 || menu.CursorPos > 255 {
		fmt.Printf("error choosing destination: %d\n", menu.CursorPos)
		os.Exit(1)
	}

	return uint8(menu.CursorPos) // #nosec G115
}
