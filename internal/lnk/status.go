package lnk

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// LinkInfo represents information about a symlink
type LinkInfo struct {
	Link     string `json:"link"`
	Target   string `json:"target"`
	IsBroken bool   `json:"is_broken"`
	Source   string `json:"source"` // Source mapping name (e.g., "home", "work")
}

// StatusOutput represents the complete status output for JSON formatting
type StatusOutput struct {
	Links   []LinkInfo `json:"links"`
	Summary struct {
		Total  int `json:"total"`
		Active int `json:"active"`
		Broken int `json:"broken"`
	} `json:"summary"`
}

// Status displays the status of all managed symlinks
func Status(config *Config) error {
	// Only print header in human format
	if !IsJSONFormat() {
		PrintCommandHeader("Symlink Status")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Find all symlinks pointing to configured source directories
	managedLinks, err := FindManagedLinks(homeDir, config)
	if err != nil {
		return fmt.Errorf("failed to find managed links: %w", err)
	}

	// Convert to LinkInfo
	var links []LinkInfo
	for _, ml := range managedLinks {
		link := LinkInfo{
			Link:     ml.Path,
			Target:   ml.Target,
			IsBroken: ml.IsBroken,
			Source:   ml.Source,
		}
		links = append(links, link)
	}

	// Sort by link path
	sort.Slice(links, func(i, j int) bool {
		return links[i].Link < links[j].Link
	})

	// If JSON format is requested, output JSON and return
	if IsJSONFormat() {
		return outputStatusJSON(links)
	}

	// Display links
	if len(links) > 0 {
		// Separate active and broken links
		var activeLinks, brokenLinks []LinkInfo
		for _, link := range links {
			if link.IsBroken {
				brokenLinks = append(brokenLinks, link)
			} else {
				activeLinks = append(activeLinks, link)
			}
		}

		// Display active links
		if len(activeLinks) > 0 {
			for _, link := range activeLinks {
				if ShouldSimplifyOutput() {
					// For piped output, use simple format
					fmt.Printf("active %s\n", ContractPath(link.Link))
				} else {
					PrintSuccess("Active: %s", ContractPath(link.Link))
				}
			}
		}

		// Display broken links
		if len(brokenLinks) > 0 {
			if len(activeLinks) > 0 && !ShouldSimplifyOutput() {
				fmt.Println()
			}
			for _, link := range brokenLinks {
				if ShouldSimplifyOutput() {
					// For piped output, use simple format
					fmt.Printf("broken %s\n", ContractPath(link.Link))
				} else {
					PrintError("Broken: %s", ContractPath(link.Link))
				}
			}
		}

		// Summary
		if !ShouldSimplifyOutput() {
			fmt.Println()
			PrintInfo("Total: %s (%s active, %s broken)",
				Bold(fmt.Sprintf("%d links", len(links))),
				Green(fmt.Sprintf("%d", len(activeLinks))),
				Red(fmt.Sprintf("%d", len(brokenLinks))))
		}
	} else {
		PrintEmptyResult("active links")
	}

	return nil
}

// outputStatusJSON outputs the status in JSON format
func outputStatusJSON(links []LinkInfo) error {
	// Ensure links is not nil for proper JSON output
	if links == nil {
		links = []LinkInfo{}
	}

	output := StatusOutput{
		Links: links,
	}

	// Calculate summary
	for _, link := range links {
		output.Summary.Total++
		if link.IsBroken {
			output.Summary.Broken++
		} else {
			output.Summary.Active++
		}
	}

	// Marshal to JSON with pretty printing
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal status to JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}
