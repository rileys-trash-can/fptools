echo hi

convert ~/Pictures/Vec2/export.png \
	-background white \
	-ordered-dither o4x4 \
	-resize 812x1200 \
	$@

# dither above fill
#bellow fill
# 	+opaque white \

#	-rotate 90 \
#	-channel RGB \
#	-negate \
