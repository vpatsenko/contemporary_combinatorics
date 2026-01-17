#!/usr/bin/env python3
import sys
import sysconfig
import time
import os
import gc
from threading import Thread, Lock, Event
from multiprocessing import Process, Value
import mmap
import ctypes

try:
    import psutil
    HAS_PSUTIL = True
except ImportError:
    HAS_PSUTIL = False
    print("Warning: psutil not installed.")
    sys.exit(1)


def get_rss_mb():
    process = psutil.Process(os.getpid())
    return process.memory_info().rss / (1024 * 1024)


def get_total_rss_with_children_mb():
    process = psutil.Process(os.getpid())
    total = process.memory_info().rss
    for child in process.children(recursive=True):
        try:
            total += child.memory_info().rss
        except psutil.NoSuchProcess:
            pass
    return total / (1024 * 1024)


class PeakMemoryTracker:
    def __init__(self, interval=0.01, include_children=False):
        self.interval = interval
        self.include_children = include_children
        self.peak_rss = 0
        self._stop_event = Event()
        self._thread = None

    def _sample_loop(self):
        while not self._stop_event.is_set():
            if self.include_children:
                current = get_total_rss_with_children_mb()
            else:
                current = get_rss_mb()
            if current > self.peak_rss:
                self.peak_rss = current
            time.sleep(self.interval)

    def start(self):
        self._stop_event.clear()
        self.peak_rss = get_rss_mb()
        self._thread = Thread(target=self._sample_loop, daemon=True)
        self._thread.start()

    def stop(self):
        self._stop_event.set()
        if self._thread:
            self._thread.join(timeout=1)
        return self.peak_rss

def memory_intensive_task(size_mb=50):
    # Allocate
    data = bytearray(size_mb * 1024 * 1024)

    # Touch each 4KB page so the OS commits the memory and RSS reflects it
    page = mmap.PAGESIZE  # usually 4096
    for i in range(0, len(data), page):
        data[i] = (data[i] + 1) & 0xFF

    # Keep it alive briefly so the peak sampler has time to see it
    time.sleep(0.2)

    # Return something that depends on the buffer (prevents "too smart" elimination)
    return data[0] + data[len(data) // 2] + data[-1]



def memory_task_for_process(size_mb):
    memory_intensive_task(size_mb)


def run_single_threaded(num_tasks, size_mb):
    for _ in range(num_tasks):
        memory_intensive_task(size_mb)
        gc.collect()


def run_multi_threaded(num_tasks, size_mb):
    threads = []
    for _ in range(num_tasks):
        t = Thread(target=memory_intensive_task, args=(size_mb,))
        threads.append(t)

    for t in threads:
        t.start()

    for t in threads:
        t.join()


def run_multi_processing(num_tasks, size_mb):
    processes = []

    for _ in range(num_tasks):
        p = Process(target=memory_task_for_process, args=(size_mb,))
        processes.append(p)

    for p in processes:
        p.start()

    for p in processes:
        p.join()


def measure_memory(name, func, num_tasks, size_mb, include_children=False):
    gc.collect()
    time.sleep(0.05)

    rss_before = get_rss_mb()

    tracker = PeakMemoryTracker(interval=0.01, include_children=include_children)
    tracker.start()

    start_time = time.perf_counter()
    func(num_tasks, size_mb)
    end_time = time.perf_counter()

    peak_rss = tracker.stop()
    rss_after = get_rss_mb()

    print(f"  Time: {end_time - start_time:.4f} seconds")
    print(f"  RSS before: {rss_before:.2f} MB")
    print(f"  RSS peak: {peak_rss:.2f} MB")
    print(f"  RSS after: {rss_after:.2f} MB")
    print(f"  RSS delta (peak - before): {peak_rss - rss_before:.2f} MB")

    return peak_rss


def main():
    print(f"Python Version: {sys.version}")

    py_version = float(".".join(sys.version.split()[0].split(".")[:2]))
    status = sysconfig.get_config_var("Py_GIL_DISABLED")

    if py_version >= 3.13:
        status = sys._is_gil_enabled()

    if status is None:
        print("GIL cannot be disabled for Python <= 3.12")
    elif status == 0:
        print("GIL is currently DISABLED (free-threading)")
    elif status == 1:
        print("GIL is currently ACTIVE")

    print("\n" + "=" * 60)
    print("MEMORY BENCHMARK (RSS-based)")
    print("=" * 60)

    num_tasks = 4
    size_mb = 50

    print(f"\nConfiguration:")
    print(f"  Number of tasks: {num_tasks}")
    print(f"  Memory per task: ~{size_mb} MB")
    print(f"  Expected peak (sequential): ~{size_mb} MB")
    print(f"  Expected peak (parallel): ~{num_tasks * size_mb} MB")

    gc.collect()
    time.sleep(0.1)
    baseline_rss = get_rss_mb()
    print(f"\nBaseline RSS: {baseline_rss:.2f} MB")

    print("\n" + "-" * 60)
    print("SINGLE-THREADED (Sequential)")
    print("-" * 60)
    print("Note: Memory reused between tasks, GC runs between iterations")
    measure_memory("single_threaded", run_single_threaded, num_tasks, size_mb)

    gc.collect()
    time.sleep(0.1)

    print("\n" + "-" * 60)
    print("MULTI-THREADED (Shared Memory Space)")
    print("-" * 60)
    print("Note: All threads share memory space, run concurrently")
    measure_memory("multi_threaded", run_multi_threaded, num_tasks, size_mb)

    gc.collect()
    time.sleep(0.1)

    print("\n" + "-" * 60)
    print("MULTI-PROCESSING (Separate Memory Spaces)")
    print("-" * 60)
    print("Note: Each process has separate memory + interpreter overhead")
    measure_memory("multi_processing", run_multi_processing, num_tasks, size_mb,
                   include_children=True)

    print("\n" + "=" * 60)
    print("SUMMARY")
    print("=" * 60)
    print(f"""
Expected results:
- Single-threaded: ~{size_mb} MB peak (one task at a time, GC between)
- Multi-threaded:
  * With GIL: ~{size_mb}-{size_mb*2} MB (threads serialize due to GIL)
  * No GIL: ~{num_tasks * size_mb} MB (all threads run in parallel)
- Multi-processing: ~{num_tasks * size_mb} MB + {num_tasks}x interpreter overhead
""")


if __name__ == "__main__":
    main()
