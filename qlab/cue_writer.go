package qlab

import (
	"fmt"
	"strings"
)

// WriteCueFile generates a CUE file string from a workspace name and cues
// It ensures all cues conform to the schema defined in lib/cj/qlab_workspace.cue
func WriteCueFile(workspaceName string, cues []Cue, comment string) string {
	var builder strings.Builder
	builder.WriteString("package qlab\n\n")
	builder.WriteString("import \"github.com/zenibako/cuejitsu/lib/cj\"\n\n")

	if comment != "" {
		// Create safe comment
		safeComment := strings.ReplaceAll(comment, "\n", " ")
		safeComment = strings.ReplaceAll(safeComment, "\r", " ")
		builder.WriteString(fmt.Sprintf("// %s\n", safeComment))
	}

	builder.WriteString("cj.#Workspace & {\n")
	builder.WriteString(fmt.Sprintf("\tname: %q\n", workspaceName))
	builder.WriteString("\tcues: [\n")

	for _, c := range cues {
		writeCue(&builder, c, 2)
	}

	builder.WriteString("\t]\n")
	builder.WriteString("}\n")

	return builder.String()
}

// writeCue recursively writes a cue and its children with proper indentation
// It ensures all fields conform to the CUE schema by always including defaults
func writeCue(builder *strings.Builder, c Cue, indent int) {
	indentStr := strings.Repeat("\t", indent)

	builder.WriteString(indentStr + "{\n")

	// Type is required
	fmt.Fprintf(builder, "%s\ttype: %q\n", indentStr, c.Type)

	// Number (optional, defaults to "")
	if c.Number != "" {
		fmt.Fprintf(builder, "%s\tnumber: %q\n", indentStr, c.Number)
	}

	// Name (optional, defaults to "")
	if c.Name != "" {
		fmt.Fprintf(builder, "%s\tname: %q\n", indentStr, c.Name)
	}

	// Mode (optional for group cues)
	if c.Mode > 0 {
		fmt.Fprintf(builder, "%s\tmode: %d\n", indentStr, c.Mode)
	}

	// Notes (optional, defaults to "")
	if c.Notes != "" && c.Notes != c.Name {
		fmt.Fprintf(builder, "%s\tnotes: %q\n", indentStr, c.Notes)
	}

	// Text (optional, defaults to "")
	if c.Text != "" {
		fmt.Fprintf(builder, "%s\ttext: %q\n", indentStr, c.Text)
	}

	// Text formatting colors (optional)
	if len(c.TextColor) == 4 {
		fmt.Fprintf(builder, "%s\t\"text/format/color\": [%g, %g, %g, %g]\n",
			indentStr, c.TextColor[0], c.TextColor[1], c.TextColor[2], c.TextColor[3])
	}
	if len(c.TextBgColor) == 4 {
		fmt.Fprintf(builder, "%s\t\"text/format/backgroundColor\": [%g, %g, %g, %g]\n",
			indentStr, c.TextBgColor[0], c.TextBgColor[1], c.TextBgColor[2], c.TextBgColor[3])
	}
	if c.TextFontSize > 0 {
		fmt.Fprintf(builder, "%s\t\"text/format/fontSize\": %g\n", indentStr, c.TextFontSize)
	}
	if c.TextAlignment != "" {
		fmt.Fprintf(builder, "%s\t\"text/format/alignment\": %q\n", indentStr, c.TextAlignment)
	}

	// Geometry properties (optional)
	if c.StageName != "" {
		fmt.Fprintf(builder, "%s\tstageName: %q\n", indentStr, c.StageName)
	}
	if c.StageID != "" {
		fmt.Fprintf(builder, "%s\tstageID: %q\n", indentStr, c.StageID)
	}
	if len(c.Translation) == 2 {
		fmt.Fprintf(builder, "%s\ttranslation: [%g, %g]\n",
			indentStr, c.Translation[0], c.Translation[1])
	}
	if c.Opacity > 0 && c.Opacity <= 1.0 {
		fmt.Fprintf(builder, "%s\topacity: %g\n", indentStr, c.Opacity)
	}

	// FileTarget (optional, defaults to "")
	if c.FileTarget != "" {
		fmt.Fprintf(builder, "%s\tfileTarget: %q\n", indentStr, c.FileTarget)
	}

	// Duration - ALWAYS write this for ALL cue types to match schema
	// According to the schema, duration defaults to "" (empty string for zero)
	// We write it as a string to match QLab's OSC format
	if c.Duration > 0 {
		fmt.Fprintf(builder, "%s\tduration: %q\n", indentStr, fmt.Sprintf("%g", c.Duration))
	} else {
		// Always include duration, even if zero, to match schema
		fmt.Fprintf(builder, "%s\tduration: \"\"\n", indentStr)
	}

	// PreWait (optional, defaults to "")
	if c.PreWait > 0 {
		fmt.Fprintf(builder, "%s\tpreWait: %q\n", indentStr, fmt.Sprintf("%g", c.PreWait))
	}

	// Armed (optional, defaults to "")
	if c.Armed {
		builder.WriteString(indentStr + "\tarmed: \"true\"\n")
	}

	// ColorName (optional, defaults to "none")
	if c.ColorName != "" && c.ColorName != "none" {
		fmt.Fprintf(builder, "%s\tcolorName: %q\n", indentStr, c.ColorName)
	}

	// CueTargetNumber (optional)
	if c.CueTargetNumber != "" {
		fmt.Fprintf(builder, "%s\tcueTargetNumber: %q\n", indentStr, c.CueTargetNumber)
	}

	// Write nested cues if present
	if len(c.Cues) > 0 {
		builder.WriteString(indentStr + "\tcues: [\n")
		for _, childCue := range c.Cues {
			writeCue(builder, childCue, indent+2)
		}
		builder.WriteString(indentStr + "\t]\n")
	}

	builder.WriteString(indentStr + "},\n")
}

// NormalizeCue ensures a cue has all required fields with proper defaults
// This ensures compatibility with the CUE schema
func NormalizeCue(c *Cue) {
	// Type is required - ensure it's not empty
	if c.Type == "" {
		c.Type = "group" // Default to group if unspecified
	}

	// Ensure TextColor and TextBgColor are valid if set
	if len(c.TextColor) > 0 && len(c.TextColor) != 4 {
		c.TextColor = nil // Clear invalid color
	}
	if len(c.TextBgColor) > 0 && len(c.TextBgColor) != 4 {
		c.TextBgColor = nil // Clear invalid color
	}

	// Recursively normalize nested cues
	for i := range c.Cues {
		NormalizeCue(&c.Cues[i])
	}
}
