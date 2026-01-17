#!/usr/bin/env python3
"""
Simple threaded HTTP server benchmark.
Run with: python 4.server.py [server|client|both]
"""
import sys
import os
import time
import threading
import argparse
from http.server import HTTPServer, BaseHTTPRequestHandler
from socketserver import ThreadingMixIn
from concurrent.futures import ThreadPoolExecutor
import urllib.request
import urllib.error

try:
    import psutil
    HAS_PSUTIL = True
except ImportError:
    HAS_PSUTIL = False

HOST = "127.0.0.1"
PORT = 8080


def get_rss_mb():
    if HAS_PSUTIL:
        return psutil.Process(os.getpid()).memory_info().rss / (1024 * 1024)
    return 0


class HelloHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-Type", "text/plain")
        self.end_headers()
        self.wfile.write(b"hello")

    def log_message(self, format, *args):
        pass


class ThreadedHTTPServer(ThreadingMixIn, HTTPServer):
    daemon_threads = True


def run_server():
    server = ThreadedHTTPServer((HOST, PORT), HelloHandler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nShutting down...")
        server.shutdown()


def make_request(url):
    try:
        with urllib.request.urlopen(url, timeout=10) as response:
            return response.status == 200
    except Exception:
        return False


def run_load_test(num_requests=1000, concurrency=50):
    url = f"http://{HOST}:{PORT}/"

    # Warm up
    for _ in range(10):
        make_request(url)

    rss_before = get_rss_mb()

    start_time = time.perf_counter()
    with ThreadPoolExecutor(max_workers=concurrency) as executor:
        results = list(executor.map(lambda _: make_request(url), range(num_requests)))
    elapsed = time.perf_counter() - start_time

    rss_after = get_rss_mb()
    avg_latency = (elapsed / num_requests) * 1000

    print(f"workers: {concurrency}")
    print(f"reqs: {num_requests}")
    print(f"latency: {avg_latency:.2f}ms")
    print(f"rss_delta: {rss_after - rss_before:.1f}MiB")


def run_both(num_requests=1000, concurrency=50):
    """Run server in background thread and then run load test"""
    server = ThreadedHTTPServer((HOST, PORT), HelloHandler)
    server_thread = threading.Thread(target=server.serve_forever, daemon=True)
    server_thread.start()
    time.sleep(0.3)

    run_load_test(num_requests, concurrency)

    server.shutdown()


def main():
    parser = argparse.ArgumentParser(description="HTTP Server Benchmark")
    parser.add_argument("mode", nargs="?", default="both",
                        choices=["server", "client", "both"],
                        help="Run mode: server, client, or both (default: both)")
    parser.add_argument("-n", "--requests", type=int, default=1000,
                        help="Number of requests (default: 1000)")
    parser.add_argument("-c", "--concurrency", type=int, default=50,
                        help="Concurrency level (default: 50)")

    args = parser.parse_args()

    if args.mode == "server":
        run_server()
    elif args.mode == "client":
        run_load_test(args.requests, args.concurrency)
    else:
        run_both(args.requests, args.concurrency)


if __name__ == "__main__":
    main()
