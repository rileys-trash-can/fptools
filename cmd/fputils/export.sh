echo hi

convert ~/Pictures/Vec2/export.png \
	-background black \
	-rotate 90 \
	-channel RGB \
	-negate \
	-resize 820x1188 \
	-ordered-dither o4x4 \
	-fill black \
	-gravity west \
	-splice 30x0 \
	$@

# dither above fill
#bellow fill
# 	+opaque white \
