// Copyright © 2018 Lars Chr. Duus Hausmann <jazz-gcalsync@zqz.dk>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"google.golang.org/api/calendar/v3"
)

// Functions related to org-mode generated. Heavily inspired by:
// https://github.com/codemac/gcalorg

func printOrgDate(start, end *calendar.EventDateTime) string {
	final := ""
	if start == nil { // this event has dates! hurrah!
		return "\n"
	}

	if start.Date != "" { // all day event!
		ts, _ := time.Parse("2006-01-02", start.Date)
		tsf := ts.Format("2006-01-02")
		final = final + fmt.Sprintf("<%s", tsf)
		if end == nil {
			return final + ">"
		}

		te, _ := time.Parse("2006-01-02", end.Date)
		te = te.AddDate(0, 0, -1)
		// The end date is "exclusive", so we should subtract a day, and
		// if the day is equivalent to start, then we should just print
		// start.
		if te.Equal(ts) {
			return final + ">"
		}
		tef := te.Format("2006-01-02")
		return final + fmt.Sprintf(">--<%s>", tef)
	}

	ts, _ := time.Parse(time.RFC3339, start.DateTime)
	ts = ts.In(time.Local)
	tsf := ts.Format("2006-01-02 Mon 15:04")
	final = final + fmt.Sprintf("<%s", tsf)

	if end == nil {
		return final + fmt.Sprintf(">")
	}

	te, _ := time.Parse(time.RFC3339, end.DateTime)
	te = te.In(time.Local)
	if te.Day() != ts.Day() {
		tef := te.Format("2006-01-02 Mon 15:04")
		return final + fmt.Sprintf(">--<%s>", tef)
	}
	tef := te.Format("15:04")
	return final + fmt.Sprintf("-%s>", tef)
}

// cleanString removes special characters for org-mode, as almost no one will be
// using org-mode formatting.
func cleanString(s string) string {
	s = strings.Replace(s, "[", "{", -1)
	s = strings.Replace(s, "]", "}", -1)
	s = strings.Replace(s, "\n*", "\n,*", -1)
	return s
}

func printOrg(e *calendar.Event, tagname string, out *strings.Builder) {
	var fullentry string
	print_entry := true
	fullentry += fmt.Sprintf("** ")
	if e.Status == "tenative" || e.Status == "cancelled" {
		fullentry += fmt.Sprintf("(%s) ", e.Status)
	}
	summary := e.Summary
	if summary == "" {
		summary = "busy"
	}
	// if tagname != "" {
	//	filllen := 90 - (len(summary) + len(tagname) + 3)
	//	if filllen > 0 {
	//		summary += strings.Repeat(" ", filllen)
	//	}
	//	summary += " :" + tagname + ":"
	// }
	fullentry += fmt.Sprintf("%s\n", summary)
	fullentry += fmt.Sprintf(":PROPERTIES:\n")
	fullentry += fmt.Sprintf(":ID:       %s\n", e.ICalUID)
	fullentry += fmt.Sprintf(":GCALLINK: %s\n", e.HtmlLink)
	if e.Creator != nil {
		fullentry += fmt.Sprintf(":CREATOR: [[mailto:%s][%s]]\n", e.Creator.Email, cleanString(e.Creator.DisplayName))
	}
	if e.Organizer != nil {
		fullentry += fmt.Sprintf(":ORGANIZER: [[mailto:%s][%s]]\n", e.Organizer.Email, cleanString(e.Organizer.DisplayName))
	}
	fullentry += fmt.Sprintf(":END:\n\n")
	fullentry += fmt.Sprintf("%s\n", printOrgDate(e.Start, e.End))
	attendees := e.Attendees
	canonical_id := func(ea *calendar.EventAttendee) string {
		if ea.Id != "" {
			return ea.Id
		} else if ea.Email != "" {
			return ea.Email
		} else if ea.DisplayName != "" {
			return cleanString(ea.DisplayName)
		}
		return "sadness"
	}

	sort.SliceStable(attendees, func(i, j int) bool {
		return canonical_id(attendees[i]) < canonical_id(attendees[j])
	})
	if len(attendees) > 0 {
		fullentry += fmt.Sprintf("Attendees:\n")
	}
	if len(attendees) > 20 {
		fullentry += fmt.Sprintf("... Many\n")
	} else {
		for _, a := range attendees {
			if a != nil {

				// ResponseStatus: The attendee's response status. Possible values are:
				//
				// - "needsAction" - The attendee has not responded to the invitation.
				//
				// - "declined" - The attendee has declined the invitation.
				// - "tentative" - The attendee has tentatively accepted the invitation.
				//
				// - "accepted" - The attendee has accepted the invitation.
				//  ResponseStatus string `json:"responseStatus,omitempty"`
				statuschar := " "
				switch a.ResponseStatus {
				case "":
				case "NeedsAction":
				case "declined":
					statuschar = "✗"
				case "tenative":
					statuschar = "☐"
				case "accepted":
					statuschar = "✓"
				}

				linkname := cleanString(a.DisplayName)
				if linkname == "" {
					linkname = a.Email
				}
				fullentry += fmt.Sprintf(" %s [[mailto:%s][%s]]\n", statuschar, a.Email, linkname)

				// If the entire thing is actually declined, why
				// the fuck does google show it to me? this is
				// the most bullshit aspect of this calendar API
				// afaict. I really hope I've found the wrong
				// way of doing this.
				if a.Self && a.ResponseStatus == "declined" {
					print_entry = false
				}

			}
		}
	}

	to_p := fmt.Sprintf("\n%s\n", e.Description)
	fullentry += cleanString(to_p)
	fullentry += "\n"
	attachment_title := "\nAttachments:\n"
	attachment_entries := ""
	for _, a := range e.Attachments {
		if a == nil {
			continue
		}

		attachment_entries += fmt.Sprintf("- [[%s][%s]]\n", a.FileUrl,
			cleanString(a.Title))
	}
	if len(attachment_entries) > 0 {
		fullentry += attachment_title + attachment_entries
	}
	if print_entry {
		out.WriteString(fullentry)
	}
}

func printCalendars(client *http.Client, cal Calendar) *strings.Builder {

	calendarMap := map[string]string{}

	approvedCals := []string{}
	for _, v := range cal.calendars {
		approvedCals = append(approvedCals, v.Calendar)
		calendarMap[v.Calendar] = v.Tag
	}

	srv, err := calendar.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve calendar Client %v", err)
	}

	// find all calendars
	calendars, err := srv.CalendarList.List().ShowHidden(false).ShowDeleted(false).
		MaxResults(250).Do()
	if err != nil {
		log.Fatalf("Unable to list calendars! %v", err)
	}

	curtime := time.Now().UTC().Add(24 * time.Hour).Truncate(24 * time.Hour)
	timeMin := curtime.AddDate(0, -9, 0).Format("2006-01-02T15:04:05Z")
	timeMax := curtime.AddDate(1, 0, 0).Format("2006-01-02T15:04:05Z")

	receivedCals := make(map[string]*calendar.CalendarListEntry, 0)
	for _, c := range calendars.Items {
		receivedCals[c.Id] = c
	}

	var strBuilder strings.Builder

	strBuilder.WriteString(fmt.Sprintf("# -*- eval: (auto-revert-mode 1); -*-\n"))
	strBuilder.WriteString(fmt.Sprintf("#+category: cal\n"))
	for _, approvedCal := range approvedCals {

		c, ok := receivedCals[approvedCal]
		if !ok {
			continue
		}

		strBuilder.WriteString(fmt.Sprintf("* %s :CALENDAR:%s:\n", c.Summary, calendarMap[approvedCal]))
		strBuilder.WriteString(fmt.Sprintf(":PROPERTIES:\n"))
		strBuilder.WriteString(fmt.Sprintf(":ID:         %s\n", c.Id))
		strBuilder.WriteString(fmt.Sprintf(":END:\n"))
		strBuilder.WriteString(fmt.Sprintf("\n  %s\n\n", c.Description))

		npt := ""
		notdone := true

		event_list := make([]*calendar.Event, 0, 250)
		for notdone {
			eventsReq := srv.Events.List(c.Id).ShowDeleted(false).
				SingleEvents(true).TimeMin(timeMin).TimeMax(timeMax).MaxResults(250)
			if npt != "" {
				eventsReq = eventsReq.PageToken(npt)
				npt = ""
			}

			events, err := eventsReq.Do()
			if err != nil {
				log.Fatalf("Unable to retrieve next ten of the user's events. %v", err)
			}

			notdone = events.NextPageToken != ""
			if notdone {
				npt = events.NextPageToken
			}

			event_list = append(event_list, events.Items...)
		}

		// sort events by Id
		sort.SliceStable(event_list, func(i, j int) bool { return event_list[i].Id < event_list[j].Id })

	itemloop:
		for _, i := range event_list {
			// If the DateTime is an empty string the Event is an
			// all-day Event.  So only Date is available.

			// skip things that are chatty (repeating calendar
			// events -> org-mode has been difficult, manually
			// manage those for now. There is probably a way of
			// getting them, but converting the ical format to the
			// org format would be a significant piece of logic)
			for _, v := range cal.titlefilters {
				if strings.Contains(i.Summary, v) {
					continue itemloop
				}
			}
			printOrg(i, calendarMap[approvedCal], &strBuilder)
		}
	}

	// Return
	return &strBuilder
}
