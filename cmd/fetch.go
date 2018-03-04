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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type CalendarMap struct {
	Tag      string
	Calendar string
}

type Calendar struct {
	name         string
	tokenfile    string
	tagname      string
	calendars    []CalendarMap
	titlefilters []string
	orgfile      string
}

func loadCalendarFromConfig(calendar string) (Calendar, error) {
	c := Calendar{}
	c.name = calendar
	v := viper.Sub(calendar)
	// Check correct length of configuration

	if len(v.AllKeys()) < 4 {
		return c, fmt.Errorf("Incorrect configuration - expected 5 items, got: %d", len(v.AllKeys()))
	}

	if tokenFile := v.GetString("tokenfile"); len(tokenFile) > 0 {
		if tokenFile[:2] == "~/" {
			usr, _ := user.Current()
			dir := usr.HomeDir
			c.tokenfile = filepath.Join(dir, tokenFile[2:])
		} else {
			c.tokenfile = tokenFile
		}

	} else {
		return c, errors.New("tokenfile not specificied")
	}

	c.orgfile = v.GetString("orgfile")

	if len(c.orgfile) > 0 && c.orgfile[:2] == "~/" {
		usr, _ := user.Current()
		dir := usr.HomeDir
		c.orgfile = filepath.Join(dir, c.orgfile[2:])
	}

	// Tag be optional
	if tagname := v.GetString("tagname"); len(tagname) > 0 {
		c.tagname = tagname
	}

	c.calendars = []CalendarMap{}
	for k, v := range v.GetStringMap("calendars") {
		c.calendars = append(c.calendars, CalendarMap{Tag: strings.ToUpper(k), Calendar: v.(string)})
	}

	c.titlefilters = v.GetStringSlice("titlefilters")

	// validate configuration
	if len(c.calendars) < 1 {
		return c, errors.New("No calendar is specified")
	}

	return c, nil
}

// fetchCmd represents the fetch command
var fetchCmd = &cobra.Command{
	Use:   "fetch [calendar]",
	Short: "Fetches gcal and creates org-mode output",
	Long:  `Fetches google calendar and creates org-mode file, which can be added to Org agenda for meetings.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("requires at least one arg")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		calendar := args[0]

		cal, err := loadCalendarFromConfig(calendar)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot find calendar stanza for calendar: %s in configuration - error %v", calendar, err)
			os.Exit(1)
		}
		cl := genClient(cal.tokenfile)
		strBuilder := printCalendars(cl, cal)
		if len(cal.orgfile) == 0 || cal.orgfile == "-" {
			fmt.Println(strBuilder.String())
		} else {
			// Write to org file
			fmt.Fprintf(os.Stderr, "Writing agenda to %s\n", cal.orgfile)
			if err := ioutil.WriteFile(cal.orgfile, []byte(strBuilder.String()), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to %s - %v", cal.orgfile, err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(fetchCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// fetchCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// fetchCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
