package main

import (
	"embed"
	"flag"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/masterdatamate/masterdatamate/internal/host"
)

//go:embed dist
var embeddedDist embed.FS

func main() {
	if len(os.Args) > 1 && os.Args[1] == "export" {
		os.Exit(host.RunExportCommand(os.Args[2:], os.Stdout, os.Stderr))
	}
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "serve" {
		args = args[1:]
	}
	var bindHost string
	var port int
	var root string
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	fs.StringVar(&bindHost, "host", "127.0.0.1", "bind host")
	fs.IntVar(&port, "port", envInt("PORT", 8787), "bind port")
	fs.StringVar(&root, "workspace", "", "workspace root containing masterdata")
	fs.Parse(args)

	workspace := root
	var err error
	if workspace == "" {
		workspace, err = host.ResolveWorkspace(".")
		if err != nil {
			log.Fatal(err)
		}
	}
	addr := net.JoinHostPort(bindHost, strconv.Itoa(port))
	log.Printf("MasterDataMate server listening on http://%s", addr)
	if workspace, err = host.NewWorkspacePath(workspace); err != nil {
		log.Fatal(err)
	}
	log.Printf("Workspace root: %s", workspace)
	if err := host.ListenAndServe(bindHost, port, workspace, embeddedDist); err != nil {
		log.Fatal(err)
	}
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil {
		return fallback
	}
	return value
}
