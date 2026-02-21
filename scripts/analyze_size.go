package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type PackageSize struct {
	Name string
	Size int64
}

func main() {
	binaryPath := "wx_channel.exe"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		fmt.Printf("Error: %s not found. Please build it first.\n", binaryPath)
		return
	}

	fmt.Printf("Analyzing %s...\n", binaryPath)

	cmd := exec.Command("go", "tool", "nm", "-size", binaryPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error creating pipe: %v\n", err)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Error starting command: %v\n", err)
		return
	}

	packageSizes := make(map[string]int64)
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}

		sizeStr := parts[1]
		typ := parts[2]
		name := parts[3]

		if !strings.ContainsAny(typ, "TtDdRr") {
			continue
		}

		size, err := strconv.ParseInt(sizeStr, 16, 64)
		if err != nil {
			continue
		}

		// Skip metadata/debug/virtual symbols
		if size > 50*1024*1024 || strings.HasPrefix(name, ".debug") || strings.EqualFold(name, "runtime.rodata") || strings.EqualFold(name, "runtime.types") {
			continue
		}

		pkg := getPackageName(name)
		if pkg == "unknown" && size > 100000 {
			// If it's a large unknown symbol, might be C code
			if strings.HasPrefix(name, "sqlite3") {
				pkg = "c_library/sqlite3"
			}
		}
		packageSizes[pkg] += size
	}

	if err := cmd.Wait(); err != nil {
		// Capture stderr to check for "no symbols"
		if strings.Contains(err.Error(), "exit status 1") {
			fmt.Printf("\nError: Analysis failed. This binary (%s) appears to be stripped (no symbols).\n", binaryPath)
			fmt.Println("Analysis requires a binary built with symbol information.")
			fmt.Println("Please build without '-s -w' flags if you want to analyze module sizes.")
		} else {
			fmt.Printf("Error waiting for command: %v\n", err)
		}
		return
	}

	// Create a new map to aggregate renamed packages
	finalSizes := make(map[string]int64)
	var totalSize int64
	for pkg, size := range packageSizes {
		name := pkg
		if name == "" || name == "unknown" {
			name = "other/C_code"
		}
		finalSizes[name] += size
		totalSize += size
	}

	var sortedSizes []PackageSize
	for name, size := range finalSizes {
		sortedSizes = append(sortedSizes, PackageSize{Name: name, Size: size})
	}

	sort.Slice(sortedSizes, func(i, j int) bool {
		return sortedSizes[i].Size > sortedSizes[j].Size
	})

	f, err := os.Create("size_analysis_report.md")
	if err != nil {
		fmt.Printf("Error creating report file: %v\n", err)
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "# Binary Size Analysis: %s\n\n", binaryPath)
	fmt.Fprintf(f, "This report shows the estimated size contribution of different packages to the compiled binary.\n\n")
	fmt.Fprintf(f, "| Package | Size (MB) | Percentage |\n")
	fmt.Fprintf(f, "| :--- | :--- | :--- |\n")

	for _, ps := range sortedSizes {
		percentage := float64(ps.Size) / float64(totalSize) * 100
		if percentage < 0.05 {
			continue
		}
		sizeMB := float64(ps.Size) / (1024 * 1024)
		fmt.Fprintf(f, "| %s | %.2f MB | %.2f%% |\n", ps.Name, sizeMB, percentage)
	}

	fmt.Fprintf(f, "\n**Estimated Total Symbol Size:** %.2f MB\n", float64(totalSize)/(1024*1024))
	fileInfo, _ := os.Stat(binaryPath)
	fmt.Fprintf(f, "**Actual File Size:** %.2f MB\n", float64(fileInfo.Size())/(1024*1024))

	fmt.Println("Analysis complete. Result written to size_analysis_report.md")
}

func getPackageName(symbolName string) string {
	// Native SQLite symbols
	if strings.HasPrefix(symbolName, "sqlite3") {
		return "c_library/sqlite3"
	}

	if strings.Contains(symbolName, "github.com/") {
		parts := strings.Split(symbolName, "/")
		if len(parts) >= 3 {
			repo := strings.Join(parts[:3], "/")
			return repo
		}
	}

	if strings.Contains(symbolName, "wx_channel/") {
		i := strings.Index(symbolName, "wx_channel/")
		rest := symbolName[i+11:]
		j := strings.Index(rest, ".")
		if j != -1 {
			return "wx_channel/" + rest[:j]
		}
		return "wx_channel_internal"
	}

	if strings.HasPrefix(symbolName, "runtime.") || strings.HasPrefix(symbolName, "runtime/") {
		return "runtime"
	}

	lastDot := -1
	inParens := 0
	for i, char := range symbolName {
		if char == '(' {
			inParens++
		} else if char == ')' {
			inParens--
		} else if char == '.' && inParens == 0 {
			lastDot = i
		}
	}

	if lastDot != -1 {
		pkg := symbolName[:lastDot]
		if strings.HasPrefix(pkg, "type..") || strings.HasPrefix(pkg, "go..") {
			return "go_system"
		}
		return pkg
	}

	if strings.HasPrefix(symbolName, ".") {
		return "system_sections"
	}

	return "unknown"
}
