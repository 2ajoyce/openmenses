#!/usr/bin/env python3
"""Generate the OpenMenses app icon PNG from scratch using only stdlib."""
import struct
import zlib
import math

W, H = 1024, 1024
OUT = "mobile/ios/OpenMenses/Assets.xcassets/AppIcon.appiconset/AppIcon-1024.png"


def write_png(filename, width, height, pixels):
    def pack_row(row):
        return b"\x00" + bytes([c for rgba in row for c in rgba])

    raw = b"".join(pack_row(pixels[y]) for y in range(height))
    compressed = zlib.compress(raw, 9)

    def chunk(ctype, data):
        c = ctype + data
        return (
            struct.pack(">I", len(data))
            + c
            + struct.pack(">I", zlib.crc32(c) & 0xFFFFFFFF)
        )

    sig = b"\x89PNG\r\n\x1a\n"
    # 8-bit depth, color type 6 (RGBA)
    ihdr = chunk(b"IHDR", struct.pack(">IIBBBBB", width, height, 8, 6, 0, 0, 0))
    idat = chunk(b"IDAT", compressed)
    iend = chunk(b"IEND", b"")
    with open(filename, "wb") as f:
        f.write(sig + ihdr + idat + iend)


# Allocate pixel buffer: list of rows, each row a list of (R,G,B,A) tuples
pixels = [[(0, 0, 0, 255)] * W for _ in range(H)]

# Background gradient: #2a9d8f → #1a6b60 top-to-bottom
bg1 = (42, 157, 143)
bg2 = (26, 107, 96)


def lerp(c1, c2, t):
    return tuple(int(c1[i] + (c2[i] - c1[i]) * t) for i in range(3))


for y in range(H):
    t = y / (H - 1)
    r, g, b = lerp(bg1, bg2, t)
    for x in range(W):
        # Apply rounded rect mask (rx=220)
        dx = max(0, 220 - x, x - (W - 1 - 220))
        dy = max(0, 220 - y, y - (H - 1 - 220))
        if dx * dx + dy * dy <= 220 * 220:
            pixels[y][x] = (r, g, b, 255)
        else:
            pixels[y][x] = (255, 255, 255, 0)  # Transparent background outside the icon


def draw_circle(cx, cy, radius, color):
    r, g, b = color[:3]
    a = color[3] if len(color) > 3 else 255
    for py in range(max(0, cy - radius - 1), min(H, cy + radius + 2)):
        for px in range(max(0, cx - radius - 1), min(W, cx + radius + 2)):
            dist = math.sqrt((px - cx) ** 2 + (py - cy) ** 2)
            if dist <= radius - 0.5:
                alpha = a
            elif dist <= radius + 0.5:
                alpha = int(a * (radius + 0.5 - dist))
            else:
                continue

            # Only draw if inside the rounded rect
            bg = pixels[py][px]
            if bg[3] == 0:
                continue

            fa = alpha / 255.0
            nr = int(r * fa + bg[0] * (1 - fa))
            ng = int(g * fa + bg[1] * (1 - fa))
            nb = int(b * fa + bg[2] * (1 - fa))
            pixels[py][px] = (nr, ng, nb, 255)


# 5×5 dot grid matching SVG: centers at 180, 346, 512, 678, 844
centers = [180, 346, 512, 678, 844]
white = (255, 255, 255, int(0.65 * 255))  # semi-transparent white
red = (230, 57, 70, 255)  # #e63946 cycle start marker

for row, cy in enumerate(centers):
    for col, cx in enumerate(centers):
        if row == 1 and col == 3:
            draw_circle(cx, cy, 68, red)  # highlighted dot — slightly larger
        else:
            draw_circle(cx, cy, 52, white)

write_png(OUT, W, H, pixels)
print(f"Written: {OUT}")
