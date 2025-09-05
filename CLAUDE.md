# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based ARP table monitoring application that tracks network device MAC addresses and detects changes. The application provides a web interface for network scanning and MAC address management.

### Key Features
- Web interface for immediate network scanning
- Network range configuration (e.g., 10.2.10.0/24)
- TCP port configuration (e.g., 80, 443, 8080)
- Three-step scanning process: clear local ARP table → connect to TCP ports → read ARP table
- SQLite3 database storage with timestamps
- MAC address change detection and alerting
- IP/MAC maintenance interface

### Technology Stack
- **Backend**: Go with Fiber web framework
- **Database**: SQLite3 with GORM ORM
- **Platform**: Windows (uses `arp -a` command)

## Development Commands

### Basic Go Commands
```bash
# Build the application
go build

# Run the application
go run main.go

# Format code
go fmt ./...

# Vet code for issues
go vet ./...

# Run tests
go test ./...

# Install dependencies
go mod tidy

# Download dependencies
go mod download
```

## Project Structure

- `main.go` - Main entry point (currently empty function)
- `go.mod` - Go module definition
- `ipmac/` - Main package directory (currently empty)

## Architecture Notes

The application is designed to:
1. Use a web interface built with Go Fiber framework
2. Store data in SQLite3 database using GORM
3. Perform network scanning without traditional ARP scan tools
4. Track MAC address changes over time with timestamps
5. Provide maintenance interface for IP/MAC associations

The scanning process bypasses traditional ARP scanning by:
1. Clearing the local ARP table
2. Attempting TCP connections to specified ports
3. Reading the ARP table after connections populate it

## Database Schema

The application stores scan results with timestamps in the format: YYYY-MM-DD HH:MM:SS, allowing for historical tracking and change detection.