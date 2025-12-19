package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type outputFormat string

const (
	formatDefault outputFormat = "default"
	formatJSON    outputFormat = "json"
	formatYAML    outputFormat = "yaml"
)

func detectOutputFormat(args []string) outputFormat {
	for i, arg := range args {
		if arg == "-o" || arg == "--output" {
			if i+1 < len(args) {
				format := strings.ToLower(args[i+1])
				if format == "json" {
					return formatJSON
				}
				if format == "yaml" {
					return formatYAML
				}
			}
		}
	}
	return formatDefault
}

func formatOutput(results []contextResult, format outputFormat, subcommand string) error {
	switch format {
	case formatJSON:
		return formatJSONOutput(results, subcommand)
	case formatYAML:
		return formatYAMLOutput(results, subcommand)
	default:
		if subcommand == "version" {
			return formatVersionOutput(results)
		}
		return formatDefaultOutput(results)
	}
}

func formatDefaultOutput(results []contextResult) error {
	// First pass: collect all contexts and their outputs to determine max context width
	type outputData struct {
		context string
		lines   []string
		err     error
		errMsg  string
	}
	var allOutputs []outputData
	maxContextWidth := len("CONTEXT")

	for _, result := range results {
		if result.err != nil {
			if len(result.context) > maxContextWidth {
				maxContextWidth = len(result.context)
			}
			allOutputs = append(allOutputs, outputData{
				context: result.context,
				err:     result.err,
				errMsg:  result.output,
			})
			continue
		}

		output := strings.TrimSpace(result.output)
		if output == "" {
			continue
		}

		lines := strings.Split(output, "\n")
		if len(lines) == 0 {
			continue
		}

		if len(result.context) > maxContextWidth {
			maxContextWidth = len(result.context)
		}

		allOutputs = append(allOutputs, outputData{
			context: result.context,
			lines:   lines,
		})
	}

	// Find the header from the first valid output
	var headerLine string
	var headerFound bool
	for _, data := range allOutputs {
		if data.err == nil && len(data.lines) > 1 {
			headerLine = data.lines[0]
			headerFound = true
			break
		}
	}

	// Print header if found
	if headerFound {
		contextPadding := strings.Repeat(" ", maxContextWidth-len("CONTEXT"))
		fmt.Printf("%s%s  %s\n", "CONTEXT", contextPadding, headerLine)
	}

	// Print all outputs
	for _, data := range allOutputs {
		if data.err != nil {
			fmt.Fprintf(os.Stderr, "Context %s: Error: %v\n", data.context, data.err)
			if data.errMsg != "" {
				fmt.Fprintf(os.Stderr, "Output: %s\n", data.errMsg)
			}
			continue
		}

		startIdx := 0
		if headerFound && len(data.lines) > 1 {
			startIdx = 1 // Skip header line
		}

		contextPadding := strings.Repeat(" ", maxContextWidth-len(data.context))

		for i := startIdx; i < len(data.lines); i++ {
			line := strings.TrimSpace(data.lines[i])
			if line == "" {
				continue
			}
			fmt.Printf("%s%s  %s\n", data.context, contextPadding, line)
		}
	}

	return nil
}

func formatVersionOutput(results []contextResult) error {
	type versionInfo struct {
		clientVersion    string
		kustomizeVersion string
		serverVersion    string
	}

	// Parse version information from results
	versionData := make(map[string]versionInfo)
	var clientVersion, kustomizeVersion string

	// First pass: extract client and kustomize version from first successful result
	for _, result := range results {
		if result.err != nil {
			continue
		}

		output := strings.TrimSpace(result.output)
		if output == "" {
			continue
		}

		// Extract client and kustomize version (same for all contexts)
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if clientVersion == "" && strings.HasPrefix(line, "Client Version:") {
				clientVersion = strings.TrimPrefix(line, "Client Version:")
				clientVersion = strings.TrimSpace(clientVersion)
			}
			if kustomizeVersion == "" && strings.HasPrefix(line, "Kustomize Version:") {
				kustomizeVersion = strings.TrimPrefix(line, "Kustomize Version:")
				kustomizeVersion = strings.TrimSpace(kustomizeVersion)
			}
		}
		// Continue looking if we haven't found both yet
		if clientVersion != "" && kustomizeVersion != "" {
			break
		}
	}

	// Second pass: extract server version for each context
	for _, result := range results {
		if result.err != nil {
			versionData[result.context] = versionInfo{
				serverVersion: "ERROR",
			}
			fmt.Fprintf(os.Stderr, "Context %s: Error: %v\n", result.context, result.err)
			if result.output != "" {
				fmt.Fprintf(os.Stderr, "Output: %s\n", result.output)
			}
			continue
		}

		output := strings.TrimSpace(result.output)
		if output == "" {
			versionData[result.context] = versionInfo{
				serverVersion: "N/A",
			}
			continue
		}

		// Extract server version
		var serverVersion string
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Server Version:") {
				serverVersion = strings.TrimPrefix(line, "Server Version:")
				serverVersion = strings.TrimSpace(serverVersion)
				break
			}
		}

		if serverVersion == "" {
			serverVersion = "N/A"
		}

		versionData[result.context] = versionInfo{
			serverVersion: serverVersion,
		}
	}

	// Print client and kustomize version at the top
	if clientVersion != "" {
		fmt.Printf("Client Version: %s\n", clientVersion)
	}
	if kustomizeVersion != "" {
		fmt.Printf("Kustomize Version: %s\n", kustomizeVersion)
	}
	if clientVersion != "" || kustomizeVersion != "" {
		fmt.Println()
	}

	// Print table header
	fmt.Printf("%-30s  %s\n", "CONTEXT", "SERVER VERSION")
	fmt.Println(strings.Repeat("-", 50))

	// Print table rows
	for _, result := range results {
		info := versionData[result.context]
		fmt.Printf("%-30s  %s\n", result.context, info.serverVersion)
	}

	return nil
}

func formatJSONOutput(results []contextResult, subcommand string) error {
	var allItems []map[string]interface{}

	for _, result := range results {
		if result.err != nil {
			fmt.Fprintf(os.Stderr, "Context %s: Error: %v\n", result.context, result.err)
			if result.output != "" {
				// Try to parse error output anyway
				var errorData map[string]interface{}
				if err := json.Unmarshal([]byte(result.output), &errorData); err == nil {
					errorData["context"] = result.context
					errorData["error"] = result.err.Error()
					allItems = append(allItems, errorData)
				}
			}
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(result.output), &data); err != nil {
			fmt.Fprintf(os.Stderr, "Context %s: Failed to parse JSON: %v\n", result.context, err)
			continue
		}

		// Extract items array if it exists
		if itemsArray, exists := data["items"]; exists {
			items, ok := itemsArray.([]interface{})
			if !ok {
				// Try to convert if it's not the right type
				if itemsSlice, ok := itemsArray.([]interface{}); ok {
					items = itemsSlice
				} else {
					continue
				}
			}

			// Add context metadata to each item
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if metadata, ok := itemMap["metadata"].(map[string]interface{}); ok {
						metadata["context"] = result.context
					} else {
						itemMap["metadata"] = map[string]interface{}{
							"context": result.context,
						}
					}
					allItems = append(allItems, itemMap)
				}
			}
		} else {
			// No items array - this might be a single object or non-list response
			// Add context to the root object
			if metadata, ok := data["metadata"].(map[string]interface{}); ok {
				metadata["context"] = result.context
			} else {
				data["context"] = result.context
			}
			allItems = append(allItems, data)
		}
	}

	output := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "List",
		"items":      allItems,
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

func formatYAMLOutput(results []contextResult, subcommand string) error {
	var allItems []map[string]interface{}

	for _, result := range results {
		if result.err != nil {
			fmt.Fprintf(os.Stderr, "Context %s: Error: %v\n", result.context, result.err)
			if result.output != "" {
				// Try to parse error output anyway
				var errorData map[string]interface{}
				if err := yaml.Unmarshal([]byte(result.output), &errorData); err == nil {
					errorData["context"] = result.context
					errorData["error"] = result.err.Error()
					allItems = append(allItems, errorData)
				}
			}
			continue
		}

		var data map[string]interface{}
		if err := yaml.Unmarshal([]byte(result.output), &data); err != nil {
			fmt.Fprintf(os.Stderr, "Context %s: Failed to parse YAML: %v\n", result.context, err)
			continue
		}

		// Extract items array if it exists
		if itemsArray, exists := data["items"]; exists {
			items, ok := itemsArray.([]interface{})
			if !ok {
				// Try to convert if it's not the right type
				if itemsSlice, ok := itemsArray.([]interface{}); ok {
					items = itemsSlice
				} else {
					continue
				}
			}

			// Add context metadata to each item
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if metadata, ok := itemMap["metadata"].(map[string]interface{}); ok {
						metadata["context"] = result.context
					} else {
						itemMap["metadata"] = map[string]interface{}{
							"context": result.context,
						}
					}
					allItems = append(allItems, itemMap)
				}
			}
		} else {
			// No items array - this might be a single object or non-list response
			// Add context to the root object
			if metadata, ok := data["metadata"].(map[string]interface{}); ok {
				metadata["context"] = result.context
			} else {
				data["context"] = result.context
			}
			allItems = append(allItems, data)
		}
	}

	output := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "List",
		"items":      allItems,
	}

	yamlData, err := yaml.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	fmt.Print(string(yamlData))
	return nil
}
