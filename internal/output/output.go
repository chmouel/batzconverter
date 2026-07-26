package output

import (
	"encoding/json"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
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

func Terminal(writer io.Writer, rows []Row, interactive, color, emoji bool) error {
	if len(rows) == 0 {
		return nil
	}
	showMarker := false
	if emoji {
		for _, row := range rows {
			if row.IsBase || row.IsLocal {
				showMarker = true
				break
			}
		}
	}
	data := make([][]string, 0, len(rows))
	for _, row := range rows {
		cells := []string{row.Name, row.Value}
		if emoji {
			marker := ""
			if row.IsBase {
				marker = "✈"
			} else if row.IsLocal {
				marker = "🏠"
			}
			cells = append(cells, strings.TrimSpace(row.Emoji))
			if showMarker {
				cells = append(cells, marker)
			}
		}
		data = append(data, cells)
	}

	renderer := table.New().
		Rows(data...).
		Wrap(false).
		BorderRow(false).
		BorderHeader(false)
	if interactive {
		renderer.
			Border(lipgloss.RoundedBorder()).
			BorderTop(true).
			BorderBottom(true).
			BorderLeft(true).
			BorderRight(true).
			BorderColumn(true)
	} else {
		renderer.
			BorderTop(false).
			BorderBottom(false).
			BorderLeft(false).
			BorderRight(false).
			BorderColumn(false)
	}
	renderer.StyleFunc(func(_, column int) lipgloss.Style {
		style := lipgloss.NewStyle()
		if interactive {
			style = style.Padding(0, 1)
		} else if column < len(data[0])-1 {
			style = style.PaddingRight(2)
		}
		if color && column == 0 {
			style = style.Bold(true)
		}
		return style
	})
	if color {
		renderer.BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240")))
	}
	rendered := renderer.String()
	if !interactive {
		lines := strings.Split(rendered, "\n")
		for i := range lines {
			lines[i] = strings.TrimRight(lines[i], " \t")
		}
		rendered = strings.Join(lines, "\n")
	}
	_, err := io.WriteString(writer, strings.TrimSuffix(rendered, "\n")+"\n")
	return err
}
