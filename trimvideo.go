package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var videoExts = map[string]bool{
	".mp4": true, ".mkv": true, ".avi": true, ".mov": true,
	".wmv": true, ".flv": true, ".webm": true, ".m4v": true,
}

var app *tview.Application
var pages *tview.Pages
var tree *tview.TreeView

func main() {
	app = tview.NewApplication()
	pages = tview.NewPages()

	// --- Apply Japanese Blossom Theme ---
	tview.Styles.PrimitiveBackgroundColor = tcell.GetColor("#2A0826")
	tview.Styles.ContrastBackgroundColor = tcell.GetColor("#5C114C")
	tview.Styles.MoreContrastBackgroundColor = tcell.GetColor("#8B1E75")
	tview.Styles.BorderColor = tcell.GetColor("#FFB7C5")
	tview.Styles.TitleColor = tcell.GetColor("#FF69B4")
	tview.Styles.PrimaryTextColor = tcell.GetColor("#FFD1DC")
	tview.Styles.SecondaryTextColor = tcell.GetColor("#E0B0FF")

	// --- Setup the Tree View ---
	tree = tview.NewTreeView()
	cwd, _ := os.Getwd()
	setTreeRoot(cwd)

	// --- Setup the Form Inputs ---
	var selectedFile string
	var maxDuration float64

	form := tview.NewForm()
	form.SetButtonBackgroundColor(tcell.GetColor("#8B1E75"))
	form.SetButtonTextColor(tcell.GetColor("#FFD1DC"))
	form.SetFieldBackgroundColor(tcell.GetColor("#5C114C"))
	form.SetFieldTextColor(tcell.GetColor("#FFFFFF"))

	startInput := tview.NewInputField().SetLabel("Start Time (HH:MM:SS): ").SetFieldWidth(12)
	endInput := tview.NewInputField().SetLabel("End Time (HH:MM:SS): ").SetFieldWidth(12)

	// --- Form Layout (Centered Modal using Flex) ---
	centeredForm := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 11, 1, true).
			AddItem(nil, 0, 1, false),
			60, 1, true).
		AddItem(nil, 0, 1, false)

	modal := tview.NewModal().
		SetText("🌸 Process completed successfully 🌸").
		AddButtons([]string{"OK"}).
		SetBackgroundColor(tcell.GetColor("#4A0E4E")).
		SetTextColor(tcell.GetColor("#FFB7C5")).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.HidePage("modal")
			app.SetFocus(tree)
		})

	// --- Form Key Captures (Up/Down) ---
	startInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyUp || event.Key() == tcell.KeyDown {
			secs := parseTimeToSeconds(startInput.GetText())
			if event.Key() == tcell.KeyUp {
				secs++
			}
			if event.Key() == tcell.KeyDown {
				secs--
			}

			if secs < 0 {
				secs = 0
			}
			if secs > maxDuration {
				secs = maxDuration
			}

			startInput.SetText(formatTime(secs))
			return nil
		}
		return event
	})

	endInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyUp || event.Key() == tcell.KeyDown {
			secs := parseTimeToSeconds(endInput.GetText())
			if event.Key() == tcell.KeyUp {
				secs++
			}
			if event.Key() == tcell.KeyDown {
				secs--
			}

			if secs < 0 {
				secs = 0
			}
			if secs > maxDuration {
				secs = maxDuration
			}

			endInput.SetText(formatTime(secs))
			return nil
		}
		return event
	})

	// --- Tree Navigation Logic ---
	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		reference := node.GetReference()
		if reference == nil {
			return
		}
		path := reference.(string)

		if path == "" {
			setTreeRoot("")
			return
		}

		stat, err := os.Stat(path)
		if err != nil {
			return
		}

		if stat.IsDir() {
			setTreeRoot(path)
		} else {
			selectedFile = path
			maxDuration = getVideoDuration(selectedFile)

			startInput.SetText("00:00:00")
			endInput.SetText(formatTime(maxDuration))

			form.SetTitle(fmt.Sprintf(" 🌸 Trimming: %s 🌸 ", filepath.Base(selectedFile)))

			pages.ShowPage("form")
			app.SetFocus(form)
		}
	})

	// --- Tree Backspace Navigation ---
	tree.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyBackspace || event.Key() == tcell.KeyBackspace2 {
			rootNode := tree.GetRoot()
			if rootNode != nil && rootNode.GetReference() != nil {
				currPath := rootNode.GetReference().(string)
				if currPath != "" {
					parentDir := filepath.Dir(currPath)

					if parentDir == currPath && runtime.GOOS == "windows" {
						setTreeRoot("")
					} else {
						setTreeRoot(parentDir)
					}
					return nil
				}
			}
		}
		return event
	})

	// --- Form Assembly ---
	form.SetBorder(true).SetTitleAlign(tview.AlignCenter)
	form.AddFormItem(startInput).
		AddFormItem(endInput).
		AddButton("Trim", func() {
			start := startInput.GetText()
			end := endInput.GetText()

			if start == "" || end == "" {
				return
			}

			outputFile := generateSafeFilename(selectedFile)

			pages.HidePage("form")
			app.Suspend(func() {
				runFFmpeg(selectedFile, start, end, outputFile)
			})
			pages.ShowPage("modal")
		}).
		AddButton("Cancel", func() {
			pages.HidePage("form")
			app.SetFocus(tree)
		})

	pages.AddPage("tree", tree, true, true)
	pages.AddPage("form", centeredForm, true, false)
	pages.AddPage("modal", modal, true, false)

	// Global Escape
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			app.Stop()
		}
		return event
	})

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

// --- Video Metadata Helper ---

func getVideoDuration(path string) float64 {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0
	}
	return duration
}

func formatTime(s float64) string {
	h := int(s) / 3600
	m := (int(s) % 3600) / 60
	sec := int(s) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, sec)
}

// --- Tree & Path Helpers ---

func setTreeRoot(path string) {
	var root *tview.TreeNode

	if path == "" && runtime.GOOS == "windows" {
		root = tview.NewTreeNode("🌸 My Computer (Drives) 🌸").
			SetColor(tcell.GetColor("#FF69B4")).SetSelectable(false)

		for _, d := range getDrives() {
			node := tview.NewTreeNode("💾 " + d).
				SetReference(d).SetColor(tcell.GetColor("#E0B0FF")).SetSelectable(true)
			root.AddChild(node)
		}
	} else {
		rootName := filepath.Base(path)
		if path == filepath.VolumeName(path)+"\\" {
			rootName = "🌸 " + path
		}

		root = tview.NewTreeNode(rootName).
			SetColor(tcell.GetColor("#FF69B4")).SetReference(path).SetSelectable(false)

		parentDir := filepath.Dir(path)
		if parentDir != path {
			upNode := tview.NewTreeNode("🔙 .. (Go Up)").
				SetReference(parentDir).SetColor(tcell.GetColor("#FFD1DC")).SetSelectable(true)
			root.AddChild(upNode)
		} else if runtime.GOOS == "windows" {
			upNode := tview.NewTreeNode("🔙 .. (View All Drives)").
				SetReference("").SetColor(tcell.GetColor("#FFD1DC")).SetSelectable(true)
			root.AddChild(upNode)
		}

		files, err := os.ReadDir(path)
		if err == nil {
			for _, file := range files {
				if file.IsDir() {
					node := tview.NewTreeNode("📁 " + file.Name()).
						SetReference(filepath.Join(path, file.Name())).
						SetColor(tcell.GetColor("#E0B0FF")).SetSelectable(true)
					root.AddChild(node)
				}
			}
			for _, file := range files {
				if !file.IsDir() && videoExts[strings.ToLower(filepath.Ext(file.Name()))] {
					node := tview.NewTreeNode("🎬 " + file.Name()).
						SetReference(filepath.Join(path, file.Name())).
						SetColor(tcell.GetColor("#FFB7C5")).SetSelectable(true)
					root.AddChild(node)
				}
			}
		}
	}
	tree.SetRoot(root).SetCurrentNode(root)
}

func getDrives() []string {
	var drives []string
	for _, drive := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		path := string(drive) + ":\\"
		if _, err := os.Stat(path); err == nil {
			drives = append(drives, path)
		}
	}
	return drives
}

func generateSafeFilename(inputPath string) string {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)

	newName := fmt.Sprintf("%s_trimed%s", base, ext)
	fullPath := filepath.Join(dir, newName)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fullPath
	}

	counter := 1
	for {
		newName = fmt.Sprintf("%s_trimed%02d%s", base, counter, ext)
		fullPath = filepath.Join(dir, newName)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fullPath
		}
		counter++
	}
}

// --- FFmpeg Engine ---

func runFFmpeg(inputFile, startTime, endTime, outputFile string) {
	encoder := getBestEncoder()
	totalDur := parseTimeToSeconds(endTime) - parseTimeToSeconds(startTime)

	pinkStart := "\033[38;2;255;183;197m"
	reset := "\033[0m"

	fmt.Printf("\n🌸 %sTrimming -> %s%s\n", pinkStart, filepath.Base(outputFile), reset)

	args := []string{"-y", "-ss", startTime, "-to", endTime, "-i", inputFile, "-c:v", encoder}

	if encoder == "libx264" {
		args = append(args, "-crf", "18", "-preset", "fast")
	} else {
		args = append(args, "-cq", "18", "-preset", "slow")
	}
	args = append(args, "-c:a", "aac", "-b:a", "192k", outputFile)

	cmd := exec.Command("ffmpeg", args...)
	stderr, _ := cmd.StderrPipe()
	cmd.Start()

	reader := bufio.NewReader(stderr)
	timeRegex := regexp.MustCompile(`time=(\d{2}:\d{2}:\d{2}\.\d{2})`)

	for {
		line, err := reader.ReadString('\r')
		if err != nil {
			break
		}
		matches := timeRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			curr := parseTimeToSeconds(matches[1])
			pct := (curr / totalDur) * 100
			if pct > 100 {
				pct = 100
			}
			if pct < 0 {
				pct = 0
			}

			completed := int((pct / 100) * 40)
			rem := 40 - completed
			bar := strings.Repeat("█", completed) + strings.Repeat("░", rem)

			fmt.Printf("\r%s🌸 [%s] %.0f%% 🌸%s", pinkStart, bar, pct, reset)
		}
	}
	cmd.Wait()
	fmt.Printf("\r%-60s\n", "")
}

func getBestEncoder() string {
	cmd := exec.Command("ffmpeg", "-encoders")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Run()
	output := out.String()

	if strings.Contains(output, "h264_nvenc") {
		return "h264_nvenc"
	}
	if strings.Contains(output, "h264_amf") {
		return "h264_amf"
	}
	if strings.Contains(output, "h264_qsv") {
		return "h264_qsv"
	}
	if strings.Contains(output, "h264_videotoolbox") {
		return "h264_videotoolbox"
	}
	return "libx264"
}

func parseTimeToSeconds(t string) float64 {
	parts := strings.Split(t, ":")
	var secs float64
	if len(parts) == 3 {
		h, _ := strconv.ParseFloat(parts[0], 64)
		m, _ := strconv.ParseFloat(parts[1], 64)
		s, _ := strconv.ParseFloat(parts[2], 64)
		secs = h*3600 + m*60 + s
	} else if len(parts) == 2 {
		m, _ := strconv.ParseFloat(parts[0], 64)
		s, _ := strconv.ParseFloat(parts[1], 64)
		secs = m*60 + s
	} else {
		secs, _ = strconv.ParseFloat(t, 64)
	}
	return secs
}
