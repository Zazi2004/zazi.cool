#!/usr/bin/env python3
"""Regenerates the app icons in web/public (standard library only).

Draws the favicon's progress ring, supersampled 3x for smooth edges.
Usage: python3 scripts/icons.py   (from apps/habits)
"""
import math
import struct
import zlib

BG = (109, 90, 230)
WHITE = (255, 255, 255)
FAINT = tuple(round(b * 0.7 + w * 0.3) for b, w in zip(BG, WHITE))
SS = 3


def render(size, full_bleed, ring_scale):
    n = size * SS
    c = n / 2
    radius = n * ring_scale
    half_stroke = radius * 0.44 / 2
    corner = 0 if full_bleed else n * 0.25
    rows = []
    for y in range(size):
        row = bytearray([0])
        for x in range(size):
            r = g = b = hits = 0
            for sy in range(SS):
                for sx in range(SS):
                    px, py = x * SS + sx + 0.5, y * SS + sy + 0.5
                    # Rounded-square coverage.
                    dx = max(abs(px - c) - (n / 2 - corner), 0)
                    dy = max(abs(py - c) - (n / 2 - corner), 0)
                    if math.hypot(dx, dy) > corner:
                        continue
                    color = BG
                    if abs(math.hypot(px - c, py - c) - radius) <= half_stroke:
                        # Clockwise from 12 o'clock: the first 270 degrees are "done".
                        angle = (math.degrees(math.atan2(py - c, px - c)) + 90) % 360
                        color = WHITE if angle <= 270 else FAINT
                    r += color[0]
                    g += color[1]
                    b += color[2]
                    hits += 1
            if hits == 0:
                row += bytes(4)
            else:
                row += bytes([r // hits, g // hits, b // hits, 255 * hits // (SS * SS)])
        rows.append(bytes(row))
    return b"".join(rows)


def write_png(path, size, full_bleed, ring_scale):
    def chunk(tag, data):
        body = tag + data
        return struct.pack(">I", len(data)) + body + struct.pack(">I", zlib.crc32(body))

    data = (
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", struct.pack(">IIBBBBB", size, size, 8, 6, 0, 0, 0))
        + chunk(b"IDAT", zlib.compress(render(size, full_bleed, ring_scale), 9))
        + chunk(b"IEND", b"")
    )
    with open(path, "wb") as f:
        f.write(data)


# The maskable icon is full-bleed and keeps its content inside the central safe zone.
write_png("web/public/icon-192.png", 192, False, 0.25)
write_png("web/public/icon-512.png", 512, False, 0.25)
write_png("web/public/icon-maskable-512.png", 512, True, 0.2)
write_png("web/public/apple-touch-icon.png", 180, True, 0.22)
