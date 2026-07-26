package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Row struct {
	Name    string
	Zone    string
	Value   string
	Emoji   string
	Icon    string
	IsBase  bool
	IsLocal bool
}

type AlfredIcon struct {
	Path string `json:"path"`
}

type AlfredItem struct {
	UID      string      `json:"uid"`
	Title    string      `json:"title"`
	Arg      string      `json:"arg"`
	Subtitle string      `json:"subtitle"`
	Icon     *AlfredIcon `json:"icon,omitempty"`
}

type Alfred struct {
	Items []AlfredItem `json:"items"`
}

func JSON(writer io.Writer, rows []Row) error {
	document := Alfred{Items: make([]AlfredItem, 0, len(rows))}
	for _, row := range rows {
		item := AlfredItem{Title: row.Name, Arg: row.Value, Subtitle: row.Value}
		if row.Icon != "" {
			item.Icon = &AlfredIcon{Path: row.Icon}
		}
		document.Items = append(document.Items, item)
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "    ")
	return encoder.Encode(document)
}

func Terminal(writer io.Writer, rows []Row, padding int, color, emoji bool) error {
	for _, row := range rows {
		name := row.Name
		if color {
			name = "\033[1m" + name + "\033[00m"
		}
		suffix := ""
		if emoji {
			suffix = row.Emoji
		}
		if row.IsBase {
			if color {
				suffix += "✈"
			}
		} else if row.IsLocal && color {
			suffix += "🏠"
		}
		width := padding
		if width < len(row.Value) {
			width = len(row.Value)
		}
		if _, err := fmt.Fprintf(writer, "%-20s %-*s %s\n", name, width, row.Value, suffix); err != nil {
			return err
		}
	}
	return nil
}

func CSV(rows []Row, emoji bool) (string, error) {
	var buffer strings.Builder
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"Timezone", "Date"}); err != nil {
		return "", err
	}
	for _, row := range rows {
		name := row.Name
		if emoji {
			name = strings.TrimSpace(row.Emoji) + " " + name
		}
		if err := writer.Write([]string{name, row.Value}); err != nil {
			return "", err
		}
	}
	writer.Flush()
	return buffer.String(), writer.Error()
}
