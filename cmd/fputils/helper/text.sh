#!/bin/sh -e
# pipe text into this script to directly print letter by letter

TMPFILE=$(mktemp --suffix=.png)
FPUTILS="$(dirname $0)/.."

echo "TMPFILE: $TMPFILE"
echo "FPUTILS: $FPUTILS"

IFS="" read input_string

# Loop through each character in the string
for ((i=0; i<${#input_string}; i++)); do
    # Get the current character
    char="${input_string:i:1}"

	figlet "$char"
	convert -size 800x800 xc:black \
		-font /usr/share/fonts/noto-cjk/NotoSansCJK-Black.ttc \
		-pointsize 1000 -fill white -gravity center \
		-draw "text 0,0 '$char'" \
	    $TMPFILE
	    #-rotate 270 \
	    #-flip \

    go run $FPUTILS -beep=false -count=1 printprbuf $TMPFILE
done

echo "cleaning tempfile"
rm "$TMPFILE"
