#!/usr/bin/env python3
import sys
import sysconfig
import time
from threading import Thread
from multiprocessing import Process


# Compute-intensive task: Iterative Fibonacci calculation
def compute_fibonacci(n):
    """Compute Fibonacci number for a given n iteratively."""
    a, b = 0, 1
    for _ in range(n):
        a, b = b, a + b
    return a


# Single-threaded task execution
def run_single_threaded(nums):
    for num in nums:
        compute_fibonacci(num)


# Multi-threaded task execution
def run_multi_threaded(nums):
    threads = [Thread(target=compute_fibonacci, args=(num,)) for num in nums]
    for thread in threads:
        thread.start()
    for thread in threads:
        thread.join()


# Multi-processing task execution
def run_multi_processing(nums):
    processes = [Process(target=compute_fibonacci, args=(num,)) for num in nums]
    for process in processes:
        process.start()
    for process in processes:
        process.join()


# Main execution function
def main():
    # Check Python version and GIL status for Python 3.13+
    print(f"Python Version: {sys.version}")

    py_version = float(".".join(sys.version.split()[0].split(".")[:2]))
    status = sysconfig.get_config_var("Py_GIL_DISABLED")

    if py_version >= 3.13:
        status = sys._is_gil_enabled()

    if status is None:
        print("GIL cannot be disabled for Python <= 3.12")
    elif status == 0:
        print("GIL is currently disabled")
    elif status == 1:
        print("GIL is currently active")

    # Run tasks on the same input size for comparison
    nums = [300000] * 10

    print("\nRunning Single-Threaded Task:")
    start = time.perf_counter()
    run_single_threaded(nums)
    print(f"run_single_threaded took {time.perf_counter() - start:.4f} seconds.")

    print("\nRunning Multi-Threaded Task:")
    start = time.perf_counter()
    run_multi_threaded(nums)
    print(f"run_multi_threaded took {time.perf_counter() - start:.4f} seconds.")

    print("\nRunning Multi-Processing Task:")
    start = time.perf_counter()
    run_multi_processing(nums)
    print(f"run_multi_processing took {time.perf_counter() - start:.4f} seconds.")


if __name__ == "__main__":
    main()
