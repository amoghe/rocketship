package shell

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/amoghe/distillog"
	"github.com/peterh/liner"
)

const (
	DefaultCommandsDir     = "/opt/shellcommands"
	DefaultHistoryFilePath = "/tmp/.shellhistory" // TODO: In the homedir
)

type Shell struct {
	Prompt          string // Prompt to display to the user.
	CommandsDirPath string // CommandsDirPath is where commands get loaded from.
	HistoryFilePath string // HistoryFilePath is where command history gets written to. Empty string disables it.
	Log             distillog.Logger

	line         *liner.State
	commandNames map[string]bool
}

func New() *Shell {
	return &Shell{
		Log: distillog.NewNullLogger("shell"),

		line:         liner.NewLiner(),
		commandNames: make(map[string]bool),
	}
}

func (s *Shell) Initialize() error {
	if s.CommandsDirPath == "" {
		s.CommandsDirPath = DefaultCommandsDir
	}

	if len(s.Prompt) <= 0 {
		s.Prompt = "> "
	}

	files, err := ioutil.ReadDir(s.CommandsDirPath)
	if err != nil {
		return fmt.Errorf("Failed to initialize commands: %s", err)
	}

	for _, f := range files {
		s.commandNames[f.Name()] = true
	}

	if len(s.HistoryFilePath) > 0 {
		if f, err := os.Open(s.HistoryFilePath); err == nil {
			s.line.ReadHistory(f)
			f.Close()
		}
	}

	return nil
}

func (s *Shell) CommandFinder(line string, pos int) (head string, completions []string, tail string) {
	commandPrefix := line[0:pos] // the potential prefix string we're trying to complete

	if strings.Contains(commandPrefix, " ") {
		// we only provide completion for the first word (the command), not its arguments
		return
	}

	for commandName, _ := range s.commandNames {
		if strings.HasPrefix(commandName, commandPrefix) {
			completions = append(completions, commandName)
		}
	}

	return
}

func (s *Shell) ExecuteCmdline(cmdline string) {
	tokens := strings.Split(cmdline, " ")

	if len(tokens) <= 0 {
		s.OutputErrorln("No command specified")
		return
	}
	if _, there := s.commandNames[tokens[0]]; !there {
		s.OutputErrorln("Invalid command specified")
		return
	}

	s.Log.Debugf("Executing cmd:[%s], args:%s", tokens[0], tokens[1:])

	cmd := exec.Cmd{
		Path:   fmt.Sprintf("./%s", tokens[0]), // relative to Dir
		Args:   tokens,
		Dir:    s.CommandsDirPath,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := cmd.Run(); err != nil {
		switch err.(type) {
		case *exec.ExitError:
			s.Log.Errorln("% ERROR: command failed to execute succesfully")
		default:
			s.Log.Errorln("% Internal error:", err.Error())
		}
	}
}

func (s *Shell) Run() {
	if err := s.Initialize(); err != nil {
		s.Log.Errorln("Failed to run:", err)
		return
	}

	s.line.SetCtrlCAborts(false)
	s.line.SetWordCompleter(s.CommandFinder)
	s.line.SetTabCompletionStyle(liner.TabPrints)

	for {
		if cmdline, err := s.line.Prompt(s.Prompt); err == nil {
			s.ExecuteCmdline(cmdline)
			s.line.AppendHistory(cmdline)
		} else if err == liner.ErrPromptAborted {
			break
		} else {
			if err == io.EOF {
				break
			}
			s.Log.Errorf("Error reading line: [%s]", err)
		}
	}
}

func (s *Shell) Close() {
	s.line.Close()

	if len(s.HistoryFilePath) > 0 {
		if f, err := os.Create(s.HistoryFilePath); err != nil {
			s.Log.Errorln("Error writing history file: ", err)
		} else {
			s.line.WriteHistory(f)
			f.Close()
		}
	}
}

func (s Shell) Outputln(args ...interface{}) {
	fmt.Println(args...)
}

func (s Shell) Outputf(f string, args ...interface{}) {
	fmt.Printf(f, args...)
}

func (s Shell) OutputErrorln(args ...interface{}) {
	fmt.Printf("%% %s", fmt.Sprintln(args...))
}

func (s Shell) OutputErrorf(f string, args ...interface{}) {
	fmt.Printf("%% %s", fmt.Sprintf(f, args...))
}
