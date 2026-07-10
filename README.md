# goVideoTrimmer
This Go file is a Terminal User Interface (TUI) application for trimming video files. It is designed with a highly stylized "Japanese Blossom" theme (featuring pink and purple color palettes) and acts as a user-friendly wrapper around the popular ffmpeg command-line tool.
Here is a breakdown of what the file does under the hood:

Interactive File Browser: It uses the tview and tcell libraries to render a directory tree in your terminal. You can navigate through your folders and drives (with explicit support for Windows drive letters) to find video files (.mp4, .mkv, .avi, etc.).

Video Metadata Extraction: Once you select a video, it invisibly runs ffprobe to determine the exact duration of the video to populate the default "End Time" field.

Smart Time Adjustment: It generates a form popup where you can input the start and end times for your trim. You can manually type the timestamp (HH:MM:SS) or use the Up/Down arrow keys to increment/decrement the seconds safely.

Hardware-Accelerated Processing: Before trimming, it checks your system for supported hardware encoders (Nvidia nvenc, AMD amf, Intel qsv, or Apple videotoolbox). It automatically selects the best available one to make the trimming process as fast as possible, falling back to standard CPU encoding (libx264) if needed.

Visual Progress Feedback: While ffmpeg processes the video, the app intercepts its output and calculates a real-time progress percentage, displaying a custom pink progress bar (████░░░) directly in the terminal.

Safe File Handling: It calculates a safe output filename (e.g., video_trimed.mp4, video_trimed01.mp4) so it never accidentally overwrites your original file.

<img width="1109" height="580" alt="image" src="https://github.com/user-attachments/assets/c91c9a13-3bf4-4d04-8ad5-15b346defe1e" />

1 . Install the required dependencies:

Bash
go get [github.com/gdamore/tcell/v2](https://github.com/gdamore/tcell/v2)
go get [github.com/rivo/tview](https://github.com/rivo/tview)


2. Run or Build the application:

Bash
go run trimvideo.go

or 

go build -o trimvideo.exe trimvideo.go



🎮 Usage
---------------------------------------------------------------------------
Navigation: Use the Arrow Keys to navigate the directory tree.

1.Select: Press Enter to open a directory or select a video file.

2.Go Back: Press Backspace to go up one directory level.

3.Trim Menu: Once a video is selected, a form will appear.

4.Input the Start Time and End Time (Format: HH:MM:SS).

5.Use Up/Down arrows on the input fields to adjust seconds dynamically.

6.Press Tab to switch between fields.

7.Select Trim to start processing.

8.Quit: Press Escape at any time to exit the application.
"""
