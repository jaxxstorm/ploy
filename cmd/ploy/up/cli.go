package up

import (
	"context"
	"fmt"
	"os"
	"sort"

	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	n "github.com/jaxxstorm/ploy/pkg/name"
	pulumi "github.com/jaxxstorm/ploy/pkg/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

const columnWidth = 50

var (
	subtle  = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	special = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	list = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), false, true, false, false).
		BorderForeground(subtle).
		MarginRight(2).
		Height(8).
		Width(columnWidth + 1)

	listHeader = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderBottom(true).
			BorderForeground(subtle).
			MarginRight(2).
			Render

	listItem = lipgloss.NewStyle().PaddingLeft(2).Render

	checkMark = lipgloss.NewStyle().SetString("✓").
			Foreground(special).
			PaddingRight(1).
			String()

	listDone = func(s string) string {
		return checkMark + lipgloss.NewStyle().
			Strikethrough(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#969B86", Dark: "#696969"}).
			Render(s)
	}

	docStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)

	dryrun    bool
	name      string
	directory string
	verbose   bool
	nlb       bool
)

type programOptions struct {
	ctx        context.Context
	org        string
	region     string
	name       string
	dockerfile string
	directory  string
}

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "up",
		Short: "Deploy your application",
		Long:  "Deploy your application to Kubernetes",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			// Set some required params
			ctx := context.Background()
			org := viper.GetString("org")
			region := viper.GetString("region")

			if org == "" {
				return fmt.Errorf("must specify pulumi org via flag or config file")
			}

			// If the user doesn't specify a name, generate a random one for them
			if len(args) < 1 {
				name = n.GenerateName()
			} else {
				name = args[0]
			}

			// check if we have a valid Dockerfile before proceeding
			dockerfile := fmt.Sprintf("%s/Dockerfile", directory)
			if _, err := os.Stat(dockerfile); os.IsNotExist(err) {
				return fmt.Errorf("no Dockerfile found in %s: %v", directory, err)
			}

			s := spinner.NewModel()
			s.Spinner = spinner.Dot
			s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

			p := tea.NewProgram(model{
				options: programOptions{
					ctx:        ctx,
					org:        org,
					region:     region,
					name:       name,
					dockerfile: dockerfile,
					directory:  directory,
				},
				logChannel:        make(chan statusMessage),
				eventChannel:      make(chan events.EngineEvent),
				spinner:           s,
				updatesInProgress: map[string]string{},
				updatesComplete:   map[string]string{},
			})

			if p.Start() != nil {
				return fmt.Errorf("unable to start program")
			}

			return nil

		},
	}

	f := command.Flags()
	f.BoolVarP(&dryrun, "preview", "p", false, "Preview changes, dry-run mode")
	f.BoolVarP(&verbose, "verbose", "v", false, "Show output of Pulumi operations")
	f.StringVarP(&directory, "dir", "d", ".", "Path to docker context to use")
	f.BoolVar(&nlb, "nlb", false, "Provision an NLB instead of ELB")

	return command
}

// watchForLogMessages forwards any log messages to the `Update` method
func watchForLogMessages(msg chan statusMessage) tea.Cmd {
	return func() tea.Msg {
		return <-msg
	}
}

// watchForEvents forwards any engine events to the `Update` method
func watchForEvents(event chan events.EngineEvent) tea.Cmd {
	return func() tea.Msg {
		return <-event
	}
}

type statusMessage struct {
	msg      string
	complete bool
}

type errMsg struct {
	err error
}

// model is the struct that holds the state for this program
type model struct {
	options           programOptions
	eventChannel      chan events.EngineEvent // where we'll receive engine events
	logChannel        chan statusMessage      // where we'll receive log messages
	spinner           spinner.Model
	quitting          bool
	currentMessage    string
	updatesInProgress map[string]string // resources with updates in progress
	updatesComplete   map[string]string // resources with updates completed
	err               error
}

// Init runs any IO needed at the initialization of the program
func (m model) Init() tea.Cmd {
	return tea.Batch(
		watchForLogMessages(m.logChannel),
		runPulumiUpdate(m.options, m.logChannel, m.eventChannel),
		watchForEvents(m.eventChannel),
		spinner.Tick,
	)
}

// Update acts on any events and updates state (model) accordingly
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case events.EngineEvent:
		if msg.ResourcePreEvent != nil {
			m.updatesInProgress[msg.ResourcePreEvent.Metadata.URN] = msg.ResourcePreEvent.Metadata.Type
		}
		if msg.ResOutputsEvent != nil {
			urn := msg.ResOutputsEvent.Metadata.URN
			m.updatesComplete[urn] = msg.ResOutputsEvent.Metadata.Type
			delete(m.updatesInProgress, urn)
		}
		return m, watchForEvents(m.eventChannel) // wait for next event
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		m.quitting = true
		return m, tea.Quit
	case statusMessage:
		if msg.complete {
			// m.currentMessage = "Succeeded!"
			return m, tea.Quit
		}
		m.currentMessage = msg.msg
		return m, watchForLogMessages(m.logChannel)
	case errMsg:
		fmt.Printf("Received an error: %v", msg.err)
		return m, tea.Quit
	default:
		return m, nil
	}
}

// View displays the state in the terminal
func (m model) View() string {
	var inProgVals []string
	var completedVals []string
	doc := strings.Builder{}

	doc.WriteString(fmt.Sprintf("Creating deployment: %s", m.options.name))

	if len(m.updatesInProgress) > 0 || len(m.updatesComplete) > 0 {
		for _, v := range m.updatesInProgress {
			inProgVals = append(inProgVals, listItem(v))
		}
		sort.Strings(inProgVals)
		for _, v := range m.updatesComplete {
			completedVals = append(completedVals, listDone(v))
		}
		sort.Strings(completedVals)

		inProgVals = append([]string{listHeader("Updates in progress")}, inProgVals...)
		completedVals = append([]string{listHeader("Updates completed")}, completedVals...)
		lists := lipgloss.JoinHorizontal(lipgloss.Top,
			list.Render(
				lipgloss.JoinVertical(lipgloss.Left,
					inProgVals...,
				),
			),
			list.Copy().Width(columnWidth).Render(
				lipgloss.JoinVertical(lipgloss.Left,
					completedVals...,
				),
			),
		)
		doc.WriteString("\n")
		doc.WriteString(lists)
	}

	physicalWidth, _, _ := term.GetSize(int(os.Stdout.Fd()))
	if physicalWidth > 0 {
		docStyle = docStyle.MaxWidth(physicalWidth)
	}

	s := fmt.Sprintf("\n%sStatus: %s%s\n", m.spinner.View(), m.currentMessage, docStyle.Render(doc.String()))
	if m.quitting {
		s += "\n"
	}
	if m.err != nil {
		return m.err.Error()
	}
	return s
}

func runPulumiUpdate(options programOptions, logChannel chan<- statusMessage, eventChannel chan<- events.EngineEvent) tea.Cmd {
	return func() tea.Msg {

		stackName := auto.FullyQualifiedStackName(options.org, "ploy", options.name)
		pulumiStack, err := auto.UpsertStackInlineSource(options.ctx, stackName, "ploy", nil)

		if err != nil {
			return errMsg{fmt.Errorf("error creating stack: %v", err)}
		}

		logChannel <- statusMessage{msg: fmt.Sprintf("Created/Selected stack: %v\n", stackName)}

		logChannel <- statusMessage{msg: fmt.Sprintf("Setting Region to %v\n", options.region)}
		err = pulumiStack.SetConfig(options.ctx, "aws:region", auto.ConfigValue{Value: options.region})
		if err != nil {
			return errMsg{fmt.Errorf("error setting aws region: %v", err)}
		}

		// Set up the workspace and install all the required plugins the user needs
		workspace := pulumiStack.Workspace()

		err = pulumi.EnsurePlugins(workspace)
		if err != nil {
			return errMsg{fmt.Errorf("error installing plugin: %v", err)}
		}

		// Now, we set the pulumi program that is going to run
		workspace.SetProgram(pulumi.Deploy(options.name, options.directory, false))

		logChannel <- statusMessage{msg: "Running update..."}
		res, err := pulumiStack.Up(options.ctx, optup.EventStreams(eventChannel))
		if err != nil {
			return errMsg{fmt.Errorf("error running pulumi update: %v", err)}
		}

		url, ok := res.Outputs["address"].Value.(string)

		if !ok {
			return errMsg{fmt.Errorf("unable to retrieve address")}
		}

		logChannel <- statusMessage{msg: "Update succeeded!", complete: false}
		logChannel <- statusMessage{msg: fmt.Sprintf("Your deployment is available at: %s", url)}

		return statusMessage{complete: true}
	}
}
