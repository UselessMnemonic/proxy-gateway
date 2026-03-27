package ipc

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

type StatusRequest struct{}

func (StatusRequest) Kind() uint16 {
	return KindStatusRequest
}

type StatusDetails struct {
	State string
	Err   string
}

type StatusResponse struct {
	Targets   map[string]StatusDetails
	Frontends map[string]StatusDetails
}

func (StatusResponse) Kind() uint16 {
	return KindStatusResponse
}

func (r StatusResponse) ConsoleString() string {
	var builder strings.Builder
	writeStatusSection(&builder, "Targets", r.Targets)
	builder.WriteString("\n")
	writeStatusSection(&builder, "Frontends", r.Frontends)
	return builder.String()
}

func writeStatusSection(builder *strings.Builder, title string, items map[string]StatusDetails) {
	builder.WriteString(title)
	builder.WriteString("\n")
	if len(items) == 0 {
		builder.WriteString("  (none)\n")
		return
	}

	names := slices.Sorted(maps.Keys(items))

	nameWidth := len("NAME")
	stateWidth := len("STATE")
	for _, name := range names {
		if len(name) > nameWidth {
			nameWidth = len(name)
		}
		state := items[name].State
		if len(state) > stateWidth {
			stateWidth = len(state)
		}
	}

	fmt.Fprintf(builder, "  %-*s  %-*s  %s\n", nameWidth, "NAME", stateWidth, "STATE", "ERROR")
	fmt.Fprintf(builder, "  %s  %s  %s\n", strings.Repeat("-", nameWidth), strings.Repeat("-", stateWidth), "-----")
	for _, name := range names {
		details := items[name]
		errText := "-"
		if details.Err != "" {
			errText = details.Err
		}
		fmt.Fprintf(builder, "  %-*s  %-*s  %s\n", nameWidth, name, stateWidth, details.State, errText)
	}
}
