package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gen2brain/beeep"
)

// --- Configuration ---

const (
	checkAnimDuration  = 290 * time.Millisecond
	deleteAnimDuration = 200 * time.Millisecond
	fps                = 60
	dataFile           = "todos.json"
)

// --- Theme Definitions ---

type Theme struct {
	Name      string
	Bg        lipgloss.Color
	Fg        lipgloss.Color
	Dim       lipgloss.Color
	Accent    lipgloss.Color
	Secondary lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
}

var themes = []Theme{
	{"Catppuccin", "#000000", "#cdd6f4", "#6c7086", "#cba6f7", "#f5c2e7", "#a6e3a1", "#f38ba8"},
	{"Nord", "#2e3440", "#eceff4", "#4c566a", "#88c0d0", "#81a1c1", "#a3be8c", "#bf616a"},
	{"Gruvbox", "#282828", "#ebdbb2", "#928374", "#fabd2f", "#fe8019", "#b8bb26", "#fb4934"},
	{"Dracula", "#282a36", "#f8f8f2", "#6272a4", "#bd93f9", "#ff79c6", "#50fa7b", "#ff5555"},
	{"Tokyo Night", "#1a1b26", "#c0caf5", "#565f89", "#7aa2f7", "#bb9af7", "#9ece6a", "#f7768e"},
	{"Rose Pine", "#191724", "#e0def4", "#6e6a86", "#ebbcba", "#c4a7e7", "#31748f", "#eb6f92"},
	{"Everforest", "#272e33", "#d3c6aa", "#859289", "#a7c080", "#7fbbb3", "#a7c080", "#e67e80"},
	{"One Dark", "#282c34", "#abb2bf", "#5c6370", "#61afef", "#c678dd", "#98c379", "#e06c75"},
	{"Solarized", "#002b36", "#839496", "#586e75", "#268bd2", "#2aa198", "#859900", "#dc322f"},
	{"Kanagawa", "#1f1f28", "#dcd7ba", "#727169", "#7e9cd8", "#957fb8", "#76946a", "#c34043"},
}

// --- Styles ---

var (
	appStyle          lipgloss.Style
	headerStyle       lipgloss.Style
	listSelectedStyle lipgloss.Style
	listItemStyle     lipgloss.Style
	inlineInputStyle  lipgloss.Style
	strikeStyle       lipgloss.Style
	binaryStyle       lipgloss.Style
	helpStyle         lipgloss.Style
	dueStyle          lipgloss.Style
	overdueStyle      lipgloss.Style
)

func updateStyles(t Theme) {
	appStyle = lipgloss.NewStyle().Padding(1).Background(t.Bg)

	headerStyle = lipgloss.NewStyle().
		Foreground(t.Bg).
		Background(t.Accent).
		Bold(true).
		Padding(0, 1).
		MarginBottom(1)

	listSelectedStyle = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(t.Accent).
		PaddingLeft(1).
		Foreground(t.Accent).
		Bold(true)

	listItemStyle = lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(t.Fg)

	inlineInputStyle = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	strikeStyle = lipgloss.NewStyle().Foreground(t.Dim).Strikethrough(true)
	binaryStyle = lipgloss.NewStyle().Foreground(t.Warning).Bold(true)
	helpStyle = lipgloss.NewStyle().Foreground(t.Dim)

	dueStyle = lipgloss.NewStyle().Foreground(t.Secondary).Italic(true)
	overdueStyle = lipgloss.NewStyle().Foreground(t.Warning).Bold(true).Blink(true)
}

// --- Types ---

type AppState int
type SortMode int

const (
	StateBrowse AppState = iota
	StateEditing
	StateCreating
	StateSettingTime
)

const (
	SortOff SortMode = iota
	SortTodoFirst
	SortDoneFirst
)

// 30 Distinct Animations
const (
	AnimSparkle = iota
	AnimMatrix
	AnimWipeRight
	AnimWipeLeft
	AnimRainbow
	AnimWave
	AnimBinary
	AnimDissolve
	AnimFlip
	AnimPulse
	AnimTypewriter
	AnimParticle
	AnimRedact
	AnimChaos
	AnimConverge
	AnimBounce
	AnimSpin
	AnimZipper
	AnimEraser
	AnimGlitch
	// NEW 10
	AnimMoons
	AnimBraille
	AnimHex
	AnimReverse
	AnimCaseFlip
	AnimWide
	AnimTraffic
	AnimCenterStrike
	AnimLoading
	AnimSlider

	AnimCount = 30
)

type Task struct {
	ID       int64     `json:"id"`
	Title    string    `json:"title"`
	Done     bool      `json:"done"`
	DueAt    time.Time `json:"dueAt"`
	Notified bool      `json:"notified"`

	// Animation States
	IsAnimatingCheck bool      `json:"-"`
	IsDeleting       bool      `json:"-"`
	AnimType         int       `json:"-"`
	AnimStart        time.Time `json:"-"`
}

type AppData struct {
	ThemeIndex int      `json:"themeIndex"`
	SortMode   SortMode `json:"sortMode"`
	Tasks      []Task   `json:"tasks"`
}

type TickMsg struct{}

// --- Model ---

type model struct {
	tasks      []Task
	state      AppState
	sortMode   SortMode
	themeIndex int
	lastAnim   int

	cursor    int
	width     int
	height    int
	textInput textinput.Model
}

// --- Persistence ---

func loadData() AppData {
	data, err := os.ReadFile(dataFile)

	hints := []Task{
		{ID: 1, Title: "Press 'n' to add a new task", Done: false},
		{ID: 2, Title: "Press 'e' to edit the selected task", Done: false},
		{ID: 3, Title: "Press 'd' to delete a task", Done: false},
		{ID: 4, Title: "Press 'space' to check/uncheck", Done: true},
		{ID: 5, Title: "Press '@' to set a timer notification", Done: false},
		{ID: 6, Title: "Press 's' to cycle sort modes", Done: false},
		{ID: 7, Title: "Press 't' to change the color theme", Done: false},
	}

	defaultData := AppData{
		ThemeIndex: 0,
		SortMode:   SortOff,
		Tasks:      hints,
	}

	if err != nil {
		return defaultData
	}

	var appData AppData
	if err := json.Unmarshal(data, &appData); err == nil {
		for i := range appData.Tasks {
			if appData.Tasks[i].ID == 0 {
				appData.Tasks[i].ID = time.Now().UnixNano() + int64(i)
			}
		}
		return appData
	}
	return defaultData
}

func saveModel(m model) {
	var validTasks []Task
	for _, t := range m.tasks {
		if !t.IsDeleting {
			validTasks = append(validTasks, t)
		}
	}
	data := AppData{
		ThemeIndex: m.themeIndex,
		SortMode:   m.sortMode,
		Tasks:      validTasks,
	}
	bytes, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(dataFile, bytes, 0644)
}

// --- Logic ---

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/time.Duration(fps), func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}

func (m *model) applySort() {
	sort.SliceStable(m.tasks, func(i, j int) bool {
		t1, t2 := m.tasks[i], m.tasks[j]
		switch m.sortMode {
		case SortTodoFirst:
			if t1.Done != t2.Done {
				return !t1.Done
			}
		case SortDoneFirst:
			if t1.Done != t2.Done {
				return t1.Done
			}
		}
		return t1.ID < t2.ID
	})
	if m.cursor >= len(m.tasks) && len(m.tasks) > 0 {
		m.cursor = len(m.tasks) - 1
	}
}

// --- Renderers ---

func renderCheckAnim(t Task, theme Theme) string {
	elapsed := time.Since(t.AnimStart).Seconds()
	total := checkAnimDuration.Seconds()
	progress := elapsed / total
	if progress > 1.0 {
		progress = 1.0
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	text := t.Title
	fallBack := lipgloss.NewStyle().Foreground(theme.Success).Render(text)

	switch t.AnimType {
	// 1-5
	case AnimSparkle:
		chars := []string{"*", "+", "°", ".", "x", "o"}
		var sb strings.Builder
		for i := 0; i < len(text); i++ {
			if r.Float32() < 0.4 {
				char := chars[r.Intn(len(chars))]
				col := theme.Accent
				if r.Intn(2) == 0 {
					col = theme.Secondary
				}
				sb.WriteString(lipgloss.NewStyle().Foreground(col).Render(char))
			} else {
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Dim).Render(string(text[i])))
			}
		}
		return sb.String()
	case AnimMatrix:
		matrixChars := "H3LL0W0RLD$#@!%*&^"
		var sb strings.Builder
		for i := 0; i < len(text); i++ {
			char := string(matrixChars[r.Intn(len(matrixChars))])
			sb.WriteString(lipgloss.NewStyle().Foreground(theme.Success).Render(char))
		}
		return sb.String()
	case AnimWipeRight:
		idx := int(math.Floor(progress * float64(len(text))))
		var sb strings.Builder
		for i := 0; i < len(text); i++ {
			if i < idx {
				sb.WriteString(strikeStyle.Render(string(text[i])))
			}
			if i == idx {
				sb.WriteString(lipgloss.NewStyle().Background(theme.Secondary).Foreground(theme.Bg).Render(string(text[i])))
			}
			if i > idx {
				sb.WriteString(string(text[i]))
			}
		}
		return sb.String()
	case AnimWipeLeft:
		idx := len(text) - 1 - int(math.Floor(progress*float64(len(text))))
		var sb strings.Builder
		for i := 0; i < len(text); i++ {
			if i > idx {
				sb.WriteString(strikeStyle.Render(string(text[i])))
			}
			if i == idx {
				sb.WriteString(lipgloss.NewStyle().Background(theme.Accent).Foreground(theme.Bg).Render(string(text[i])))
			}
			if i < idx {
				sb.WriteString(string(text[i]))
			}
		}
		return sb.String()
	case AnimRainbow:
		colors := []lipgloss.Color{theme.Accent, theme.Secondary, theme.Success, theme.Warning, "#FF0000", "#00FF00", "#0000FF"}
		var sb strings.Builder
		for _, char := range text {
			c := colors[r.Intn(len(colors))]
			sb.WriteString(lipgloss.NewStyle().Foreground(c).Render(string(char)))
		}
		return sb.String()

	// 6-10
	case AnimWave:
		colors := []lipgloss.Color{theme.Accent, theme.Secondary, theme.Success, theme.Fg}
		offset := int(elapsed * 30)
		var sb strings.Builder
		for i, char := range text {
			cIdx := (i + offset) % len(colors)
			sb.WriteString(lipgloss.NewStyle().Foreground(colors[cIdx]).Render(string(char)))
		}
		return sb.String()
	case AnimBinary:
		var sb strings.Builder
		for i := 0; i < len(text); i++ {
			bit := "0"
			if r.Intn(2) == 1 {
				bit = "1"
			}
			sb.WriteString(lipgloss.NewStyle().Foreground(theme.Success).Render(bit))
		}
		return sb.String()
	case AnimDissolve:
		var sb strings.Builder
		for i := 0; i < len(text); i++ {
			if r.Float64() < progress*1.5 {
				sb.WriteString(strikeStyle.Render(string(text[i])))
			} else {
				sb.WriteString(string(text[i]))
			}
		}
		return sb.String()
	case AnimFlip:
		var sb strings.Builder
		for _, char := range text {
			s := string(char)
			if r.Float32() < 0.3 {
				if strings.ToUpper(s) == s {
					s = strings.ToLower(s)
				} else {
					s = strings.ToUpper(s)
				}
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Secondary).Render(s))
			} else {
				sb.WriteString(s)
			}
		}
		return sb.String()
	case AnimPulse:
		var sb strings.Builder
		phase := math.Sin(elapsed * 40)
		col := theme.Fg
		if phase > 0 {
			col = theme.Accent
		}
		for _, char := range text {
			sb.WriteString(lipgloss.NewStyle().Foreground(col).Bold(phase > 0).Render(string(char)))
		}
		return sb.String()

	// 11-15
	case AnimTypewriter:
		visibleChars := int(float64(len(text)) * progress)
		var sb strings.Builder
		for i := 0; i < len(text); i++ {
			if i <= visibleChars {
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Success).Render(string(text[i])))
			} else {
				sb.WriteString(" ")
			}
		}
		return sb.String()
	case AnimParticle:
		var sb strings.Builder
		for i := 0; i < len(text); i++ {
			if r.Float32() < 0.5 {
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Secondary).Render("."))
			} else {
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Accent).Render(string(text[i])))
			}
		}
		return sb.String()
	case AnimRedact:
		var sb strings.Builder
		chars := []string{"█", "▓", "▒", "░"}
		for i := 0; i < len(text); i++ {
			if r.Float32() < 0.5 {
				char := chars[r.Intn(len(chars))]
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Warning).Render(char))
			} else {
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Dim).Render(string(text[i])))
			}
		}
		return sb.String()
	case AnimChaos:
		symbols := "!@#$%^&*()_+"
		var sb strings.Builder
		for i := 0; i < len(text); i++ {
			if r.Float32() < 0.5 {
				s := string(symbols[r.Intn(len(symbols))])
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Secondary).Render(s))
			} else {
				sb.WriteString(string(text[i]))
			}
		}
		return sb.String()
	case AnimConverge:
		var sb strings.Builder
		mid := len(text) / 2
		fill := int(float64(mid) * progress)
		for i := 0; i < len(text); i++ {
			if i < fill || i >= len(text)-fill {
				sb.WriteString(strikeStyle.Render(string(text[i])))
			} else {
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Accent).Render(string(text[i])))
			}
		}
		return sb.String()

	// 16-20
	case AnimBounce:
		var sb strings.Builder
		for i, char := range text {
			if r.Intn(2) == 0 {
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Accent).Render(string(char)))
			} else {
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Secondary).Render(string(char)))
			}
			if i%2 == int(elapsed*10)%2 {
				sb.WriteString("")
			}
		}
		return sb.String()
	case AnimSpin:
		spinners := []string{"-", "\\", "|", "/"}
		spinIdx := int(elapsed*20) % 4
		var sb strings.Builder
		for range text {
			sb.WriteString(lipgloss.NewStyle().Foreground(theme.Success).Render(spinners[spinIdx]))
		}
		return sb.String()
	case AnimZipper:
		var sb strings.Builder
		mid := len(text) / 2
		zipperPos := int(progress * float64(mid))
		for i := 0; i < len(text); i++ {
			distFromEdge := i
			if i >= mid {
				distFromEdge = len(text) - 1 - i
			}

			if distFromEdge < zipperPos {
				sb.WriteString(strikeStyle.Render(string(text[i])))
			} else {
				sb.WriteString(lipgloss.NewStyle().Background(theme.Accent).Foreground(theme.Bg).Render(string(text[i])))
			}
		}
		return sb.String()
	case AnimEraser:
		var sb strings.Builder
		for i := 0; i < len(text); i++ {
			if r.Float64() < progress {
				sb.WriteString(" ")
			} else {
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Dim).Render(string(text[i])))
			}
		}
		return sb.String()
	case AnimGlitch:
		var sb strings.Builder
		glitchChars := "¡¢£¤¥¦§¨©ª«¬®¯°±²³´µ¶·¸¹º»¼½¾¿"
		for i := 0; i < len(text); i++ {
			if r.Float32() < 0.3 {
				char := string(glitchChars[r.Intn(len(glitchChars))])
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Warning).Background(theme.Dim).Render(char))
			} else {
				sb.WriteString(string(text[i]))
			}
		}
		return sb.String()

	// 21-30 (NEW)
	case AnimMoons:
		phases := []string{"◐", "◓", "◑", "◒"}
		var sb strings.Builder
		for i := 0; i < len(text); i++ {
			if r.Float32() < 0.3 {
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Secondary).Render(phases[r.Intn(len(phases))]))
			} else {
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Dim).Render(string(text[i])))
			}
		}
		return sb.String()
	case AnimBraille:
		var sb strings.Builder
		for i := 0; i < len(text); i++ {
			if r.Float32() < 0.4 {
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Accent).Render(string(rune(0x2800 + r.Intn(255)))))
			} else {
				sb.WriteString(string(text[i]))
			}
		}
		return sb.String()
	case AnimHex:
		var sb strings.Builder
		hexChars := "0123456789ABCDEF"
		for i := 0; i < len(text); i++ {
			sb.WriteString(lipgloss.NewStyle().Foreground(theme.Success).Render(string(hexChars[r.Intn(len(hexChars))])))
		}
		return sb.String()
	case AnimReverse:
		var sb strings.Builder
		for i := len(text) - 1; i >= 0; i-- {
			sb.WriteString(lipgloss.NewStyle().Foreground(theme.Warning).Render(string(text[i])))
		}
		return sb.String()
	case AnimCaseFlip:
		var sb strings.Builder
		for _, char := range text {
			s := string(char)
			if r.Intn(2) == 0 {
				s = strings.ToUpper(s)
			} else {
				s = strings.ToLower(s)
			}
			sb.WriteString(lipgloss.NewStyle().Foreground(theme.Accent).Render(s))
		}
		return sb.String()
	case AnimWide:
		var sb strings.Builder
		for _, char := range text {
			sb.WriteString(lipgloss.NewStyle().Foreground(theme.Secondary).Render(string(char) + " "))
		}
		return sb.String()
	case AnimTraffic:
		colors := []lipgloss.Color{theme.Warning, "#FFFF00", theme.Success}
		cIdx := int(elapsed*10) % 3
		return lipgloss.NewStyle().Foreground(colors[cIdx]).Render(text)
	case AnimCenterStrike:
		var sb strings.Builder
		mid := len(text) / 2
		strikeWidth := int(progress * float64(mid))
		for i := 0; i < len(text); i++ {
			dist := int(math.Abs(float64(i - mid)))
			if dist < strikeWidth {
				sb.WriteString(strikeStyle.Render(string(text[i])))
			} else {
				sb.WriteString(string(text[i]))
			}
		}
		return sb.String()
	case AnimLoading:
		var sb strings.Builder
		fill := int(progress * float64(len(text)))
		for i := 0; i < len(text); i++ {
			if i < fill {
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Success).Render("█"))
			} else {
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Dim).Render("▒"))
			}
		}
		return sb.String()
	case AnimSlider:
		var sb strings.Builder
		for i := 0; i < len(text); i++ {
			if r.Float32() < 0.3 {
				sb.WriteString(lipgloss.NewStyle().Foreground(theme.Accent).Render("^"))
			} else {
				sb.WriteString(string(text[i]))
			}
		}
		return sb.String()
	}

	return fallBack
}

func renderDeleteAnim(text string, theme Theme) string {
	var sb strings.Builder
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < len(text); i++ {
		bit := "0"
		if r.Intn(2) == 1 {
			bit = "1"
		}
		sb.WriteString(lipgloss.NewStyle().Foreground(theme.Warning).Bold(true).Render(bit))
	}
	return sb.String()
}

// --- Update ---

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:

		if m.state == StateEditing || m.state == StateCreating || m.state == StateSettingTime {
			switch msg.String() {
			case "enter":
				val := m.textInput.Value()

				if m.state == StateSettingTime {
					if val != "" {
						dur, err := time.ParseDuration(val)
						if err == nil {
							m.tasks[m.cursor].DueAt = time.Now().Add(dur)
							m.tasks[m.cursor].Notified = false // Reset notification
						} else {
							m.tasks[m.cursor].DueAt = time.Time{}
						}
					} else {
						m.tasks[m.cursor].DueAt = time.Time{}
					}
					saveModel(m)
					m.state = StateBrowse
					m.textInput.Blur()
					// Jumpstart ticker for timer updates
					return m, tickCmd()
				}

				if val == "" {
					m.state = StateBrowse
					m.textInput.Blur()
					return m, nil
				}

				if m.state == StateCreating {
					m.tasks = append(m.tasks, Task{
						ID:    time.Now().UnixNano(),
						Title: val,
					})
					if m.sortMode != SortOff {
						m.applySort()
					}
					saveModel(m)
					m.state = StateBrowse
					m.textInput.Blur()
					if m.sortMode == SortOff {
						m.cursor = len(m.tasks) - 1
					}
					return m, nil
				} else {
					m.tasks[m.cursor].Title = val
					saveModel(m)
					m.state = StateBrowse
					m.textInput.Blur()
					return m, nil
				}

			case "esc":
				m.state = StateBrowse
				m.textInput.Blur()
				return m, nil
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "q", "ctrl+c":
			saveModel(m)
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}

		case "t":
			m.themeIndex = (m.themeIndex + 1) % len(themes)
			updateStyles(themes[m.themeIndex])
			saveModel(m)

		case "s":
			m.sortMode = (m.sortMode + 1) % 3
			m.applySort()
			saveModel(m)

		case "n":
			m.state = StateCreating
			m.textInput.Placeholder = "Task name..."
			m.textInput.SetValue("")
			m.textInput.Focus()
			m.cursor = len(m.tasks)
			return m, textinput.Blink

		case "e":
			if len(m.tasks) > 0 {
				m.state = StateEditing
				m.textInput.SetValue(m.tasks[m.cursor].Title)
				m.textInput.Focus()
				m.textInput.SetCursor(len(m.textInput.Value()))
				return m, textinput.Blink
			}

		case "@":
			if len(m.tasks) > 0 {
				m.state = StateSettingTime
				m.textInput.Placeholder = "e.g. 10m, 1h2s, 10s..."
				m.textInput.SetValue("")
				m.textInput.Focus()
				return m, textinput.Blink
			}

		case "d":
			if len(m.tasks) > 0 {
				m.tasks[m.cursor].IsDeleting = true
				m.tasks[m.cursor].AnimStart = time.Now()
				cmds = append(cmds, tickCmd())
			}

		case " ", "enter":
			if len(m.tasks) > 0 {
				t := &m.tasks[m.cursor]
				t.Done = !t.Done

				if t.Done {
					t.IsAnimatingCheck = true
					t.AnimStart = time.Now()

					// Force Unique Random Animation
					newAnim := rand.Intn(AnimCount)
					for newAnim == m.lastAnim {
						newAnim = rand.Intn(AnimCount)
					}
					t.AnimType = newAnim
					m.lastAnim = newAnim

					cmds = append(cmds, tickCmd())
				} else {
					t.IsAnimatingCheck = false
				}
				m.applySort()
				saveModel(m)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width - 10

	case TickMsg:
		needsTick := false
		for i := len(m.tasks) - 1; i >= 0; i-- {
			t := &m.tasks[i]

			// Animations
			if t.IsDeleting {
				if time.Since(t.AnimStart) > deleteAnimDuration {
					m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
					if m.cursor >= len(m.tasks) && m.cursor > 0 {
						m.cursor--
					}
					saveModel(m)
				} else {
					needsTick = true
				}
			}
			if t.IsAnimatingCheck {
				if time.Since(t.AnimStart) > checkAnimDuration {
					t.IsAnimatingCheck = false
				} else {
					needsTick = true
				}
			}
			// Timer updates
			if !t.Done && !t.DueAt.IsZero() {
				needsTick = true
				if time.Now().After(t.DueAt) && !t.Notified {
					// CORRECTED NOTIFICATION
					beeep.Notify("Todo Alert!", t.Title, "")
					t.Notified = true
					saveModel(m)
				}
			}
		}
		if needsTick {
			cmds = append(cmds, tickCmd())
		}
	}

	return m, tea.Batch(cmds...)
}

// --- View ---

func (m model) View() string {
	currentTheme := themes[m.themeIndex]
	var content string

	content = m.viewList(currentTheme)

	header := headerStyle.Render("// TODO LIST")

	container := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(currentTheme.Accent).
		Width(min(m.width-4, 100)).
		Height(m.height - 7).
		Render(content)

	sortStr := "Off"
	if m.sortMode == SortTodoFirst {
		sortStr = "Todo"
	}
	if m.sortMode == SortDoneFirst {
		sortStr = "Done"
	}

	help := fmt.Sprintf("Theme: %s (t) • Sort: %s (s) • New (n) • Edit (e) • Check (Space) • Notify (@) • Del (d)", currentTheme.Name, sortStr)
	status := helpStyle.Render(help)

	ui := lipgloss.JoinVertical(lipgloss.Center, header, container, status)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, ui)
}

func (m *model) viewList(t Theme) string {
	if len(m.tasks) == 0 && m.state != StateCreating {
		return helpStyle.Padding(2).Render("No tasks.")
	}
	var s strings.Builder

	count := len(m.tasks)
	if m.state == StateCreating {
		count++
	}
	creatingIndex := len(m.tasks)

	// Layout Calc: Window - Borders(2) - Number(4) - Icon(3) - Timer(approx 25) - Spacers(6)
	availableWidth := min(m.width-4, 100)
	textWidth := availableWidth - 40 // Give extra room for timer
	if textWidth < 10 {
		textWidth = 10
	}

	for i := 0; i < count; i++ {
		selected := false
		if m.state == StateCreating {
			if i == creatingIndex {
				selected = true
			}
		} else {
			if m.cursor == i {
				selected = true
			}
		}

		numberStr := fmt.Sprintf("%d.", i+1)
		var checkIcon string
		var titleContent string
		var dueContent string

		isEditingThis := (m.state == StateEditing && i == m.cursor)
		isCreatingThis := (m.state == StateCreating && i == creatingIndex)
		isSettingTime := (m.state == StateSettingTime && i == m.cursor)

		if isEditingThis || isCreatingThis {
			checkIcon = lipgloss.NewStyle().Foreground(t.Accent).Render(">")
			m.textInput.Width = textWidth
			titleContent = inlineInputStyle.Render(m.textInput.View())
		} else {
			task := m.tasks[i]

			if task.Done {
				checkIcon = lipgloss.NewStyle().Foreground(t.Success).Render("[✔]")
			} else {
				checkIcon = lipgloss.NewStyle().Foreground(t.Accent).Render("[ ]")
			}

			var rawTitle string
			if task.IsDeleting {
				rawTitle = renderDeleteAnim(task.Title, t)
			} else if task.IsAnimatingCheck {
				rawTitle = renderCheckAnim(task, t)
			} else if task.Done {
				rawTitle = strikeStyle.Render(task.Title)
			} else {
				rawTitle = lipgloss.NewStyle().Foreground(t.Fg).Render(task.Title)
			}

			titleContent = lipgloss.NewStyle().Width(textWidth).Render(rawTitle)

			if isSettingTime {
				m.textInput.Width = 20
				dueContent = inlineInputStyle.Render(m.textInput.View())
			} else if !task.DueAt.IsZero() && !task.Done {
				timeRemaining := time.Until(task.DueAt)
				if timeRemaining < 0 {
					dueContent = overdueStyle.Render("[OVERDUE]")
				} else {
					dueContent = dueStyle.Render(shortDur(timeRemaining))
				}
			}
		}

		leftBlock := lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Foreground(t.Dim).Width(4).Align(lipgloss.Right).Render(numberStr),
			" ",
			lipgloss.NewStyle().Width(3).Align(lipgloss.Center).Render(checkIcon),
			" ",
		)

		row := lipgloss.JoinHorizontal(lipgloss.Top,
			leftBlock,
			titleContent,
			"   ",
			dueContent,
		)

		if selected {
			s.WriteString(listSelectedStyle.Render(row))
		} else {
			s.WriteString(listItemStyle.Render(row))
		}
		s.WriteString("\n")
	}
	return s.String()
}

func shortDur(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 50
	ti.Prompt = ""

	rand.Seed(time.Now().UnixNano())

	data := loadData()
	initialModel := model{
		tasks:      data.Tasks,
		state:      StateBrowse,
		sortMode:   data.SortMode,
		themeIndex: data.ThemeIndex,
		textInput:  ti,
	}

	if initialModel.themeIndex >= len(themes) {
		initialModel.themeIndex = 0
	}
	updateStyles(themes[initialModel.themeIndex])
	initialModel.applySort()

	p := tea.NewProgram(initialModel, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
	}
}
