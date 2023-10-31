package main

import (
	"fmt"
	"github.com/datasparq-ai/houston/api"
	"github.com/datasparq-ai/houston/client"
	"github.com/spf13/cobra"
	"strings"
)

func main() {

	if err := func() (rootCmd *cobra.Command) {

		rootCmd = &cobra.Command{
			Use:   "houston",
			Short: "HOUSTON Orchestration API · https://callhouston.io",
			Args:  cobra.ArbitraryArgs,
			Run: func(c *cobra.Command, args []string) {
				s := "\u001B[1;38;2;58;145;172m"
				e := "\u001B[0m"
				fmt.Println("\n🚀 \u001B[1mHOUSTON\u001B[0m · Orchestration API · https://callhouston.io\nBasic usage:")
				fmt.Printf("  %[1]vhouston api%[2]v                    \u001B[37m# starts a local API server%[2]v\n", s, e)
				fmt.Printf("  %[1]vhouston save%[2]v \u001B[1m--plan plan.yaml%[2]v  \u001B[37m# saves a new plan%[2]v\n", s, e)
				fmt.Printf("  %[1]vhouston start%[2]v \u001B[1m--plan my-plan%[2]v   \u001B[37m# creates and triggers a new mission%[2]v\n", s, e)
				fmt.Printf("  %[1]vhouston help%[2]v                   \u001B[37m# shows help for all commands%[2]v\n", s, e)
				return
			},
		}

		rootCmd.AddCommand(func() (createCmd *cobra.Command) {
			createCmd = &cobra.Command{
				Use:   "version",
				Short: "Print the version number",
				Run: func(c *cobra.Command, args []string) {
					fmt.Println("v0.6.0")
				},
			}
			return
		}())

		rootCmd.AddCommand(func() (createCmd *cobra.Command) {
			createCmd = &cobra.Command{
				Use:   "api",
				Short: "Run the Houston API server",
				Run: func(c *cobra.Command, args []string) {
					configPath, _ := createCmd.Flags().GetString("config")
					a := api.New(configPath)
					go a.Monitor()
					a.Run()
				},
			}
			createCmd.Flags().String("config", "", "path to a config file")
			return
		}())

		rootCmd.AddCommand(func() (createCmd *cobra.Command) {
			var id = ""
			var name = ""
			var password = ""
			createCmd = &cobra.Command{
				Use:   "create-key",
				Short: "Create a new API key (requires admin password)",
				Run: func(c *cobra.Command, args []string) {
					err := client.CreateKey(id, name, password)
					if err != nil {
						client.HandleCommandLineError(err)
					}
				},
			}
			createCmd.Flags().StringVarP(&id, "id", "i", "", "New API key value")
			createCmd.Flags().StringVarP(&name, "name", "n", "", "Description for this key")
			createCmd.Flags().StringVarP(&password, "password", "p", "", "API admin password")

			return
		}())

		rootCmd.AddCommand(func() (createCmd *cobra.Command) {
			var password = ""
			createCmd = &cobra.Command{
				Use:   "keys",
				Short: "List all API keys (requires admin password)",
				Run: func(c *cobra.Command, args []string) {

					err := client.ListKeys(password)
					if err != nil {
						client.HandleCommandLineError(err)
					}
				},
			}
			createCmd.Flags().StringVarP(&password, "password", "p", "", "API admin password")

			return
		}())

		rootCmd.AddCommand(func() (createCmd *cobra.Command) {
			var plan string
			createCmd = &cobra.Command{
				Use:   "save",
				Short: "Save a plan",
				Run: func(c *cobra.Command, args []string) {
					err := client.Save(plan)
					if err != nil {
						client.HandleCommandLineError(err)
					}
				},
			}
			createCmd.Flags().StringVarP(&plan, "plan", "p", "", "File path or URL of the plan to save.")
			createCmd.MarkFlagRequired("plan")
			return
		}())

		rootCmd.AddCommand(func() (createCmd *cobra.Command) {
			var plan string
			var missionId = ""
			var stages = ""
			var exclude = ""
			var skip = ""
			createCmd = &cobra.Command{
				Use:   "start",
				Short: "Create a new mission and trigger the first stage(s)",
				Run: func(c *cobra.Command, args []string) {
					err := client.Start(plan, missionId,
						strings.Split(strings.Replace(stages, " ", "", -1), ","),
						strings.Split(strings.Replace(exclude, " ", "", -1), ","),
						strings.Split(strings.Replace(skip, " ", "", -1), ","))
					if err != nil {
						client.HandleCommandLineError(err)
					}
				},
			}
			createCmd.Flags().StringVarP(&plan, "plan", "p", "", "Name or file path of the plan to create a new mission with")
			createCmd.MarkFlagRequired("plan")
			createCmd.Flags().StringVarP(&missionId, "mission-id", "m", "", "Mission ID to assign to the new mission")
			createCmd.Flags().StringVarP(&stages, "stages", "s", "", "Comma separated list of stage names to be used as the starting point for the mission. \nIf not provided, all stages with no upstream stages will be triggered")
			createCmd.Flags().StringVarP(&exclude, "exclude", "i", "", "Comma separated list of stage names to be excluded in the new mission")
			createCmd.Flags().StringVarP(&skip, "skip", "k", "", "Comma separated list of stage names to be skipped in the new mission")
			return
		}())

		rootCmd.AddCommand(func() (createCmd *cobra.Command) {
			createCmd = &cobra.Command{
				Use:   "demo",
				Short: "Run the API in demo mode",
				Run: func(c *cobra.Command, args []string) {
					demo(createCmd)
				},
			}
			createCmd.Flags().String("config", "", "path to a config file")
			return
		}())

		return
	}().Execute(); err != nil {
		panic(err)
	}
}
