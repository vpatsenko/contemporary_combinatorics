#!/usr/bin/env python3
"""
Mandelbrot set benchmark - multithreaded
"""
import sys
import os
import time
from concurrent.futures import ThreadPoolExecutor

try:
    import psutil
    HAS_PSUTIL = True
except ImportError:
    HAS_PSUTIL = False

SIZE = 4000
MAX_ITER = 50

def get_rss_mb():
    if HAS_PSUTIL:
        return psutil.Process(os.getpid()).memory_info().rss / (1024 * 1024)
    return 0

def compute_row(y):
    row = bytearray((SIZE + 7) // 8)
    c1 = 2.0 / SIZE
    ci = y * c1 - 1.0

    for x in range(SIZE):
        cr = x * c1 - 1.5
        zr, zi = cr, ci

        inside = True
        for _ in range(MAX_ITER):
            zr2, zi2 = zr * zr, zi * zi
            if zr2 + zi2 > 4.0:
                inside = False
                break
            zi = 2.0 * zr * zi + ci
            zr = zr2 - zi2 + cr

        if inside:
            row[x // 8] |= (128 >> (x % 8))

    return row

def mandelbrot_sequential():
    result = []
    for y in range(SIZE):
        result.append(compute_row(y))
    return result

def mandelbrot_threaded(workers=None):
    if workers is None:
        workers = os.cpu_count() or 4

    with ThreadPoolExecutor(max_workers=workers) as executor:
        result = list(executor.map(compute_row, range(SIZE)))
    return result

def benchmark(name, func):
    rss_before = get_rss_mb()
    start = time.perf_counter()

    result = func()

    elapsed = time.perf_counter() - start
    rss_after = get_rss_mb()

    print(f"{name}:")
    print(f"  time: {elapsed*1000:.0f}ms")
    print(f"  rss_delta: {rss_after - rss_before:.1f}MiB")
    return result

def main():
    print(f"Mandelbrot {SIZE}x{SIZE}, max_iter={MAX_ITER}")

    py_version = float(".".join(sys.version.split()[0].split(".")[:2]))
    if py_version >= 3.13:
        gil = "disabled" if not sys._is_gil_enabled() else "enabled"
        print(f"GIL: {gil}")
    print()

    benchmark("sequential", mandelbrot_sequential)
    print()
    benchmark("threaded", mandelbrot_threaded)

if __name__ == "__main__":
    main()
