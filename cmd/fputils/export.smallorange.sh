echo hi

convert ~/Pictures/Vec2/export.png \
	-background black \
	-channel RGB \
	-negate \
	-resize 792x396 \
	-fill black \
	-gravity west \
	-depth 1 \
	-splice 30x0 \
	$@

# dither above fill
#	-ordered-dither o4x4 \
#bellow fill
# 	+opaque white \
