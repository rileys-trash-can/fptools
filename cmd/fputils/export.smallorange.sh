echo hi

convert ~/Pictures/Vec2/export.png \
	-background white \
	-ordered-dither o4x4 \
	-resize 792x396 \
	$@

# dither above fill
#	-ordered-dither o4x4 \
#bellow fill
# 	+opaque white \
