package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode"

	"github.com/labtether/labtether-cli/internal/client"
)

// emptyCell marks a value the Hub left out or sent as null.
const emptyCell = "-"

// tableColumn is one column of human output. key is a dot path into a row,
// such as "metadata.entity_id"; format defaults to cellText.
type tableColumn struct {
	header string
	key    string
	format func(any) string
}

func col(header, key string) tableColumn {
	return tableColumn{header: header, key: key}
}

func colFormat(header, key string, format func(any) string) tableColumn {
	return tableColumn{header: header, key: key, format: format}
}

// printList prints one list from the Hub data payload as a table. With --json
// it prints the whole payload unchanged.
func printList(resp *client.V2Response, field string, columns ...tableColumn) error {
	if jsonOutput {
		printJSON(json.RawMessage(resp.Data))
		return nil
	}
	rows, err := decodeTableRows(resp, field)
	if err != nil {
		return err
	}
	printTable(rows, columns...)
	return nil
}

// decodeTableRows reads the named list from the data payload. Numbers keep
// the Hub's exact text so large integers never print as floats.
func decodeTableRows(resp *client.V2Response, field string) ([]map[string]any, error) {
	var data map[string]json.RawMessage
	if err := decodeResponseData(resp, &data); err != nil {
		return nil, err
	}
	raw, found := data[field]
	if !found {
		return nil, fmt.Errorf("decode response data: missing %s", field)
	}
	rows, err := decodeRows(raw)
	if err != nil {
		return nil, fmt.Errorf("decode response data: %s: %w", field, err)
	}
	return rows, nil
}

func decodeRows(raw json.RawMessage) ([]map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var rows []map[string]any
	if err := decoder.Decode(&rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func printTable(rows []map[string]any, columns ...tableColumn) {
	w := tabwriter.NewWriter(rootCmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	cells := make([]string, len(columns))
	for i, column := range columns {
		cells[i] = column.header
	}
	fmt.Fprintln(w, strings.Join(cells, "\t"))
	for _, row := range rows {
		for i, column := range columns {
			format := column.format
			if format == nil {
				format = cellText
			}
			cells[i] = cleanCell(format(lookupPath(row, column.key)))
		}
		fmt.Fprintln(w, strings.Join(cells, "\t"))
	}
	_ = w.Flush()
}

func lookupPath(row map[string]any, path string) any {
	var current any = row
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = object[part]
	}
	return current
}

// cleanCell keeps one value on one table line and drops terminal control
// characters that a remote host could put in a name or command.
func cleanCell(text string) string {
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, text)
	text = strings.TrimSpace(text)
	if text == "" {
		return emptyCell
	}
	return text
}

func cellText(value any) string {
	switch v := value.(type) {
	case nil:
		return emptyCell
	case string:
		return v
	case bool:
		if v {
			return "yes"
		}
		return "no"
	case json.Number:
		return numberText(v)
	case float64:
		return numberText(json.Number(strconv.FormatFloat(v, 'f', -1, 64)))
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if text := cellText(item); text != emptyCell {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		data, err := json.Marshal(v)
		if err != nil {
			return emptyCell
		}
		return string(data)
	default:
		return fmt.Sprint(v)
	}
}

// numberText prints whole numbers as integers and other numbers with at most
// two decimals, never in exponent form.
func numberText(n json.Number) string {
	if _, err := strconv.ParseInt(n.String(), 10, 64); err == nil {
		return n.String()
	}
	if _, err := strconv.ParseUint(n.String(), 10, 64); err == nil {
		return n.String()
	}
	f, err := n.Float64()
	if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
		return n.String()
	}
	if f == math.Trunc(f) {
		return strconv.FormatFloat(f, 'f', 0, 64)
	}
	text := strconv.FormatFloat(f, 'f', 2, 64)
	text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	if text == "-0" {
		return "0"
	}
	return text
}

// shortID shortens a 64-character Docker ID to the 12 characters that
// `docker ps` shows and the Hub accepts for container actions.
func shortID(value any) string {
	text := cellText(value)
	if len(text) > 12 && strings.Trim(strings.ToLower(text), "0123456789abcdef") == "" {
		return text[:12]
	}
	return text
}
