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

	"github.com/lchausmann/gcalsync/pkg/config"
	"github.com/spf13/cobra"
)

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

		cal, err := config.LoadCalendarFromConfig(calendar)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot find calendar stanza for calendar: %s in configuration - error %v", calendar, err)
			os.Exit(1)
		}
		cl := genClient(cal.TokenFile)
		strBuilder := printCalendars(cl, cal)
		if len(cal.OrgFile) == 0 || cal.OrgFile == "-" {
			fmt.Println(strBuilder.String())
		} else {
			// Write to org file
			fmt.Fprintf(os.Stderr, "Writing agenda to %s\n", cal.OrgFile)
			if err := ioutil.WriteFile(cal.OrgFile, []byte(strBuilder.String()), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to %s - %v", cal.OrgFile, err)
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
