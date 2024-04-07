package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

func main() {

	var (
		downloadUrl   string
		csvFileName   string
		csvDelimiter  string
		txtFileName   string
		webpageFilter string
	)

	flag.StringVar(&downloadUrl, "url", "https://help.nice-incontact.com/content/studio/actions/timezone/timezone.htm", "CXone Studio Timezone documentation webpage")
	flag.StringVar(&csvFileName, "csv", "cxone_timezones.csv", "Output delimited file of all CXone Studio timezones")
	flag.StringVar(&csvDelimiter, "delimiter", "\t", "Output file delimiter")
	flag.StringVar(&txtFileName, "txt", "cxone_timezones_array.txt", "Output text file array of all CXone Studio timezones")
	flag.StringVar(&webpageFilter, "filter", "DST", "Select appropriate webpage table by filtered keyword")

	flag.Parse()

	body, err := downloadHTML(downloadUrl)
	if err != nil {
		fmt.Println("Error downloading HTML:", err)
		return
	}

	tables, err := extractTables(csvDelimiter, body)
	if err != nil {
		fmt.Println("Error extracting tables:", err)
		return
	}

	filteredTables := filterTablesByKeyword(tables, webpageFilter)

	if err := writeTablesToCSV(filteredTables, csvFileName, csvDelimiter); err != nil {
		fmt.Println("Error writing tables to CSV:", err)
		return
	}

	cx1_sample_array(csvFileName, csvDelimiter, txtFileName, downloadUrl)
}

func downloadHTML(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

func extractTables(delimiter string, body []byte) ([][]string, error) {
	var tables [][]string

	tokenizer := html.NewTokenizer(strings.NewReader(string(body)))

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			err := tokenizer.Err()
			if err == io.EOF {
				break
			}
			return nil, err
		}

		token := tokenizer.Token()
		if tokenType == html.StartTagToken && token.Data == "table" {
			table, err := extractTable(delimiter, tokenizer)
			if err != nil {
				return nil, err
			}
			tables = append(tables, table)
		}
	}

	return tables, nil
}

func extractTable(delimiter string, tokenizer *html.Tokenizer) ([]string, error) {
	var tableRows [][]string

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			err := tokenizer.Err()
			if err == io.EOF {
				break
			}
			return nil, err
		}

		token := tokenizer.Token()

		if tokenType == html.StartTagToken && token.Data == "tr" {
			row, err := extractRow(tokenizer)
			if err != nil {
				return nil, err
			}
			tableRows = append(tableRows, row)
		}

		if tokenType == html.EndTagToken && token.Data == "table" {
			break
		}
	}

	// Convert tableRows to a flat representation (list of strings)
	var tableFlat []string
	for _, row := range tableRows {
		tableFlat = append(tableFlat, strings.Join(row, delimiter))
	}

	return tableFlat, nil
}

func extractRow(tokenizer *html.Tokenizer) ([]string, error) {
	var row []string

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			err := tokenizer.Err()
			if err == io.EOF {
				break
			}
			return nil, err
		}

		token := tokenizer.Token()
		if tokenType == html.TextToken {
			row = append(row, strings.TrimSpace(token.Data))
		}

		if tokenType == html.EndTagToken && token.Data == "tr" {
			break
		}
	}

	return row, nil
}

func filterTablesByKeyword(tables [][]string, keyword string) [][]string {
	var filteredTables [][]string

	for _, table := range tables {
		for _, row := range table {
			if strings.Contains(row, keyword) {
				filteredTables = append(filteredTables, table)
				break
			}
		}
	}

	return filteredTables
}

func writeTablesToCSV(tables [][]string, filename string, delimiter string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = []rune(delimiter)[0]
	defer writer.Flush()

	allCells := extractCellsFromTables(tables, delimiter)
	header := allCells[0]
	header = append(header, "Additional Notes")

	allCells[0] = header

	return writer.WriteAll(allCells)
}

func splitRow(row string, delimiter string) []string {
	doubleDelimiter := delimiter + delimiter
	cells := strings.Split(row, doubleDelimiter)
	// Trim whitespace from each cell
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

func cleanCells(cells []string, delimiter string) []string {
	for i := range cells {
		cells[i] = strings.ReplaceAll(cells[i], delimiter, " ")
		cells[i] = strings.ReplaceAll(cells[i], " ", " ")  // non-breaking space
		cells[i] = strings.ReplaceAll(cells[i], "  ", " ") // double space
		// Remove leading and trailing spaces
		cells[i] = strings.TrimSpace(cells[i])
		// Remove empty cells
		if cells[i] == "" {
			cells = append(cells[:i], cells[i+1:]...)
			i-- // Adjust index after removing cell
		}
	}
	return cells
}

func extractCellsFromTables(tables [][]string, delimiter string) [][]string {
	var allCells [][]string
	for _, table := range tables {
		for _, row := range table {
			cells := splitRow(row, delimiter)
			cells = cleanCells(cells, delimiter)
			allCells = append(allCells, cells)
		}
	}
	return allCells
}

func cx1_sample_array(inputFilename string, delimiter string, outputFilename string, webpage string) {
	file, err := os.Open(inputFilename)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = []rune(delimiter)[0] // Set the delimiter
	reader.FieldsPerRecord = -1         // Allow variable number of fields per record

	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error reading CSV:", err)
		return
	}

	outFile, err := os.Create(outputFilename)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer outFile.Close()

	fmt.Fprintf(outFile, "// %s\n", webpage)

	for index, record := range records {
		timezone := record[0]
		if index == 0 {
			continue
		}
		fmt.Fprintf(outFile, "VALID_TIMEZONES[%d] = \"%s\"\n", index, timezone)
	}
}
