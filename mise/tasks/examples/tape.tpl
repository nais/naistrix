Set Shell bash
Set FontFamily "DejaVu Sans Mono"
Set FontSize 20
Set Width 1200
Set Height 600
Set Framerate 30
Set TypingSpeed 40ms
Set CursorBlink false

Output {{ .WorkDir }}/recording.gif

# Use the binary built for this recording.
Hide
Type "alias {{ .Name }}='HOME=/tmp/naistrix-vhs XDG_CONFIG_HOME=/tmp/naistrix-vhs ./{{ .WorkDir }}/{{ .Name }}'; clear"
Enter
Wait
Show

{{ .Steps }}

# Capture the settled terminal screen. Earlier commands may still be drawing.
Output {{ .WorkDir }}/recording.ascii
