import time
import os
import psutil
from threading import Thread


def rss_mb():
    return psutil.Process(os.getpid()).memory_info().rss / (1024 * 1024)


def cpu_heavy(n):
    total = 0
    for i in range(n):
        total += (i * i) % 17
        total ^= (total << 1) & 0xFFFFFFFFFFFFFFFF
    return total


def worker(n_iters):
    cpu_heavy(n_iters)


def run_bench(n_iters=50_000_00, threads=4):
    print("\n==============================================")
    print("           PARALLEL MULTITHREAD BENCH")
    print("==============================================")
    print(f"Interpreter: {os.popen('python3 -V').read().strip()}")
    print(f"Threads:     {threads}")
    print(f"Iterations per thread: {n_iters}")
    print()

    rss_start = rss_mb()
    t0 = time.perf_counter()

    ts = [Thread(target=worker, args=(n_iters,)) for _ in range(threads)]
    for t in ts:
        t.start()
    for t in ts:
        t.join()

    t1 = time.perf_counter()
    rss_end = rss_mb()

    print("--------------- RESULTS ---------------")
    print(f"Time taken:  {t1 - t0:.3f} sec")
    print(f"RSS start:   {rss_start:.2f} MB")
    print(f"RSS end:     {rss_end:.2f} MB")
    print(f"RSS growth:  {rss_end - rss_start:.2f} MB")
    print("---------------------------------------\n")


if __name__ == "__main__":
    run_bench(n_iters=50_000_00, threads=4)
