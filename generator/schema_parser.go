package generator

import (
	"fmt"
	"strings"
)

// SchemaBlock represents a block in the Prisma schema
type SchemaBlock struct {
	Type     string // "generator", "datasource", "model", "enum"
	Name     string
	Content  string
	StartPos int
	EndPos   int
}

// SchemaParser handles parsing and filtering of Prisma schema content
type SchemaParser struct {
	content string
}

// NewSchemaParser creates a new schema parser with the given content
func NewSchemaParser(content string) *SchemaParser {
	return &SchemaParser{
		content: content,
	}
}

// FilterByGenerator filters the schema to only include the specified generator
// along with all datasources, models, enums, and other non-generator blocks
func (p *SchemaParser) FilterByGenerator(generatorName string) (string, error) {
	blocks, err := p.parseBlocks()
	if err != nil {
		return "", fmt.Errorf("failed to parse schema blocks: %w", err)
	}

	var filteredBlocks []SchemaBlock
	var foundGenerator bool

	for _, block := range blocks {
		switch block.Type {
		case "generator":
			// Only include the specified generator
			if block.Name == generatorName {
				filteredBlocks = append(filteredBlocks, block)
				foundGenerator = true
			}
		case "datasource", "model", "enum":
			// Include all datasources, models, and enums
			filteredBlocks = append(filteredBlocks, block)
		default:
			// Include any other non-generator blocks (comments, etc.)
			filteredBlocks = append(filteredBlocks, block)
		}
	}

	if !foundGenerator {
		return "", fmt.Errorf("generator '%s' not found in schema", generatorName)
	}

	return p.reconstructSchema(filteredBlocks), nil
}

// parseBlocks parses the schema content into individual blocks
func (p *SchemaParser) parseBlocks() ([]SchemaBlock, error) {
	var blocks []SchemaBlock
	lines := strings.Split(p.content, "\n")

	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Skip empty lines
		if line == "" {
			i++
			continue
		}

		// Handle comments as separate blocks
		if strings.HasPrefix(line, "//") {
			blocks = append(blocks, SchemaBlock{
				Type:     "comment",
				Content:  lines[i],
				StartPos: i,
				EndPos:   i,
			})
			i++
			continue
		}

		// Check for block declarations
		if block, endLine, found := p.parseBlockAt(lines, i); found {
			blocks = append(blocks, block)
			i = endLine + 1
		} else {
			// Handle standalone lines that aren't part of a block
			blocks = append(blocks, SchemaBlock{
				Type:     "other",
				Content:  lines[i],
				StartPos: i,
				EndPos:   i,
			})
			i++
		}
	}

	return blocks, nil
}

// parseBlockAt attempts to parse a block starting at the given line index
func (p *SchemaParser) parseBlockAt(lines []string, startLine int) (SchemaBlock, int, bool) {
	line := strings.TrimSpace(lines[startLine])

	// Check for block types
	blockTypes := []string{"generator", "datasource", "model", "enum"}
	var blockType, blockName string

	for _, bt := range blockTypes {
		if strings.HasPrefix(line, bt+" ") {
			blockType = bt
			// Extract block name
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				blockName = parts[1]
			}
			break
		}
	}

	if blockType == "" {
		return SchemaBlock{}, 0, false
	}

	// Find the opening brace
	openBracePos := strings.Index(line, "{")
	if openBracePos == -1 {
		// Look for opening brace on subsequent lines
		for j := startLine + 1; j < len(lines); j++ {
			if strings.Contains(lines[j], "{") {
				openBracePos = 0 // Found on a different line
				break
			}
		}
		if openBracePos == -1 {
			return SchemaBlock{}, 0, false
		}
	}

	// Find the matching closing brace
	braceCount := 0
	var contentLines []string
	endLine := startLine

	for i := startLine; i < len(lines); i++ {
		currentLine := lines[i]
		contentLines = append(contentLines, currentLine)

		// Count braces in this line
		for _, char := range currentLine {
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
				if braceCount == 0 {
					endLine = i
					goto blockComplete
				}
			}
		}
	}

blockComplete:
	if braceCount != 0 {
		return SchemaBlock{}, 0, false // Unmatched braces
	}

	return SchemaBlock{
		Type:     blockType,
		Name:     blockName,
		Content:  strings.Join(contentLines, "\n"),
		StartPos: startLine,
		EndPos:   endLine,
	}, endLine, true
}

// reconstructSchema rebuilds the schema from filtered blocks
func (p *SchemaParser) reconstructSchema(blocks []SchemaBlock) string {
	var result []string

	for i, block := range blocks {
		// Add the block content
		result = append(result, block.Content)

		// Add spacing between blocks (except for the last block)
		if i < len(blocks)-1 {
			// Don't add extra spacing if the next block is a comment or empty
			nextBlock := blocks[i+1]
			if nextBlock.Type != "comment" && strings.TrimSpace(nextBlock.Content) != "" {
				result = append(result, "")
			}
		}
	}

	return strings.Join(result, "\n")
}
