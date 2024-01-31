#!/bin/sh -e
# pipe text into this script to generate a out.png

OUTFILE="out.png"
TMPFILE=""$(mktemp --suffix=.png)

echo "OUTFILE: $OUTFILE"
echo "TMPFILE: $TMPFILE"

IFS="" read input_string

rm buf.png
char="${input_string:0:1}"

convert -size 800x800 xc:black \
	$TMPFILE

# Loop through each character in the string
for ((i=0; i<${#input_string}; i++)); do

    # Get the current character
    char="${input_string:i:1}"

	figlet "$char"
	convert -size 800x800 xc:black \
		-font /usr/share/fonts/noto-cjk/NotoSansCJK-Black.ttc \
		-pointsize 900 -fill white -gravity center \
		-draw "text 0,-50 '$char'" \
 		-rotate 0 \
 		$TMPFILE
	   # -flip \
	
	convert -append $OUTFILE $TMPFILE \
		 $TMPFILE
done

convert $TMPFILE -colorspace Gray $TMPFILE

echo "cleaning tempfile"
rm "$TMPFILE"

