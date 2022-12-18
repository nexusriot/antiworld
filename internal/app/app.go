package app

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/muesli/reflow/indent"
	log "github.com/sirupsen/logrus"

	cf "github.com/nexusriot/antiworld/internal/config"
	"github.com/nexusriot/antiworld/internal/net"
	"github.com/nexusriot/antiworld/internal/processor"
	"github.com/nexusriot/antiworld/internal/utils"
)

const (
	greetings = `
░█████╗░███╗░░██╗████████╗██╗░██╗░░░░░░░██╗░█████╗░██████╗░██╗░░░░░██████╗░
██╔══██╗████╗░██║╚══██╔══╝██║░██║░░██╗░░██║██╔══██╗██╔══██╗██║░░░░░██╔══██╗
███████║██╔██╗██║░░░██║░░░██║░╚██╗████╗██╔╝██║░░██║██████╔╝██║░░░░░██║░░██║
██╔══██║██║╚████║░░░██║░░░██║░░████╔═████║░██║░░██║██╔══██╗██║░░░░░██║░░██║
██║░░██║██║░╚███║░░░██║░░░██║░░╚██╔╝░╚██╔╝░╚█████╔╝██║░░██║███████╗██████╔╝

ver. 3.0 beta`
	maxNameLen  = 55
	halfNameLen = 25
)

var (
	opts []tea.ProgramOption
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render

type App struct {
	Config    *cf.Config
	Net       *net.Net
	Processor *processor.Processor
}

type result struct {
	duration time.Duration
	emoji    string
	fileName string
	linkId   string
}

type model struct {
	spinner    spinner.Model
	results    []result
	dwn        chan *processor.FileInfo
	page       int
	totalPages int
	quitting   bool
	pages      chan int
	done       bool
	processor  *processor.Processor
	cfg        *cf.Config
}

type processFinishedMsg struct {
	duration time.Duration
	fileInfo *processor.FileInfo
}

func proxyStatus(config *cf.Config) string {
	var msg string
	if config.Proxy != nil {
		msg = "enabled"
	} else {
		msg = "disabled"
	}
	return msg
}

func (m *model) runPretendProcess() tea.Msg {
	start := time.Now()
	fi, ok := <-m.dwn

	if !ok {
		m.done = true
		m.quitting = true
	}
	page, ok := <-m.pages

	if ok {
		m.page = page
	}
	elapsed := time.Since(start)
	return processFinishedMsg{
		duration: elapsed,
		fileInfo: fi,
	}
}

func randomEmoji() string {
	emojis := []rune("📚")
	return string(emojis[rand.Intn(len(emojis))])
}

func (a *App) newModel() *model {
	const showLastResults = 10

	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("206"))

	return &model{
		spinner:   sp,
		results:   make([]result, showLastResults),
		dwn:       make(chan *processor.FileInfo),
		pages:     make(chan int),
		processor: a.Processor,
		cfg:       a.Config,
	}
}

func (m *model) Init() tea.Cmd {
	log.Info("starting work...")

	m.totalPages = m.processor.TotalPages()
	go m.processor.ProcessPages(m.totalPages, m.pages, m.dwn)

	return tea.Batch(
		spinner.Tick,
		m.runPretendProcess,
	)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// TODO: try to make graceful
		m.quitting = true
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case processFinishedMsg:
		d := msg.duration
		if m.done {
			return m, tea.Quit
		}
		res := result{emoji: randomEmoji(), duration: d, fileName: msg.fileInfo.FileName, linkId: msg.fileInfo.FileLink}
		log.Infof("%s finished in %s", res.fileName, res.duration)
		m.results = append(m.results[1:], res)
		return m, m.runPretendProcess
	default:
		return m, nil
	}
}

func trimFileName(fileName string) string {
	if len(fileName) > maxNameLen {
		return fileName[:halfNameLen] + "[..]" + fileName[len(fileName)-halfNameLen:]
	}
	return fileName
}

func (m *model) View() string {

	prcMsg := func() string {
		var msg string

		if m.done {
			msg = "done"
		} else {
			msg = fmt.Sprintf("%d/%d", m.page, m.totalPages)
		}
		return msg
	}
	s := "\n" +
		greetings + "\n\n" +
		fmt.Sprintf("Proxy: %s\n\n", proxyStatus(m.cfg)) +
		m.spinner.View() + fmt.Sprintf(" Processing page...%s\n\n", prcMsg())

	for _, res := range m.results {
		if res.duration == 0 {
			s += "..........................\n"
		} else {
			fields := strings.FieldsFunc(strings.TrimSpace(res.linkId), utils.SplitFunc)

			s += fmt.Sprintf("%s %s(id=%s) in %d sec.\n", res.emoji, trimFileName(res.fileName), fields[1], int(res.duration.Seconds()))
		}
	}
	s += helpStyle("\nPress any key to exit\n")
	if m.quitting {
		s += "\n"
	}
	return indent.String(s, 1)
}

func NewApp() *App {
	cfg := cf.LoadConfiguration()
	client := net.NewNet(cfg.Proxy)
	proc := processor.NewProcessor(cfg, client)
	return &App{
		Config:    cfg,
		Net:       client,
		Processor: proc,
	}
}

func (a *App) Start(daemonMode bool) {
	downloadFolder := a.Config.DownloadFolder
	if _, err := os.Stat(downloadFolder); os.IsNotExist(err) {
		err := os.Mkdir(downloadFolder, 0755)
		if err != nil {
			log.Fatalf("failed to create download folder: %s", err.Error())
		}
	}

	if daemonMode || !isatty.IsTerminal(os.Stdout.Fd()) {
		opts = []tea.ProgramOption{tea.WithoutRenderer()}
	}

	p := tea.NewProgram(a.newModel(), opts...)
	if err := p.Start(); err != nil {
		fmt.Println("Error starting antiworld: ", err)
		os.Exit(1)
	}
}
