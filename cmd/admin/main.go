package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"netconnector/internal/server"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./mappings.db"
	}

	db, err := server.NewSQLiteDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "add":
		addCmd := flag.NewFlagSet("add", flag.ExitOnError)
		addCmd.Parse(os.Args[2:])
		args := addCmd.Args()

		if len(args) != 2 {
			fmt.Println("Usage: admin add <subdomain> <client_id>")
			os.Exit(1)
		}

		subdomain := args[0]
		clientID := args[1]

		err = db.AddMapping(subdomain, clientID)
		if err != nil {
			log.Fatalf("Error adding mapping: %v", err)
		}
		fmt.Printf("Successfully mapped %s -> %s\n", subdomain, clientID)

	case "remove":
		rmCmd := flag.NewFlagSet("remove", flag.ExitOnError)
		rmCmd.Parse(os.Args[2:])
		args := rmCmd.Args()

		if len(args) != 1 {
			fmt.Println("Usage: admin remove <subdomain>")
			os.Exit(1)
		}

		subdomain := args[0]
		err = db.RemoveMapping(subdomain)
		if err != nil {
			log.Fatalf("Error removing mapping: %v", err)
		}
		fmt.Printf("Successfully removed mapping for %s\n", subdomain)

	case "list":
		mappings, err := db.ListMappings()
		if err != nil {
			log.Fatalf("Error listing mappings: %v", err)
		}

		fmt.Println("--- Current Allocations ---")
		for sub, cid := range mappings {
			fmt.Printf("%s\t->\t%s\n", sub, cid)
		}
		fmt.Println("---------------------------")

	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Netconnector Admin CLI")
	fmt.Println("Usage:")
	fmt.Println("  admin add <subdomain> <client_id>  # Map a subdomain to a client")
	fmt.Println("  admin remove <subdomain>           # Remove a subdomain mapping")
	fmt.Println("  admin list                         # List active mappings")
}
