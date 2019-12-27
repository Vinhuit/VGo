package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"log"
	"os"
	// "github.com/urfave/cli/v2/altsrc"
	//"github.com/urfave/cli/v2/altsrc"
	"github.com/tealeg/xlsx"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var lport, rport int
var mode, lname, lnum, stunnel string
var ipplan []string
var config configuration

type configuration struct {
	Kind       string      `yaml:"kind"`
	APIVersion string      `yaml:"apiVersion"`
	Metadata   interface{} `yaml:"metadata"`
	Spec       struct {
		Lab []struct {
			Name     string `yaml:"name"`
			User     string `yaml:"user"`
			Password string `yaml:"password"`
		} `yaml:"Lab"`
		Tunnel []struct {
			User     string `yaml:"user"`
			Hostname string `yaml:"hostname"`
		} `yaml:"tunnel"`
	} `yaml:"spec"`
}

func (c *configuration) ReadConfig() *configuration {
	yamlFile, err := ioutil.ReadFile("test.yml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c

}



func callSSH(ip string, user string, args []string, node string, config configuration) error {
	var cmd *exec.Cmd
	mode = strings.ToLower(mode)
	dest := fmt.Sprintf("%s@%s", user, ip)
	//fmt.Printf("%v", lport)

	//fmt.Printf("%v", tunnel)
	tmpfile, err := ioutil.TempFile("", "ssh")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())

	if err := os.Chmod(tmpfile.Name(), 0600); err != nil {
		return err
	}

	if err := tmpfile.Close(); err != nil {
		return err
	}
	if node == "client" {
		fmt.Println("Conneting to " + stunnel)
		cmd = exec.Command("ssh", append([]string{"-i", tmpfile.Name(), dest}, args...)...)

	} else if node == "tunnel" {
		config.ReadConfig()
		stunnel, _ := strconv.Atoi(stunnel)

		host := config.Spec.Tunnel[stunnel-1].User + "@" + config.Spec.Tunnel[stunnel-1].Hostname
		//stringargs := strings.Join(args, " ")
		fmt.Println("Conneting to " + dest + " over tunnel " + host)
		cmd = exec.Command("ssh", append([]string{"-i", tmpfile.Name(), "-t", host, " ssh ", dest}, args...)...)
	} else {
		tunnel := fmt.Sprintf("%d:localhost:%d", lport, rport)
		if mode == "local" {
			fmt.Println("create Local Port Forwarding " + tunnel)
			cmd = exec.Command("ssh", append([]string{"-L", tunnel, dest}, args...)...)

		} else if mode == "remote" {
			fmt.Println("Create Remote Port Forwarding " + tunnel)
			cmd = exec.Command("ssh", append([]string{"-R", tunnel, dest}, args...)...)
		} else {
			fmt.Println("Create Dynamic Port " + strconv.Itoa(lport))
			cmd = exec.Command("ssh", append([]string{"-D", strconv.Itoa(lport), dest}, args...)...)
		}

	}
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func SshCommand(ctx *cli.Context) error {
	args := ctx.Args()
	var user string = ""

	//fmt.Println(ctx)
	if args.Len() == 0 {
		return cli.ShowCommandHelp(ctx, ctx.Command.Name)
	} else if args.Len() > 0 && (args.Get(0) == "-h" || args.Get(0) == "--help") {
		return cli.ShowCommandHelp(ctx, ctx.Command.Name)
	}

	nodeName := ctx.Args().First()

	if strings.Contains(nodeName, "@") {
		user = strings.Split(nodeName, "@")[0]
		nodeName = strings.Split(nodeName, "@")[1]

	}
	// fmt.Printf("%v, %v", user, nodeName)
	//addrs, err := net.ResolveIPAddr("ip", nodeName)
	// fmt.Println("%v ", addrs)

	addrs := strings.Trim(nodeName, "\t")

	stringargs := args.Slice()[1:]

	//fmt.Println("aaaa", len(user))

	if ctx.Command.Name == "connect" {
		m := "client"
		if ctx.IsSet("tunnel") {
			// stringargs = append(stringargs, stunnel)
			// fmt.Println(stringargs)
			m = "tunnel"
		}
		if user == "" {
			resource := GetResource(lname, lnum)
			addrs, _ := strconv.Atoi(addrs)
			ip := resource[addrs][5] + "." + resource[addrs][6]

			config.ReadConfig()
			for i := 0; i < len(config.Spec.Lab); i++ {
				lab := config.Spec.Lab[i].Name
				if strings.ToLower(lab) == strings.ToLower(lname) {
					user = config.Spec.Lab[i].User
					password := config.Spec.Lab[i].Password
					fmt.Println("User: ", ip, " | Password: ", password)
				}
			}
			fmt.Println(ip)

			return callSSH(ip, user, stringargs, m, config)
		} else {
			return callSSH(addrs, user, stringargs, m, config)
		}

	} else {
		return callSSH(addrs, user, stringargs, "server", config)
	}

}

func ScpDowload(ctx *cli.Context) error {
	return nil
}
func GetResource(patten string, lab string) [][]string {
	excelFileName := "IP.xlsx"
	i := 0
	count := 0
	var allcell []string
	var listfilter [][]string
	xlFile, err := xlsx.OpenFile(excelFileName)
	if err != nil {
		return [][]string{}
	}
	for _, sheet := range xlFile.Sheets {
		//fmt.Println(sheet.Name)
		if sheet.Name == "NET-1" || sheet.Name == "CEE-NET-2" {

			for _, row := range sheet.Rows {
				//fmt.Println(row.Height)

				for _, cell := range row.Cells {
					//r := regexp.MustCompile(patten)
					allcell = append(allcell, cell.String())

					if i > 0 {
						//ipplan = append(ipplan, cell.String())
						i = i + 1
						//fmt.Printf("MATCHX: %v\n", ipplan)
					}
					text := cell.String()
					matches, _ := regexp.MatchString(strings.ToLower(patten), strings.ToLower(text))
					if matches {
						//ipplan = append(ipplan, allcell[j-1])
						//ipplan = append(ipplan, cell.String())
						i = i + 1
						//fmt.Printf("MATCH: %v\n", ipplan)

					}
					if i == 3 {
						//fmt.Printf("MATCH: %v\n", row.Cells)

						var temp []string
						//fmt.Printf("MATCH: %v\n", ipplan)
						for _, filter := range row.Cells {
							temp = append(temp, filter.String())
						}
						for _, filter := range temp {
							matches, _ := regexp.MatchString(strings.ToLower(lab), strings.ToLower(filter))
							if matches {

								//ipplan = append(ipplan, allcell[j-1])
								listfilter = append(listfilter, temp)
								fmt.Printf("LAB-%v: %v\n", count, temp)
								count++
							}
						}

						//fmt.Printf("ALLCELL: %v\n", listfilter)
						i = 0
						allcell = []string{""}
						ipplan = []string{""}

					}

					//ipplan = append([]string{cell.String()})
					//fmt.Printf("%s\n", text)

				}

			}
		}
	}

	return listfilter
}
func main() {

	// flags := []cli.Flag{
	// 	altsrc.NewIntFlag(&cli.IntFlag{Name: "test"}),
	// 	&cli.StringFlag{Name: "load"},
	// }

	app := &cli.App{

		Commands: []*cli.Command{
			{
				Name:   "connect",
				Usage:  "SSH into a node format user@hostname ",
				Action: SshCommand,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "labname",
						Aliases:     []string{"l"},
						Usage:       "Name of Lab  ",
						Value:       "CC",
						Destination: &lname,
					},
					&cli.StringFlag{
						Name:        "labnum",
						Aliases:     []string{"n"},
						Usage:       "number of lab ",
						Value:       "208",
						Destination: &lnum,
					},
					&cli.StringFlag{
						Name:        "tunnel",
						Aliases:     []string{"t"},
						Usage:       "connect over server tunnel",
						Value:       "vgo@127.0.0.01",
						Destination: &stunnel,
					},
				},
			},
			{
				Name:   "server",
				Usage:  "server listen",
				Action: SshCommand,
				Subcommands: []*cli.Command{
					{
						Name:  "list",
						Usage: "client connect to node",
						Action: func(c *cli.Context) error {
							fmt.Println(GetResource(lname, lnum))
							return nil
						},
					},
				},
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:        "remoteport",
						Aliases:     []string{"rport"},
						Usage:       "remote `PORT` to foward",
						Value:       8080,
						Destination: &rport,
					},
					&cli.IntFlag{
						Name:        "localport",
						Aliases:     []string{"lport"},
						Usage:       "local `PORT` to listen",
						Value:       8080,
						Destination: &lport,
					},
					&cli.StringFlag{
						Name:        "mode, m",
						Aliases:     []string{"m"},
						Usage:       "Local,Remote or `Dynamic` SSH Port Forwarding",
						Value:       "Dynamic",
						Destination: &mode,
					},
				},
			},
			{

				Name:  "list",
				Usage: "client connect to node",
				Action: func(c *cli.Context) error {
					fmt.Println(GetResource(lname, lnum))
					return nil
				},
				Subcommands: []*cli.Command{
					{
						Name:  "host",
						Usage: "List Host ",
						Action: func(c *cli.Context) error {
							fmt.Println(GetResource(lname, lnum))
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:        "labname",
								Aliases:     []string{"l"},
								Usage:       "Name of Lab  ",
								Value:       "atlas",
								Destination: &lname,
							},
							&cli.StringFlag{
								Name:        "labnum",
								Aliases:     []string{"n"},
								Usage:       "number of lab ",
								Value:       "208",
								Destination: &lnum,
							},
						},
					},
					{
						Name:  "tunnel",
						Usage: "List tunnel",
						Action: func(c *cli.Context) error {
							fmt.Println(config.ReadConfig().Spec.Tunnel)
							return nil
						},
					},
					{
						Name:  "user",
						Usage: "List tunnel",
						Action: func(c *cli.Context) error {
							for _, per := range config.ReadConfig().Spec.Lab {
								fmt.Printf("%+v\n", per)
							}
							//fmt.Printf("%+v\n, %T", config.ReadConfig().Spec.Lab, config.ReadConfig().Spec.Lab)
							return nil
						},
					},
				},
			},
			{
				Name:    "download",
				Aliases: []string{"c"},
				Usage:   "complete a task on the list",
				Action: func(c *cli.Context) error {
					fmt.Println("completed task: ", c.Args().First())
					return nil
				},
			},
			{
				Name:    "upload",
				Aliases: []string{"t"},
				Usage:   "options for task templates",
				Subcommands: []*cli.Command{
					{
						Name:  "add",
						Usage: "add a new template",
						Action: func(c *cli.Context) error {
							fmt.Println("new task template: ", c.Args().First())
							return nil
						},
					},
					{
						Name:  "remove",
						Usage: "remove an existing template",
						Action: func(c *cli.Context) error {
							fmt.Println("removed task template: ", c.Args().First())
							return nil
						},
					},
				},
			},
		},

		// Action: func(c *cli.Context) error {
		// 	fmt.Println("yaml ist rad")
		// 	return nil
		// },
		// Before: altsrc.InitInputSourceWithContext(flags, altsrc.NewYamlSourceFromFlagFunc("load")),
		// Flags:  flags,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
