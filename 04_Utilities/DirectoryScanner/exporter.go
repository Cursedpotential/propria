package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

// Exporter handles exporting scan results to various formats
type Exporter struct {
	includeAnalysis bool
	analyses        []FileAnalysis
	duplicates      []DuplicateGroup
}

// NewExporter creates a new exporter
func NewExporter() *Exporter {
	return &Exporter{}
}

// ExportOptions contains options for exporting
type ExportOptions struct {
	OutputPath      string
	IncludeHidden   bool
	MaxDepth        int
	Format          string
	IncludeAnalysis bool
	Analyses        []FileAnalysis
	Duplicates      []DuplicateGroup
}

// FileRecord represents a file record for export
type FileRecord struct {
	Path          string    `json:"path"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Size          int64     `json:"size"`
	FormattedSize string    `json:"formattedSize"`
	Modified      time.Time `json:"modified"`
	Created       time.Time `json:"created"`
	IsCloud       bool      `json:"isCloud"`
	CloudType     string    `json:"cloudType"`
	Extension     string    `json:"extension"`
	Depth         int       `json:"depth"`
	IsHidden      bool      `json:"isHidden"`
	Hash          string    `json:"hash,omitempty"`
	MimeType      string    `json:"mimeType,omitempty"`
}

// Export exports the file tree to the specified format
func (e *Exporter) Export(root *FileNode, options ExportOptions) error {
	switch strings.ToLower(options.Format) {
	case "json":
		return e.exportJSON(root, options)
	case "csv":
		return e.exportCSV(root, options)
	case "excel", "xlsx":
		return e.exportExcel(root, options)
	case "html":
		return e.exportHTML(root, options)
	case "pdf":
		return e.exportPDF(root, options)
	case "markdown", "md":
		return e.exportMarkdown(root, options)
	case "xml":
		return e.exportXML(root, options)
	case "tree":
		return e.exportTree(root, options)
	default:
		return fmt.Errorf("unsupported format: %s", options.Format)
	}
}

// ExportWithAnalysis exports with deep analysis data
func (e *Exporter) ExportWithAnalysis(root *FileNode, options ExportOptions) error {
	e.includeAnalysis = options.IncludeAnalysis
	e.analyses = options.Analyses
	e.duplicates = options.Duplicates

	return e.Export(root, options)
}

// exportJSON exports to JSON format
func (e *Exporter) exportJSON(root *FileNode, options ExportOptions) error {
	file, err := os.Create(options.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	records := e.collectRecords(root, options)

	exportData := map[string]interface{}{
		"metadata": map[string]interface{}{
			"exportDate":    time.Now(),
			"rootPath":      root.Path,
			"totalFiles":    e.countFiles(root),
			"totalDirs":     e.countDirs(root),
			"totalSize":     e.getTotalSize(root),
			"includeHidden": options.IncludeHidden,
			"maxDepth":      options.MaxDepth,
		},
		"files": records,
	}

	if e.includeAnalysis {
		exportData["analyses"] = e.analyses
		exportData["duplicates"] = e.duplicates
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(exportData)
}

// exportCSV exports to CSV format
func (e *Exporter) exportCSV(root *FileNode, options ExportOptions) error {
	file, err := os.Create(options.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	headers := []string{
		"Path", "Name", "Type", "Size", "FormattedSize",
		"Modified", "Created", "Extension", "Depth",
		"IsCloud", "CloudType", "IsHidden",
	}

	if e.includeAnalysis {
		headers = append(headers, "Hash", "MimeType", "LineCount", "IsDuplicate")
	}

	if err := writer.Write(headers); err != nil {
		return err
	}

	// Collect and write records
	records := e.collectRecords(root, options)

	for _, record := range records {
		row := []string{
			record.Path,
			record.Name,
			record.Type,
			fmt.Sprintf("%d", record.Size),
			record.FormattedSize,
			record.Modified.Format("2006-01-02 15:04:05"),
			record.Created.Format("2006-01-02 15:04:05"),
			record.Extension,
			fmt.Sprintf("%d", record.Depth),
			fmt.Sprintf("%t", record.IsCloud),
			record.CloudType,
			fmt.Sprintf("%t", record.IsHidden),
		}

		if e.includeAnalysis {
			isDup := "false"
			for _, dup := range e.duplicates {
				for _, f := range dup.Files {
					if f.Path == record.Path {
						isDup = "true"
						break
					}
				}
			}
			row = append(row, record.Hash, record.MimeType, "0", isDup)
		}

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// exportExcel exports to Excel format
func (e *Exporter) exportExcel(root *FileNode, options ExportOptions) error {
	f := excelize.NewFile()

	// Create main sheet
	sheet := "Files"
	index, _ := f.NewSheet(sheet)

	// Set headers
	headers := []string{
		"Path", "Name", "Type", "Size", "Formatted Size",
		"Modified", "Created", "Extension", "Depth",
		"Is Cloud", "Cloud Type", "Is Hidden",
	}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	// Style headers
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s1", string(rune('A'+len(headers)-1))), style)

	// Add data
	records := e.collectRecords(root, options)
	for i, record := range records {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), record.Path)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), record.Name)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), record.Type)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), record.Size)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), record.FormattedSize)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), record.Modified)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), record.Created)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), record.Extension)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), record.Depth)
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), record.IsCloud)
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), record.CloudType)
		f.SetCellValue(sheet, fmt.Sprintf("L%d", row), record.IsHidden)
	}

	// Add summary sheet
	summarySheet := "Summary"
	f.NewSheet(summarySheet)

	summaryData := [][]interface{}{
		{"Metric", "Value"},
		{"Total Files", e.countFiles(root)},
		{"Total Directories", e.countDirs(root)},
		{"Total Size", humanize.Bytes(uint64(e.getTotalSize(root)))},
		{"Export Date", time.Now().Format("2006-01-02 15:04:05")},
		{"Root Path", root.Path},
	}

	for i, row := range summaryData {
		for j, val := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+1)
			f.SetCellValue(summarySheet, cell, val)
		}
	}

	// Add duplicates sheet if analysis is included
	if e.includeAnalysis && len(e.duplicates) > 0 {
		dupSheet := "Duplicates"
		f.NewSheet(dupSheet)

		dupHeaders := []string{"Hash", "Size", "Count", "Wasted Space", "Files"}
		for i, header := range dupHeaders {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(dupSheet, cell, header)
		}

		row := 2
		for _, dup := range e.duplicates {
			files := make([]string, len(dup.Files))
			for i, f := range dup.Files {
				files[i] = f.Path
			}

			f.SetCellValue(dupSheet, fmt.Sprintf("A%d", row), dup.Hash)
			f.SetCellValue(dupSheet, fmt.Sprintf("B%d", row), humanize.Bytes(uint64(dup.Size)))
			f.SetCellValue(dupSheet, fmt.Sprintf("C%d", row), len(dup.Files))
			f.SetCellValue(dupSheet, fmt.Sprintf("D%d", row), humanize.Bytes(uint64(dup.TotalWasted)))
			f.SetCellValue(dupSheet, fmt.Sprintf("E%d", row), strings.Join(files, "\n"))
			row++
		}
	}

	f.SetActiveSheet(index)

	// Auto-fit columns
	for i := 0; i < len(headers); i++ {
		col := string(rune('A' + i))
		f.SetColWidth(sheet, col, col, 15)
	}

	return f.SaveAs(options.OutputPath)
}

// exportHTML exports to HTML format
func (e *Exporter) exportHTML(root *FileNode, options ExportOptions) error {
	file, err := os.Create(options.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	tmpl := `<!DOCTYPE html>
<html>
<head>
	<title>Directory Scan Report</title>
	<style>
		body { font-family: Arial, sans-serif; margin: 20px; }
		h1 { color: #333; }
		.summary { background: #f0f0f0; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
		table { border-collapse: collapse; width: 100%; }
		th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
		th { background-color: #4CAF50; color: white; }
		tr:nth-child(even) { background-color: #f2f2f2; }
		.cloud { background-color: #e3f2fd; }
		.hidden { opacity: 0.6; }
		.duplicate { background-color: #fff3e0; }
	</style>
</head>
<body>
	<h1>Directory Scan Report</h1>
	<div class="summary">
		<h2>Summary</h2>
		<p><strong>Root Path:</strong> {{.RootPath}}</p>
		<p><strong>Total Files:</strong> {{.TotalFiles}}</p>
		<p><strong>Total Directories:</strong> {{.TotalDirs}}</p>
		<p><strong>Total Size:</strong> {{.TotalSize}}</p>
		<p><strong>Export Date:</strong> {{.ExportDate}}</p>
		{{if .DuplicateCount}}<p><strong>Duplicate Groups:</strong> {{.DuplicateCount}}</p>{{end}}
		{{if .WastedSpace}}<p><strong>Wasted Space:</strong> {{.WastedSpace}}</p>{{end}}
	</div>
	
	<h2>Files</h2>
	<table>
		<tr>
			<th>Path</th>
			<th>Name</th>
			<th>Type</th>
			<th>Size</th>
			<th>Modified</th>
			<th>Cloud</th>
		</tr>
		{{range .Records}}
		<tr class="{{if .IsCloud}}cloud{{end}} {{if .IsHidden}}hidden{{end}}">
			<td>{{.Path}}</td>
			<td>{{.Name}}</td>
			<td>{{.Type}}</td>
			<td>{{.FormattedSize}}</td>
			<td>{{.Modified.Format "2006-01-02 15:04:05"}}</td>
			<td>{{if .IsCloud}}{{.CloudType}}{{end}}</td>
		</tr>
		{{end}}
	</table>
</body>
</html>`

	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return err
	}

	data := struct {
		RootPath       string
		TotalFiles     int
		TotalDirs      int
		TotalSize      string
		ExportDate     string
		Records        []FileRecord
		DuplicateCount int
		WastedSpace    string
	}{
		RootPath:   root.Path,
		TotalFiles: e.countFiles(root),
		TotalDirs:  e.countDirs(root),
		TotalSize:  humanize.Bytes(uint64(e.getTotalSize(root))),
		ExportDate: time.Now().Format("2006-01-02 15:04:05"),
		Records:    e.collectRecords(root, options),
	}

	if e.includeAnalysis && len(e.duplicates) > 0 {
		data.DuplicateCount = len(e.duplicates)
		var wastedSpace int64
		for _, dup := range e.duplicates {
			wastedSpace += dup.TotalWasted
		}
		data.WastedSpace = humanize.Bytes(uint64(wastedSpace))
	}

	return t.Execute(file, data)
}

// exportPDF exports to PDF format
func (e *Exporter) exportPDF(root *FileNode, options ExportOptions) error {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)

	// Title
	pdf.Cell(40, 10, "Directory Scan Report")
	pdf.Ln(12)

	// Summary
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(40, 8, "Summary")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(40, 6, fmt.Sprintf("Root Path: %s", root.Path))
	pdf.Ln(6)
	pdf.Cell(40, 6, fmt.Sprintf("Total Files: %d", e.countFiles(root)))
	pdf.Ln(6)
	pdf.Cell(40, 6, fmt.Sprintf("Total Directories: %d", e.countDirs(root)))
	pdf.Ln(6)
	pdf.Cell(40, 6, fmt.Sprintf("Total Size: %s", humanize.Bytes(uint64(e.getTotalSize(root)))))
	pdf.Ln(6)
	pdf.Cell(40, 6, fmt.Sprintf("Export Date: %s", time.Now().Format("2006-01-02 15:04:05")))
	pdf.Ln(10)

	// File listing header
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(100, 7, "Path")
	pdf.Cell(40, 7, "Name")
	pdf.Cell(20, 7, "Type")
	pdf.Cell(30, 7, "Size")
	pdf.Cell(40, 7, "Modified")
	pdf.Cell(30, 7, "Cloud")
	pdf.Ln(7)

	// File data
	pdf.SetFont("Arial", "", 9)
	records := e.collectRecords(root, options)

	// Limit records for PDF (too many will make huge file)
	maxRecords := 500
	if len(records) > maxRecords {
		records = records[:maxRecords]
		pdf.Cell(40, 6, fmt.Sprintf("Note: Showing first %d of %d total files", maxRecords, len(records)))
		pdf.Ln(6)
	}

	for _, record := range records {
		// Truncate long paths
		path := record.Path
		if len(path) > 60 {
			path = "..." + path[len(path)-57:]
		}

		pdf.Cell(100, 5, path)
		pdf.Cell(40, 5, record.Name)
		pdf.Cell(20, 5, record.Type)
		pdf.Cell(30, 5, record.FormattedSize)
		pdf.Cell(40, 5, record.Modified.Format("2006-01-02"))
		if record.IsCloud {
			pdf.Cell(30, 5, record.CloudType)
		} else {
			pdf.Cell(30, 5, "")
		}
		pdf.Ln(5)

		// Check for page break
		if pdf.GetY() > 180 {
			pdf.AddPage()
			// Repeat header
			pdf.SetFont("Arial", "B", 10)
			pdf.Cell(100, 7, "Path")
			pdf.Cell(40, 7, "Name")
			pdf.Cell(20, 7, "Type")
			pdf.Cell(30, 7, "Size")
			pdf.Cell(40, 7, "Modified")
			pdf.Cell(30, 7, "Cloud")
			pdf.Ln(7)
			pdf.SetFont("Arial", "", 9)
		}
	}

	return pdf.OutputFileAndClose(options.OutputPath)
}

// exportMarkdown exports to Markdown format
func (e *Exporter) exportMarkdown(root *FileNode, options ExportOptions) error {
	file, err := os.Create(options.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "# Directory Scan Report\n\n")
	fmt.Fprintf(file, "## Summary\n\n")
	fmt.Fprintf(file, "- **Root Path:** %s\n", root.Path)
	fmt.Fprintf(file, "- **Total Files:** %d\n", e.countFiles(root))
	fmt.Fprintf(file, "- **Total Directories:** %d\n", e.countDirs(root))
	fmt.Fprintf(file, "- **Total Size:** %s\n", humanize.Bytes(uint64(e.getTotalSize(root))))
	fmt.Fprintf(file, "- **Export Date:** %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	if e.includeAnalysis && len(e.duplicates) > 0 {
		fmt.Fprintf(file, "## Duplicate Files\n\n")
		fmt.Fprintf(file, "Found %d groups of duplicate files:\n\n", len(e.duplicates))

		for i, dup := range e.duplicates {
			fmt.Fprintf(file, "### Duplicate Group %d\n", i+1)
			fmt.Fprintf(file, "- **Size:** %s\n", humanize.Bytes(uint64(dup.Size)))
			fmt.Fprintf(file, "- **Wasted Space:** %s\n", humanize.Bytes(uint64(dup.TotalWasted)))
			fmt.Fprintf(file, "- **Files:**\n")
			for _, f := range dup.Files {
				fmt.Fprintf(file, "  - %s\n", strings.ReplaceAll(strings.ReplaceAll(f.Path, "\n", ""), "\r", ""))
			}
			fmt.Fprintf(file, "\n")
		}
	}

	// Write file tree
	fmt.Fprintf(file, "## File Tree\n\n")
	fmt.Fprintf(file, "```\n")
	e.writeTreeNode(file, root, "", options, 0)
	fmt.Fprintf(file, "```\n\n")

	// Write detailed file list
	fmt.Fprintf(file, "## File Details\n\n")
	fmt.Fprintf(file, "| Path | Size | Modified | Cloud |\n")
	fmt.Fprintf(file, "|------|------|----------|-------|\n")

	records := e.collectRecords(root, options)
	for _, record := range records {
		cloudInfo := ""
		if record.IsCloud {
			cloudInfo = record.CloudType
		}
		fmt.Fprintf(file, "| %s | %s | %s | %s |\n",
			record.Path,
			record.FormattedSize,
			record.Modified.Format("2006-01-02"),
			cloudInfo,
		)
	}

	return nil
}

// exportXML exports to XML format
func (e *Exporter) exportXML(root *FileNode, options ExportOptions) error {
	file, err := os.Create(options.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	fmt.Fprintf(file, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	fmt.Fprintf(file, "<DirectoryScan>\n")
	fmt.Fprintf(file, "  <Metadata>\n")
	fmt.Fprintf(file, "    <ExportDate>%s</ExportDate>\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(file, "    <RootPath>%s</RootPath>\n", root.Path)
	fmt.Fprintf(file, "    <TotalFiles>%d</TotalFiles>\n", e.countFiles(root))
	fmt.Fprintf(file, "    <TotalDirectories>%d</TotalDirectories>\n", e.countDirs(root))
	fmt.Fprintf(file, "    <TotalSize>%d</TotalSize>\n", e.getTotalSize(root))
	fmt.Fprintf(file, "  </Metadata>\n")
	fmt.Fprintf(file, "  <Files>\n")

	e.writeXMLNode(file, root, options, 0, 2)

	fmt.Fprintf(file, "  </Files>\n")
	fmt.Fprintf(file, "</DirectoryScan>\n")

	return nil
}

// exportTree exports as a text tree structure
func (e *Exporter) exportTree(root *FileNode, options ExportOptions) error {
	file, err := os.Create(options.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	fmt.Fprintf(file, "Directory Tree: %s\n", root.Path)
	fmt.Fprintf(file, "Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "================================================================================\n\n")

	e.writeTreeNode(file, root, "", options, 0)

	fmt.Fprintf(file, "\n================================================================================\n")
	fmt.Fprintf(file, "Summary:\n")
	fmt.Fprintf(file, "  Total Files: %d\n", e.countFiles(root))
	fmt.Fprintf(file, "  Total Directories: %d\n", e.countDirs(root))
	fmt.Fprintf(file, "  Total Size: %s\n", humanize.Bytes(uint64(e.getTotalSize(root))))

	return nil
}

// Helper methods

func (e *Exporter) collectRecords(node *FileNode, options ExportOptions) []FileRecord {
	var records []FileRecord
	e.walkNode(node, &records, options, 0)
	return records
}

func (e *Exporter) walkNode(node *FileNode, records *[]FileRecord, options ExportOptions, depth int) {
	if options.MaxDepth > 0 && depth > options.MaxDepth {
		return
	}

	if !options.IncludeHidden && node.IsHidden {
		return
	}

	record := FileRecord{
		Path:          node.Path,
		Name:          node.Name,
		Type:          e.getNodeType(node),
		Size:          node.Size,
		FormattedSize: humanize.Bytes(uint64(node.Size)),
		Modified:      node.Modified,
		Created:       node.Created,
		IsCloud:       node.IsCloud,
		CloudType:     node.CloudType,
		Extension:     node.Extension,
		Depth:         depth,
		IsHidden:      node.IsHidden,
		Hash:          node.Hash,
		MimeType:      node.MimeType,
	}

	*records = append(*records, record)

	if node.IsDir && node.ChildrenLoaded {
		for _, child := range node.Children {
			e.walkNode(child, records, options, depth+1)
		}
	}
}

func (e *Exporter) writeTreeNode(file *os.File, node *FileNode, prefix string, options ExportOptions, depth int) {
	if options.MaxDepth > 0 && depth > options.MaxDepth {
		return
	}

	if !options.IncludeHidden && node.IsHidden {
		return
	}

	// Determine the symbol
	symbol := "├── "
	if depth == 0 {
		symbol = ""
	}

	// Write the node
	nodeInfo := node.Name
	if node.IsDir {
		nodeInfo += "/"
	} else {
		nodeInfo += fmt.Sprintf(" (%s)", humanize.Bytes(uint64(node.Size)))
	}

	if node.IsCloud {
		nodeInfo += fmt.Sprintf(" [%s]", node.CloudType)
	}

	fmt.Fprintf(file, "%s%s%s\n", prefix, symbol, nodeInfo)

	// Process children
	if node.IsDir && node.ChildrenLoaded {
		// Sort children
		children := make([]*FileNode, len(node.Children))
		copy(children, node.Children)
		sort.Slice(children, func(i, j int) bool {
			if children[i].IsDir != children[j].IsDir {
				return children[i].IsDir
			}
			return children[i].Name < children[j].Name
		})

		for i, child := range children {
			newPrefix := prefix
			if depth > 0 {
				if i == len(children)-1 {
					newPrefix += "    "
				} else {
					newPrefix += "│   "
				}
			}
			e.writeTreeNode(file, child, newPrefix, options, depth+1)
		}
	}
}

func (e *Exporter) writeXMLNode(file *os.File, node *FileNode, options ExportOptions, depth int, indent int) {
	if options.MaxDepth > 0 && depth > options.MaxDepth {
		return
	}

	if !options.IncludeHidden && node.IsHidden {
		return
	}

	indentStr := strings.Repeat("  ", indent)

	if node.IsDir {
		fmt.Fprintf(file, "%s<Directory>\n", indentStr)
	} else {
		fmt.Fprintf(file, "%s<File>\n", indentStr)
	}

	fmt.Fprintf(file, "%s  <Name>%s</Name>\n", indentStr, node.Name)
	fmt.Fprintf(file, "%s  <Path>%s</Path>\n", indentStr, node.Path)
	fmt.Fprintf(file, "%s  <Size>%d</Size>\n", indentStr, node.Size)
	fmt.Fprintf(file, "%s  <Modified>%s</Modified>\n", indentStr, node.Modified.Format(time.RFC3339))

	if node.IsCloud {
		fmt.Fprintf(file, "%s  <CloudProvider>%s</CloudProvider>\n", indentStr, node.CloudType)
	}

	if node.IsDir && node.ChildrenLoaded {
		for _, child := range node.Children {
			e.writeXMLNode(file, child, options, depth+1, indent+1)
		}
	}

	if node.IsDir {
		fmt.Fprintf(file, "%s</Directory>\n", indentStr)
	} else {
		fmt.Fprintf(file, "%s</File>\n", indentStr)
	}
}

func (e *Exporter) getNodeType(node *FileNode) string {
	if node.IsDir {
		return "Directory"
	}
	return "File"
}

func (e *Exporter) countFiles(node *FileNode) int {
	count := 0
	if !node.IsDir {
		count = 1
	}

	if node.ChildrenLoaded {
		for _, child := range node.Children {
			count += e.countFiles(child)
		}
	}

	return count
}

func (e *Exporter) countDirs(node *FileNode) int {
	count := 0
	if node.IsDir {
		count = 1
	}

	if node.ChildrenLoaded {
		for _, child := range node.Children {
			count += e.countDirs(child)
		}
	}

	return count
}

func (e *Exporter) getTotalSize(node *FileNode) int64 {
	size := node.Size

	if node.ChildrenLoaded {
		for _, child := range node.Children {
			size += e.getTotalSize(child)
		}
	}

	return size
}
