package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func buildBoardCalendarICS(board Board, columns []BoardColumn, cards []Card, cardURL func(cardID string) string) string {
	lines := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//Mattermost Flow//Board Calendar//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		fmt.Sprintf("X-WR-CALNAME:%s", escapeICSField(board.Name)),
	}

	if strings.TrimSpace(board.Description) != "" {
		lines = append(lines, fmt.Sprintf("X-WR-CALDESC:%s", escapeICSField(board.Description)))
	}

	scheduled := make([]Card, 0, len(cards))
	for _, card := range cards {
		if strings.TrimSpace(card.StartDate) != "" || strings.TrimSpace(card.DueDate) != "" {
			scheduled = append(scheduled, card)
		}
	}

	sort.Slice(scheduled, func(i, j int) bool {
		left := calendarSortDate(scheduled[i])
		right := calendarSortDate(scheduled[j])
		if left == right {
			return scheduled[i].UpdatedAt > scheduled[j].UpdatedAt
		}
		return left < right
	})

	for _, card := range scheduled {
		lines = append(lines, buildCalendarEventLines(board, columns, card, cardURL)...)
	}

	lines = append(lines, "END:VCALENDAR", "")
	return strings.Join(lines, "\r\n")
}

func buildCalendarEventLines(board Board, columns []BoardColumn, card Card, cardURL func(cardID string) string) []string {
	startDate, hasStart := parseDay(card.StartDate)
	dueDate, hasDue := parseDay(card.DueDate)

	if hasStart && hasDue {
		if dueDate.Before(startDate) {
			dueDate = startDate
		}
	} else if hasDue {
		startDate = dueDate
		hasStart = true
	} else if hasStart {
		dueDate = startDate
		hasDue = true
	} else {
		return nil
	}

	eventStart := startOfDay(startDate)
	eventEndExclusive := startOfDay(dueDate).Add(24 * time.Hour)
	description := buildCalendarDescription(columns, card, cardURL)

	lines := []string{
		"BEGIN:VEVENT",
		fmt.Sprintf("UID:%s@mattermost-flow", card.ID),
		fmt.Sprintf("DTSTAMP:%s", formatICSTimestamp(card.UpdatedAt)),
		fmt.Sprintf("SUMMARY:%s", escapeICSField(card.Title)),
		fmt.Sprintf("DTSTART;VALUE=DATE:%s", formatICSDate(eventStart)),
		fmt.Sprintf("DTEND;VALUE=DATE:%s", formatICSDate(eventEndExclusive)),
		fmt.Sprintf("DESCRIPTION:%s", escapeICSField(description)),
		fmt.Sprintf("STATUS:%s", calendarStatus(card)),
	}

	if len(card.Labels) > 0 {
		lines = append(lines, fmt.Sprintf("CATEGORIES:%s", escapeICSField(strings.Join(card.Labels, ","))))
	}

	if url := strings.TrimSpace(cardURL(card.ID)); url != "" {
		lines = append(lines, fmt.Sprintf("URL:%s", escapeICSField(url)))
	}

	lines = append(lines, "END:VEVENT")
	return lines
}

func buildCalendarDescription(columns []BoardColumn, card Card, cardURL func(cardID string) string) string {
	parts := []string{
		fmt.Sprintf("Column: %s", columnNameForCalendar(columns, card.ColumnID)),
		fmt.Sprintf("Priority: %s", formatCalendarPriority(card.Priority)),
		fmt.Sprintf("Progress: %d%%", card.Progress),
	}

	if strings.TrimSpace(card.Description) != "" {
		parts = append(parts, "", card.Description)
	}
	if len(card.Labels) > 0 {
		parts = append(parts, "", fmt.Sprintf("Labels: %s", strings.Join(card.Labels, ", ")))
	}
	if url := strings.TrimSpace(cardURL(card.ID)); url != "" {
		parts = append(parts, "", fmt.Sprintf("Open in Mattermost Flow: %s", url))
	}

	return strings.Join(parts, "\n")
}

func columnNameForCalendar(columns []BoardColumn, columnID string) string {
	for _, column := range columns {
		if column.ID == columnID {
			return column.Name
		}
	}
	return "Board"
}

func calendarSortDate(card Card) string {
	if strings.TrimSpace(card.StartDate) != "" {
		return card.StartDate
	}
	if strings.TrimSpace(card.DueDate) != "" {
		return card.DueDate
	}
	return "9999-12-31"
}

func calendarStatus(card Card) string {
	_ = card
	return "CONFIRMED"
}

func formatCalendarPriority(value string) string {
	if value == "" {
		return "Normal"
	}
	return strings.ToUpper(value[:1]) + strings.ToLower(value[1:])
}

func formatICSDate(value time.Time) string {
	return value.UTC().Format("20060102")
}

func formatICSTimestamp(value int64) string {
	if value <= 0 {
		return time.Now().UTC().Format("20060102T150405Z")
	}
	return time.UnixMilli(value).UTC().Format("20060102T150405Z")
}

func escapeICSField(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		";", "\\;",
		",", "\\,",
		"\r\n", "\\n",
		"\n", "\\n",
		"\r", "\\n",
	)
	return replacer.Replace(strings.TrimSpace(value))
}
